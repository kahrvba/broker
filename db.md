root@teknikwebserver-virtual-machine:/opt/telemetry-api#   psql -h localhost -U telemetry -d inverterdb -P pager=off -c "\d+ inverters"
Password for user telemetry: 
                                                                  Table "public.inverters"
   Column   |           Type           | Collation | Nullable |                Default                | Storage  | Compression | Stats target | Description 
------------+--------------------------+-----------+----------+---------------------------------------+----------+-------------+--------------+-------------
 id         | bigint                   |           | not null | nextval('inverters_id_seq'::regclass) | plain    |             |              | 
 serial     | text                     |           | not null |                                       | extended |             |              | 
 created_at | timestamp with time zone |           |          | now()                                 | plain    |             |              | 
Indexes:
    "inverters_pkey" PRIMARY KEY, btree (id)
    "inverters_serial_key" UNIQUE CONSTRAINT, btree (serial)
Access method: heap

root@teknikwebserver-virtual-machine:/opt/telemetry-api# 


root@teknikwebserver-virtual-machine:/opt/telemetry-api#   psql -h localhost -U telemetry -d inverterdb -P pager=off -c "\d+ telemetry"
Password for user telemetry: 
                                                            Table "public.telemetry"
             Column             |           Type           | Collation | Nullable | Default | Storage | Compression | Stats target | Description 
--------------------------------+--------------------------+-----------+----------+---------+---------+-------------+--------------+-------------
 time                           | timestamp with time zone |           | not null |         | plain   |             |              | 
 inverter_id                    | bigint                   |           | not null |         | plain   |             |              | 
 ac_input_voltage               | double precision         |           |          |         | plain   |             |              | 
 ac_input_frequency             | double precision         |           |          |         | plain   |             |              | 
 ac_output_voltage              | double precision         |           |          |         | plain   |             |              | 
 ac_output_frequency            | double precision         |           |          |         | plain   |             |              | 
 ac_output_apparent_power       | integer                  |           |          |         | plain   |             |              | 
 ac_output_active_power         | integer                  |           |          |         | plain   |             |              | 
 ac_output_load                 | integer                  |           |          |         | plain   |             |              | 
 bus_voltage                    | integer                  |           |          |         | plain   |             |              | 
 battery_voltage                | double precision         |           |          |         | plain   |             |              | 
 battery_charging_current       | integer                  |           |          |         | plain   |             |              | 
 battery_capacity               | integer                  |           |          |         | plain   |             |              | 
 inverter_heat_sink_temperature | integer                  |           |          |         | plain   |             |              | 
 pv_input_current_for_battery   | double precision         |           |          |         | plain   |             |              | 
 pv_input_voltage               | double precision         |           |          |         | plain   |             |              | 
 battery_voltage_from_scc       | double precision         |           |          |         | plain   |             |              | 
 battery_discharge_current      | integer                  |           |          |         | plain   |             |              | 
 is_sbu_priority_version_added  | boolean                  |           |          |         | plain   |             |              | 
 is_configuration_changed       | boolean                  |           |          |         | plain   |             |              | 
 is_scc_firmware_updated        | boolean                  |           |          |         | plain   |             |              | 
 is_load_on                     | boolean                  |           |          |         | plain   |             |              | 
 is_charging_on                 | boolean                  |           |          |         | plain   |             |              | 
 is_scc_charging_on             | boolean                  |           |          |         | plain   |             |              | 
 is_ac_charging_on              | boolean                  |           |          |         | plain   |             |              | 
 is_charging_to_float           | boolean                  |           |          |         | plain   |             |              | 
 is_switched_on                 | boolean                  |           |          |         | plain   |             |              | 
 is_reserved                    | boolean                  |           |          |         | plain   |             |              | 
 rsv1                           | integer                  |           |          |         | plain   |             |              | 
 rsv2                           | integer                  |           |          |         | plain   |             |              | 
 pv_input_power                 | integer                  |           |          |         | plain   |             |              | 
 pv2_input_current              | double precision         |           |          |         | plain   |             |              | 
 pv2_input_voltage              | double precision         |           |          |         | plain   |             |              | 
 pv2_charging_power             | integer                  |           |          |         | plain   |             |              | 
Indexes:
    "telemetry_inverter_time_idx" btree (inverter_id, "time" DESC)
    "telemetry_time_idx" btree ("time" DESC)
Child tables: _timescaledb_internal._hyper_3_2_chunk,
              _timescaledb_internal._hyper_3_3_chunk,
              _timescaledb_internal._hyper_3_4_chunk
Access method: heap

root@teknikwebserver-virtual-machine:/opt/telemetry-api# 



root@teknikwebserver-virtual-machine:/opt/telemetry-api#   psql -h localhost -U telemetry -d inverterdb -P pager=off -c "\d+ inverter_latest"
Password for user telemetry: 
                                                     Table "public.inverter_latest"
         Column         |           Type           | Collation | Nullable | Default | Storage | Compression | Stats target | Description 
------------------------+--------------------------+-----------+----------+---------+---------+-------------+--------------+-------------
 inverter_id            | bigint                   |           | not null |         | plain   |             |              | 
 time                   | timestamp with time zone |           | not null |         | plain   |             |              | 
 ac_output_active_power | integer                  |           |          |         | plain   |             |              | 
 battery_voltage        | double precision         |           |          |         | plain   |             |              | 
 pv_input_power         | integer                  |           |          |         | plain   |             |              | 
 pv2_charging_power     | integer                  |           |          |         | plain   |             |              | 
 is_load_on             | boolean                  |           |          |         | plain   |             |              | 
 is_charging_on         | boolean                  |           |          |         | plain   |             |              | 
 is_scc_charging_on     | boolean                  |           |          |         | plain   |             |              | 
 is_ac_charging_on      | boolean                  |           |          |         | plain   |             |              | 
 is_switched_on         | boolean                  |           |          |         | plain   |             |              | 
