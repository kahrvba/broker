package main

import (
	"context"
	"encoding/json"
	"log"
	"sync"
	"time"

	mqtt "github.com/eclipse/paho.mqtt.golang"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

var (
	dbPool *pgxpool.Pool

	fixedSerial = "00202507001160"

	serialCache = make(map[string]int64)
	cacheMutex  sync.RWMutex

	telemetryBatch []TelemetryInsert
	batchMutex     sync.Mutex
	batchSize      = 300
	flushInterval  = 2 * time.Second
)

/* ========================
   JSON STRUCTS
======================== */

type QpigsPayload map[string]QpigsData
type QpigsData struct {
	AcInputVoltage                   float64 `json:"ac_input_voltage"`
	AcInputFrequency                 float64 `json:"ac_input_frequency"`
	AcOutputVoltage                  float64 `json:"ac_output_voltage"`
	AcOutputFrequency                float64 `json:"ac_output_frequency"`
	AcOutputApparentPower            int     `json:"ac_output_apparent_power"`
	AcOutputActivePower              int     `json:"ac_output_active_power"`
	AcOutputLoad                     int     `json:"ac_output_load"`
	BusVoltage                       int     `json:"bus_voltage"`
	BatteryVoltage                   float64 `json:"battery_voltage"`
	BatteryChargingCurrent           int     `json:"battery_charging_current"`
	BatteryCapacity                  int     `json:"battery_capacity"`
	InverterHeatSinkTemperature      int     `json:"inverter_heat_sink_temperature"`
	PvInputCurrentForBattery         float64 `json:"pv_input_current_for_battery"`
	PvInputVoltage                   float64 `json:"pv_input_voltage"`
	BatteryVoltageFromScc            float64 `json:"battery_voltage_from_scc"`
	BatteryDischargeCurrent          int     `json:"battery_discharge_current"`
	IsSbuPriorityVersionAdded        int     `json:"is_sbu_priority_version_added"`
	IsConfigurationChanged           int     `json:"is_configuration_changed"`
	IsSccFirmwareUpdated             int     `json:"is_scc_firmware_updated"`
	IsLoadOn                         int     `json:"is_load_on"`
	IsChargingOn                     int     `json:"is_charging_on"`
	IsSccChargingOn                  int     `json:"is_scc_charging_on"`
	IsAcChargingOn                   int     `json:"is_ac_charging_on"`
	IsChargingToFloat                int     `json:"is_charging_to_float"`
	IsSwitchedOn                     int     `json:"is_switched_on"`
	IsReserved                       int     `json:"is_reserved"`
	Rsv1                             int     `json:"rsv1"`
	Rsv2                             int     `json:"rsv2"`
	PvInputPower                     int     `json:"pv_input_power"`
}

type Qpigs2Payload map[string]Qpigs2Data
type Qpigs2Data struct {
	Pv2InputCurrent  float64 `json:"pv2_input_current"`
	Pv2InputVoltage  float64 `json:"pv2_input_voltage"`
	Pv2ChargingPower int     `json:"pv2_Charging_Power"`
}

type QpiwsPayload map[string]QpiwsData
type QpiwsData struct {
	InverterFault                 int `json:"inverter_fault"`
	BusOverFault                  int `json:"bus_over_fault"`
	BusUnderFault                 int `json:"bus_under_fault"`
	BusSoftFailFault              int `json:"bus_soft_fail_fault"`
	LineFailWarning               int `json:"line_fail_warning"`
	OpvShortWarning               int `json:"opv_short_warning"`
	InverterVoltageTooLowFault    int `json:"inverter_voltage_too_low_fault"`
	InverterVoltageTooHighFault   int `json:"inverter_voltage_too_high_fault"`
	OverTemperatureFault          int `json:"over_temperature_fault"`
	FanLockedFault                int `json:"fan_locked_fault"`
	BatteryVoltageTooHighFault    int `json:"battery_voltage_too_high_fault"`
	BatteryLowAlarmWarning        int `json:"battery_low_alarm_warning"`
	BatteryUnderShutdownWarning   int `json:"battery_under_shutdown_warning"`
	OverloadFault                 int `json:"overload_fault"`
	EepromFault                   int `json:"eeprom_fault"`
	InverterOverCurrentFault      int `json:"inverter_over_current_fault"`
	InverterSoftFailFault         int `json:"inverter_soft_fail_fault"`
	SelfTestFailFault             int `json:"self_test_fail_fault"`
	OpDcVoltageOverFault          int `json:"op_dc_voltage_over_fault"`
	BatteryOpenFault              int `json:"battery_open_fault"`
	CurrentSensorFailFault        int `json:"current_sensor_fail_fault"`
	BatteryShortFault             int `json:"battery_short_fault"`
	PowerLimitWarning             int `json:"power_limit_warning"`
	PvVoltageHighWarning          int `json:"pv_voltage_high_warning"`
	MpptOverloadFault             int `json:"mppt_overload_fault"`
	MpptOverloadWarning           int `json:"mppt_overload_warning"`
	Reserved                      int `json:"reserved"`
}

/* ========================
   TELEMETRY STRUCT
======================== */

type TelemetryInsert struct {
	Time       time.Time
	InverterID int64
	QpigsData
	Pv2InputCurrent  float64
	Pv2InputVoltage  float64
	Pv2ChargingPower int
}

/* ========================
   MAIN
======================== */

func main() {
	ctx := context.Background()

	connStr := "postgres://telemetry:9090@localhost:5432/inverterdb?sslmode=disable"

	pool, err := pgxpool.New(ctx, connStr)
	if err != nil {
		log.Fatal(err)
	}
	dbPool = pool
	defer dbPool.Close()

	go batchFlusher()

	opts := mqtt.NewClientOptions().
		AddBroker("tcp://localhost:1883").
		SetClientID("telemetry-collector")

	client := mqtt.NewClient(opts)

	if token := client.Connect(); token.Wait() && token.Error() != nil {
		log.Fatal(token.Error())
	}

	client.Subscribe("mpp/output/"+fixedSerial+"/#", 0, messageHandler)

	log.Println("Collector running...")
	select {}
}

/* ========================
   SERIAL LOOKUP
======================== */

func getInverterID(ctx context.Context, serial string) (int64, error) {
	cacheMutex.RLock()
	if id, ok := serialCache[serial]; ok {
		cacheMutex.RUnlock()
		return id, nil
	}
	cacheMutex.RUnlock()

	var id int64
	err := dbPool.QueryRow(ctx,
		`INSERT INTO inverters(serial)
		 VALUES($1)
		 ON CONFLICT (serial) DO UPDATE SET serial = EXCLUDED.serial
		 RETURNING id`, serial).Scan(&id)
	if err != nil {
		return 0, err
	}

	cacheMutex.Lock()
	serialCache[serial] = id
	cacheMutex.Unlock()

	return id, nil
}

/* ========================
   MESSAGE HANDLER
======================== */

func messageHandler(client mqtt.Client, msg mqtt.Message) {
	ctx := context.Background()
	id, err := getInverterID(ctx, fixedSerial)
	if err != nil {
		return
	}

	switch msg.Topic() {

	case "mpp/output/" + fixedSerial + "/qpigs":
		var payload QpigsData
		if err := json.Unmarshal(msg.Payload(), &payload); err != nil {
			return
		}
		addQpigsToBatch(id, payload)
		updateLatest(ctx, id, payload)

	case "mpp/output/" + fixedSerial + "/qpigs2":
		var payload Qpigs2Data
		if err := json.Unmarshal(msg.Payload(), &payload); err != nil {
			return
		}
		addQpigs2ToBatch(id, payload)
		updateLatestPv2(ctx, id, payload)

	case "mpp/output/" + fixedSerial + "/qpiws":
		var payload QpiwsData
		if err := json.Unmarshal(msg.Payload(), &payload); err != nil {
			return
		}
		updateFaults(ctx, id, payload)
	}
}

/* ========================
   BATCH MANAGEMENT
======================== */

func addQpigsToBatch(id int64, d QpigsData) {
	row := TelemetryInsert{
		Time:       time.Now(),
		InverterID: id,
		QpigsData:  d,
	}

	batchMutex.Lock()
	telemetryBatch = append(telemetryBatch, row)
	if len(telemetryBatch) >= batchSize {
		go flushBatch()
	}
	batchMutex.Unlock()
}

func addQpigs2ToBatch(id int64, d Qpigs2Data) {
	batchMutex.Lock()
	for i := range telemetryBatch {
		if telemetryBatch[i].InverterID == id {
			telemetryBatch[i].Pv2InputCurrent = d.Pv2InputCurrent
			telemetryBatch[i].Pv2InputVoltage = d.Pv2InputVoltage
			telemetryBatch[i].Pv2ChargingPower = d.Pv2ChargingPower
			break
		}
	}
	batchMutex.Unlock()
}

func batchFlusher() {
	ticker := time.NewTicker(flushInterval)
	for range ticker.C {
		flushBatch()
	}
}

func flushBatch() {
	batchMutex.Lock()
	rows := telemetryBatch
	telemetryBatch = nil
	batchMutex.Unlock()

	if len(rows) == 0 {
		return
	}

	ctx := context.Background()

	copyRows := make([][]interface{}, len(rows))

	for i, r := range rows {
		copyRows[i] = []interface{}{
			r.Time, r.InverterID,
			r.AcInputVoltage, r.AcInputFrequency,
			r.AcOutputVoltage, r.AcOutputFrequency,
			r.AcOutputApparentPower, r.AcOutputActivePower,
			r.AcOutputLoad, r.BusVoltage,
			r.BatteryVoltage, r.BatteryChargingCurrent,
			r.BatteryCapacity, r.InverterHeatSinkTemperature,
			r.PvInputCurrentForBattery, r.PvInputVoltage,
			r.BatteryVoltageFromScc,
			r.BatteryDischargeCurrent,
			r.IsSbuPriorityVersionAdded == 1,
			r.IsConfigurationChanged == 1,
			r.IsSccFirmwareUpdated == 1,
			r.IsLoadOn == 1,
			r.IsChargingOn == 1,
			r.IsSccChargingOn == 1,
			r.IsAcChargingOn == 1,
			r.IsChargingToFloat == 1,
			r.IsSwitchedOn == 1,
			r.IsReserved == 1,
			r.Rsv1, r.Rsv2,
			r.PvInputPower,
			r.Pv2InputCurrent,
			r.Pv2InputVoltage,
			r.Pv2ChargingPower,
		}
	}

	_, err := dbPool.CopyFrom(
		ctx,
		pgx.Identifier{"telemetry"},
		[]string{
			"time", "inverter_id",
			"ac_input_voltage", "ac_input_frequency",
			"ac_output_voltage", "ac_output_frequency",
			"ac_output_apparent_power", "ac_output_active_power",
			"ac_output_load", "bus_voltage",
			"battery_voltage", "battery_charging_current",
			"battery_capacity", "inverter_heat_sink_temperature",
			"pv_input_current_for_battery", "pv_input_voltage",
			"battery_voltage_from_scc",
			"battery_discharge_current",
			"is_sbu_priority_version_added",
			"is_configuration_changed",
			"is_scc_firmware_updated",
			"is_load_on",
			"is_charging_on",
			"is_scc_charging_on",
			"is_ac_charging_on",
			"is_charging_to_float",
			"is_switched_on",
			"is_reserved",
			"rsv1", "rsv2",
			"pv_input_power",
			"pv2_input_current",
			"pv2_input_voltage",
			"pv2_charging_power",
		},
		pgx.CopyFromRows(copyRows),
	)

	if err != nil {
		log.Println("Batch insert error:", err)
	}
}

/* ========================
   LATEST + FAULTS
======================== */

func updateLatest(ctx context.Context, id int64, d QpigsData) {
	_, _ = dbPool.Exec(ctx,
		`INSERT INTO inverter_latest
		(inverter_id,time,ac_output_active_power,battery_voltage,pv_input_power,
		is_load_on,is_charging_on,is_scc_charging_on,is_ac_charging_on,is_switched_on)
		VALUES($1,$2,$3,$4,$5,$6,$7,$8,$9,$10)
		ON CONFLICT (inverter_id) DO UPDATE SET
		time=EXCLUDED.time,
		ac_output_active_power=EXCLUDED.ac_output_active_power,
		battery_voltage=EXCLUDED.battery_voltage,
		pv_input_power=EXCLUDED.pv_input_power,
		is_load_on=EXCLUDED.is_load_on,
		is_charging_on=EXCLUDED.is_charging_on,
		is_scc_charging_on=EXCLUDED.is_scc_charging_on,
		is_ac_charging_on=EXCLUDED.is_ac_charging_on,
		is_switched_on=EXCLUDED.is_switched_on`,
		id, time.Now(),
		d.AcOutputActivePower,
		d.BatteryVoltage,
		d.PvInputPower,
		d.IsLoadOn == 1,
		d.IsChargingOn == 1,
		d.IsSccChargingOn == 1,
		d.IsAcChargingOn == 1,
		d.IsSwitchedOn == 1,
	)
}

func updateLatestPv2(ctx context.Context, id int64, d Qpigs2Data) {
	_, _ = dbPool.Exec(ctx,
		`UPDATE inverter_latest
		SET pv2_charging_power=$1,time=$2
		WHERE inverter_id=$3`,
		d.Pv2ChargingPower, time.Now(), id)
}

func updateFaults(ctx context.Context, id int64, d QpiwsData) {
	_, _ = dbPool.Exec(ctx,
		`INSERT INTO inverter_faults
		(inverter_id,time,
		inverter_fault,bus_over_fault,bus_under_fault,bus_soft_fail_fault,
		line_fail_warning,opv_short_warning,
		inverter_voltage_too_low_fault,inverter_voltage_too_high_fault,
		over_temperature_fault,fan_locked_fault,
		battery_voltage_too_high_fault,battery_low_alarm_warning,
		battery_under_shutdown_warning,overload_fault,eeprom_fault,
		inverter_over_current_fault,inverter_soft_fail_fault,
		self_test_fail_fault,op_dc_voltage_over_fault,
		battery_open_fault,current_sensor_fail_fault,
		battery_short_fault,power_limit_warning,
		pv_voltage_high_warning,mppt_overload_fault,
		mppt_overload_warning,reserved)
		VALUES($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,
		       $11,$12,$13,$14,$15,$16,$17,$18,
		       $19,$20,$21,$22,$23,$24,$25,$26,
		       $27,$28,$29)
		ON CONFLICT (inverter_id) DO UPDATE SET
		time=EXCLUDED.time,
		inverter_fault=EXCLUDED.inverter_fault,
		bus_over_fault=EXCLUDED.bus_over_fault,
		bus_under_fault=EXCLUDED.bus_under_fault,
		bus_soft_fail_fault=EXCLUDED.bus_soft_fail_fault,
		line_fail_warning=EXCLUDED.line_fail_warning,
		opv_short_warning=EXCLUDED.opv_short_warning,
		inverter_voltage_too_low_fault=EXCLUDED.inverter_voltage_too_low_fault,
		inverter_voltage_too_high_fault=EXCLUDED.inverter_voltage_too_high_fault,
		over_temperature_fault=EXCLUDED.over_temperature_fault,
		fan_locked_fault=EXCLUDED.fan_locked_fault,
		battery_voltage_too_high_fault=EXCLUDED.battery_voltage_too_high_fault,
		battery_low_alarm_warning=EXCLUDED.battery_low_alarm_warning,
		battery_under_shutdown_warning=EXCLUDED.battery_under_shutdown_warning,
		overload_fault=EXCLUDED.overload_fault,
		eeprom_fault=EXCLUDED.eeprom_fault,
		inverter_over_current_fault=EXCLUDED.inverter_over_current_fault,
		inverter_soft_fail_fault=EXCLUDED.inverter_soft_fail_fault,
		self_test_fail_fault=EXCLUDED.self_test_fail_fault,
		op_dc_voltage_over_fault=EXCLUDED.op_dc_voltage_over_fault,
		battery_open_fault=EXCLUDED.battery_open_fault,
		current_sensor_fail_fault=EXCLUDED.current_sensor_fail_fault,
		battery_short_fault=EXCLUDED.battery_short_fault,
		power_limit_warning=EXCLUDED.power_limit_warning,
		pv_voltage_high_warning=EXCLUDED.pv_voltage_high_warning,
		mppt_overload_fault=EXCLUDED.mppt_overload_fault,
		mppt_overload_warning=EXCLUDED.mppt_overload_warning,
		reserved=EXCLUDED.reserved`,
		id, time.Now(),
		d.InverterFault == 1,
		d.BusOverFault == 1,
		d.BusUnderFault == 1,
		d.BusSoftFailFault == 1,
		d.LineFailWarning == 1,
		d.OpvShortWarning == 1,
		d.InverterVoltageTooLowFault == 1,
		d.InverterVoltageTooHighFault == 1,
		d.OverTemperatureFault == 1,
		d.FanLockedFault == 1,
		d.BatteryVoltageTooHighFault == 1,
		d.BatteryLowAlarmWarning == 1,
		d.BatteryUnderShutdownWarning == 1,
		d.OverloadFault == 1,
		d.EepromFault == 1,
		d.InverterOverCurrentFault == 1,
		d.InverterSoftFailFault == 1,
		d.SelfTestFailFault == 1,
		d.OpDcVoltageOverFault == 1,
		d.BatteryOpenFault == 1,
		d.CurrentSensorFailFault == 1,
		d.BatteryShortFault == 1,
		d.PowerLimitWarning == 1,
		d.PvVoltageHighWarning == 1,
		d.MpptOverloadFault == 1,
		d.MpptOverloadWarning == 1,
		d.Reserved == 1,
	)
}
