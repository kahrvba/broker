package main

import (
	"context"
	"encoding/json"
	"log"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

var db *pgxpool.Pool

func main() {

	connStr := "postgres://telemetry:9090@localhost:5432/inverterdb?sslmode=disable"

	pool, err := pgxpool.New(context.Background(), connStr)
	if err != nil {
		log.Fatal(err)
	}
	db = pool
	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/" {
			http.NotFound(w, r)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]string{
			"api-status":	"ok",
		})
	})

	http.HandleFunc("/api/inverters", getInverters)
	http.HandleFunc("/api/inverters/", inverterRouter)

	log.Println("API running on :8080")
	log.Fatal(http.ListenAndServe("127.0.0.1:8080", nil))
}

/* ========================
   ROUTER
======================== */

func inverterRouter(w http.ResponseWriter, r *http.Request) {

	path := strings.TrimPrefix(r.URL.Path, "/api/inverters/")
	parts := strings.Split(path, "/")

	if len(parts) < 2 {
		http.Error(w, "invalid path", 400)
		return
	}

	serial := parts[0]
	action := parts[1]

	inverterID, err := getInverterIDBySerial(serial)
	if err != nil {
		http.Error(w, "inverter not found", 404)
		return
	}

	switch action {
	case "live":
		getLive(w, inverterID)
	case "qpigs":
		getQpigs(w, inverterID)
	case "history":
		getHistory(w, r, inverterID)
	case "faults":
		getFaults(w, inverterID)
	default:
		http.NotFound(w, r)
	}
}

/* ========================
   SERIAL → ID
======================== */

func getInverterIDBySerial(serial string) (int64, error) {
	var id int64
	err := db.QueryRow(context.Background(),
		`SELECT id FROM inverters WHERE serial=$1`,
		serial,
	).Scan(&id)
	return id, err
}

/* ========================
   LIST INVERTERS
======================== */

func getInverters(w http.ResponseWriter, r *http.Request) {

	rows, err := db.Query(context.Background(),
		`SELECT serial FROM inverters`)
	if err != nil {
		http.Error(w, err.Error(), 500)
		return
	}
	defer rows.Close()

	var serials []string

	for rows.Next() {
		var s string
		rows.Scan(&s)
		serials = append(serials, s)
	}

	json.NewEncoder(w).Encode(serials)
}

/* ========================
   LIVE
======================== */

func getLive(w http.ResponseWriter, id int64) {

	type Live struct {
		Time             time.Time `json:"time"`
		ActivePower      int       `json:"ac_output_active_power"`
		BatteryVoltage   float64   `json:"battery_voltage"`
		PvInputPower     int       `json:"pv_input_power"`
		Pv2ChargingPower int       `json:"pv2_charging_power"`
		IsLoadOn         bool      `json:"is_load_on"`
		IsChargingOn     bool      `json:"is_charging_on"`
		IsSccChargingOn  bool      `json:"is_scc_charging_on"`
		IsAcChargingOn   bool      `json:"is_ac_charging_on"`
		IsSwitchedOn     bool      `json:"is_switched_on"`
	}

	var result Live

	err := db.QueryRow(context.Background(),
		`SELECT time,
		        ac_output_active_power,
		        battery_voltage,
		        pv_input_power,
		        pv2_charging_power,
		        is_load_on,
		        is_charging_on,
		        is_scc_charging_on,
		        is_ac_charging_on,
		        is_switched_on
		 FROM inverter_latest
		 WHERE inverter_id=$1`,
		id,
	).Scan(
		&result.Time,
		&result.ActivePower,
		&result.BatteryVoltage,
		&result.PvInputPower,
		&result.Pv2ChargingPower,
		&result.IsLoadOn,
		&result.IsChargingOn,
		&result.IsSccChargingOn,
		&result.IsAcChargingOn,
		&result.IsSwitchedOn,
	)

	if err != nil {
		http.Error(w, err.Error(), 404)
		return
	}

	json.NewEncoder(w).Encode(result)
}

/* ========================
   QPIGS FULL SNAPSHOT
======================== */

func getQpigs(w http.ResponseWriter, id int64) {

	row := db.QueryRow(context.Background(),
		`SELECT *
		 FROM telemetry
		 WHERE inverter_id=$1
		 ORDER BY time DESC
		 LIMIT 1`,
		id,
	)

	rows, err := db.Query(context.Background(),
		`SELECT column_name
		 FROM information_schema.columns
		 WHERE table_name='telemetry'
		 ORDER BY ordinal_position`)
	if err != nil {
		http.Error(w, err.Error(), 500)
		return
	}
	defer rows.Close()

	var columns []string
	for rows.Next() {
		var col string
		rows.Scan(&col)
		columns = append(columns, col)
	}

	values := make([]interface{}, len(columns))
	valuePtrs := make([]interface{}, len(columns))
	for i := range values {
		valuePtrs[i] = &values[i]
	}

	if err := row.Scan(valuePtrs...); err != nil {
		http.Error(w, err.Error(), 404)
		return
	}

	result := make(map[string]interface{})
	for i, col := range columns {
		result[col] = values[i]
	}

	json.NewEncoder(w).Encode(result)
}

/* ========================
   HISTORY
======================== */