Indexes:
    "inverter_latest_pkey" PRIMARY KEY, btree (inverter_id)
Access method: heap

root@teknikwebserver-virtual-machine:/opt/telemetry-api# 


root@teknikwebserver-virtual-machine:/opt/telemetry-api#   psql -h localhost -U telemetry -d inverterdb -P pager=off -c "\d+ inverter_faults"
Password for user telemetry: 
                                                           Table "public.inverter_faults"
              Column               |           Type           | Collation | Nullable | Default | Storage | Compression | Stats target | Description 
-----------------------------------+--------------------------+-----------+----------+---------+---------+-------------+--------------+-------------
 inverter_id                       | bigint                   |           | not null |         | plain   |             |              | 
 time                              | timestamp with time zone |           | not null |         | plain   |             |              | 
 inverter_fault                    | boolean                  |           |          |         | plain   |             |              | 
 bus_over_fault                    | boolean                  |           |          |         | plain   |             |              | 
 bus_under_fault                   | boolean                  |           |          |         | plain   |             |              | 
 bus_soft_fail_fault               | boolean                  |           |          |         | plain   |             |              | 
 line_fail_warning                 | boolean                  |           |          |         | plain   |             |              | 
 opv_short_warning                 | boolean                  |           |          |         | plain   |             |              | 
 inverter_voltage_too_low_fault    | boolean                  |           |          |         | plain   |             |              | 
 inverter_voltage_too_high_fault   | boolean                  |           |          |         | plain   |             |              | 
 over_temperature_fault            | boolean                  |           |          |         | plain   |             |              | 
 fan_locked_fault                  | boolean                  |           |          |         | plain   |             |              | 
 battery_voltage_too_high_fault    | boolean                  |           |          |         | plain   |             |              | 
 battery_low_alarm_warning         | boolean                  |           |          |         | plain   |             |              | 
 battery_under_shutdown_warning    | boolean                  |           |          |         | plain   |             |              | 
 overload_fault                    | boolean                  |           |          |         | plain   |             |              | 
 eeprom_fault                      | boolean                  |           |          |         | plain   |             |              | 
 inverter_over_current_fault       | boolean                  |           |          |         | plain   |             |              | 
 inverter_soft_fail_fault          | boolean                  |           |          |         | plain   |             |              | 
 self_test_fail_fault              | boolean                  |           |          |         | plain   |             |              | 
 op_dc_voltage_over_fault          | boolean                  |           |          |         | plain   |             |              | 
 battery_open_fault                | boolean                  |           |          |         | plain   |             |              | 
 current_sensor_fail_fault         | boolean                  |           |          |         | plain   |             |              | 
 battery_short_fault               | boolean                  |           |          |         | plain   |             |              | 
 power_limit_warning               | boolean                  |           |          |         | plain   |             |              | 
 pv_voltage_high_warning           | boolean                  |           |          |         | plain   |             |              | 
 mppt_overload_fault               | boolean                  |           |          |         | plain   |             |              | 
 mppt_overload_warning             | boolean                  |           |          |         | plain   |             |              | 
 reserved                          | boolean                  |           |          |         | plain   |             |              | 
 reserved_13                       | boolean                  |           |          |         | plain   |             |              | 
 reserved_15                       | boolean                  |           |          |         | plain   |             |              | 
 battery_too_low_to_charge_warning | boolean                  |           |          |         | plain   |             |              | 
 reserved_30                       | boolean                  |           |          |         | plain   |             |              | 
 reserved_31                       | boolean                  |           |          |         | plain   |             |              | 
Indexes:
    "inverter_faults_pkey" PRIMARY KEY, btree (inverter_id)
Access method: heap

root@teknikwebserver-virtual-machine:/opt/telemetry-api# 


root@teknikwebserver-virtual-machine:/opt/telemetry-api#   psql -h localhost -U telemetry -d inverterdb -P pager=off -c "\d+ mqtt_messages" 
Password for user telemetry: 
                                                                   Table "public.mqtt_messages"
    Column    |           Type           | Collation | Nullable |                  Default                  | Storage  | Compression | Stats target | Description 
--------------+--------------------------+-----------+----------+-------------------------------------------+----------+-------------+--------------+-------------
 id           | bigint                   |           | not null | nextval('mqtt_messages_id_seq'::regclass) | plain    |             |              | 
 received_at  | timestamp with time zone |           | not null | now()                                     | plain    |             |              | 
 inverter_id  | bigint                   |           | not null |                                           | plain    |             |              | 
 serial       | text                     |           | not null |                                           | extended |             |              | 
 metric       | text                     |           | not null |                                           | extended |             |              | 
 topic        | text                     |           | not null |                                           | extended |             |              | 
 raw_payload  | text                     |           | not null |                                           | extended |             |              | 
 payload_json | jsonb                    |           |          |                                           | extended |             |              | 
Indexes:
    "mqtt_messages_pkey" PRIMARY KEY, btree (id)
    "mqtt_messages_inverter_time_idx" btree (inverter_id, received_at DESC)
    "mqtt_messages_metric_time_idx" btree (metric, received_at DESC)
Access method: heap

root@teknikwebserver-virtual-machine:/opt/telemetry-api# 