func getHistory(w http.ResponseWriter, r *http.Request, id int64) {

	limit := 50
	if l := r.URL.Query().Get("limit"); l != "" {
		if parsed, err := strconv.Atoi(l); err == nil {
			limit = parsed
		}
	}

	rows, err := db.Query(context.Background(),
		`SELECT time,
		        ac_output_active_power,
		        battery_voltage,
		        pv_input_power
		 FROM telemetry
		 WHERE inverter_id=$1
		 ORDER BY time DESC
		 LIMIT $2`,
		id, limit)
	if err != nil {
		http.Error(w, err.Error(), 500)
		return
	}
	defer rows.Close()

	type History struct {
		Time           time.Time `json:"time"`
		Power          int       `json:"power"`
		BatteryVoltage float64   `json:"battery_voltage"`
		PvInputPower   int       `json:"pv_input_power"`
	}

	var result []History

	for rows.Next() {
		var h History
		if err := rows.Scan(&h.Time, &h.Power, &h.BatteryVoltage, &h.PvInputPower); err != nil {
			continue
		}
		result = append(result, h)
	}

	json.NewEncoder(w).Encode(result)
}

/* ========================
   FAULTS
======================== */

func getFaults(w http.ResponseWriter, id int64) {

	type Faults struct {
		Time                     time.Time `json:"time"`
		InverterFault            bool      `json:"inverter_fault"`
		BusOverFault             bool      `json:"bus_over_fault"`
		BusUnderFault            bool      `json:"bus_under_fault"`
		BusSoftFailFault         bool      `json:"bus_soft_fail_fault"`
		LineFailWarning          bool      `json:"line_fail_warning"`
		OpvShortWarning          bool      `json:"opv_short_warning"`
		InverterVoltageTooLow    bool      `json:"inverter_voltage_too_low_fault"`
		InverterVoltageTooHigh   bool      `json:"inverter_voltage_too_high_fault"`
		OverTemperatureFault     bool      `json:"over_temperature_fault"`
		FanLockedFault           bool      `json:"fan_locked_fault"`
		BatteryVoltageTooHigh    bool      `json:"battery_voltage_too_high_fault"`
		BatteryLowAlarmWarning   bool      `json:"battery_low_alarm_warning"`
		BatteryUnderShutdown     bool      `json:"battery_under_shutdown_warning"`
		OverloadFault            bool      `json:"overload_fault"`
		EepromFault              bool      `json:"eeprom_fault"`
		InverterOverCurrentFault bool      `json:"inverter_over_current_fault"`
		InverterSoftFailFault    bool      `json:"inverter_soft_fail_fault"`
		SelfTestFailFault        bool      `json:"self_test_fail_fault"`
		OpDcVoltageOverFault     bool      `json:"op_dc_voltage_over_fault"`
		BatteryOpenFault         bool      `json:"battery_open_fault"`
		CurrentSensorFailFault   bool      `json:"current_sensor_fail_fault"`
		BatteryShortFault        bool      `json:"battery_short_fault"`
		PowerLimitWarning        bool      `json:"power_limit_warning"`
		PvVoltageHighWarning     bool      `json:"pv_voltage_high_warning"`
		MpptOverloadFault        bool      `json:"mppt_overload_fault"`
		MpptOverloadWarning      bool      `json:"mppt_overload_warning"`
		Reserved                 bool      `json:"reserved"`
	}

	var f Faults

	err := db.QueryRow(context.Background(),
		`SELECT time,
		        inverter_fault,
		        bus_over_fault,
		        bus_under_fault,
		        bus_soft_fail_fault,
		        line_fail_warning,
		        opv_short_warning,
		        inverter_voltage_too_low_fault,
		        inverter_voltage_too_high_fault,
		        over_temperature_fault,
		        fan_locked_fault,
		        battery_voltage_too_high_fault,
		        battery_low_alarm_warning,
		        battery_under_shutdown_warning,
		        overload_fault,
		        eeprom_fault,
		        inverter_over_current_fault,
		        inverter_soft_fail_fault,
		        self_test_fail_fault,
		        op_dc_voltage_over_fault,
		        battery_open_fault,
		        current_sensor_fail_fault,
		        battery_short_fault,
		        power_limit_warning,
		        pv_voltage_high_warning,
		        mppt_overload_fault,
		        mppt_overload_warning,
		        reserved
		 FROM inverter_faults
		 WHERE inverter_id=$1`,
		id,
	).Scan(
		&f.Time,
		&f.InverterFault,
		&f.BusOverFault,
		&f.BusUnderFault,
		&f.BusSoftFailFault,
		&f.LineFailWarning,
		&f.OpvShortWarning,
		&f.InverterVoltageTooLow,
		&f.InverterVoltageTooHigh,
		&f.OverTemperatureFault,
		&f.FanLockedFault,
		&f.BatteryVoltageTooHigh,
		&f.BatteryLowAlarmWarning,
		&f.BatteryUnderShutdown,
		&f.OverloadFault,
		&f.EepromFault,
		&f.InverterOverCurrentFault,
		&f.InverterSoftFailFault,
		&f.SelfTestFailFault,
		&f.OpDcVoltageOverFault,
		&f.BatteryOpenFault,
		&f.CurrentSensorFailFault,
		&f.BatteryShortFault,
		&f.PowerLimitWarning,
		&f.PvVoltageHighWarning,
		&f.MpptOverloadFault,
		&f.MpptOverloadWarning,
		&f.Reserved,
	)

	if err != nil {
		http.Error(w, err.Error(), 404)
		return
	}

	json.NewEncoder(w).Encode(f)
}
