Database structure
==================

## agents table

- Lists all available agents, only `active` ones are taken in account.
- The `is_alive`, `load` and `last_checked_at` are update on each keep-alive request.

## devices table

- Lists all devices to poll, only `active` ones are taken in account.
- Each device is part of a profile through the `profile_id` field. It defines the list of metrics to poll (see below).
- The `is_polling` flag is set by the dispatcher to lock the device during the polling, it is unlocked when the dispatcher receives the poll report. A cleaner goroutine unlocks periodically locked devices with no polling.
- The `last_polled_at` and `last_pinged_at` fields are updated by the dispatcher on each successful job submission.
- The following table lists the main fields with their description and default values:

| field                      | type   | default | description
| ---------------------------| ------ | ------- | --------------------------------------------------------------
| active                     | bool   | false   | flag to activate device polling.
| hostname                   | string | -       | device hostname (fqdn)
| ip\_address                | string | -       | device IP address for snmp requests.
| ping\_frequency            | int    | 0       | ping frequency in seconds for the device. The device is pinged only if the value of this field is > 0.
| polling\_frequency         | int    | 0       | snmp polling frequency in seconds for the device. The device is polled only if the value of this field is > 0.
| snmp\_alternate\_community | string |""       | alternate snmp community to use for metrics with `use_alternate_community` flag set (same as `snmp_community` if empty)
| snmp\_community            | string | -       | device snmp community.
| snmp\_connection\_count    | bool   | 1       | max number of parallel snmp connections allowed for the device.
| snmp\_disable\_bulk        | bool   | false   | flag to disable snmp bulk requests. Automatically set to true for snmp v1.
| snmp\_port                 | int    | 161     | device snmp port.
| snmp\_retries              | int    | 1       | number of snmp query retries (excluding initial query) in case of timeout.
| snmp\_timeout              | int    | 10      | timeout in seconds for snmp queries.
| snmp\_version              | string | 2c      | device snmp version, one of `1`, `2c` or `3`.
| snmpv3\_auth\_passwd       | string | ""      | snmp v3 authentication password.
| snmpv3\_auth\_proto        | string | ""      | snmp v3 authentication protocol, one of `MD5` or `SHA`.
| snmpv3\_auth\_user         | string | ""      | snmp v3 authentication user, mandatory when security level is AuthNoPriv or AuthPriv.
| snmpv3\_privacy\_passwd    | string | ""      | snmp v3 privacy password, mandatory when security level is AuthPriv.
| snmpv3\_privacy\_proto     | string | ""      | snmp v3 privacy protocol, one of `DES` or `AES`.
| snmpv3\_security\_level    | string | ""      | snmp v3 security level, one of `NoAuthNoPriv`, `AuthNoPriv` or `AuthPriv`.
| tags                       | json   | {}      | json to export as labels or tags in all measures of this device. Default labels already include: id, hostname, category, vendor and model
| to\_influx                 | bool   | false   | flag to export results to influxdb. If set, agents must also have influx connection to make it work.
| to\_kafka                  | bool   | false   | flag to export results to kafka. Same note as above.
| to\_prometheus             | bool   | false   | flag to export results to prometheus. One of these 3 flags must be activated.

## metrics table

- This table lists all snmp metrics (OID) to poll from devices.
- The `name` is the canonical metric name as found on the MIB files.
- The `oid` is the metric OID with the leading dot
- The `index_pattern` field is applicable only to indexed metrics and is a regexp that defines how to extract a composite index that is not at the end of the base OID. It is a go compatible regexp with one group for the index position. Example: `.1.3.6.1.2.1.10.48.1.5.1.1.(\d+).2.1.\d`
- The `export_as_label` flag indicates wether the result should be exported as a Prometheus label, when it's a string for example.
- The `use_alternate_community` flag indicates wether to use the alternate snmp community for this metric if it is defined, falls back to the main community otherwise.
- The `polling_frequency` field defines a specific polling frequency for this metric. It must be a multiple of the device polling frequency, allows to poll this metric less frequently.
- The `is_string_counter` flag indicates wether the metric value is a counter exported as a string (ADVA specific `StringCounter` type)

## measures table

- A measure is a group of similar snmp metrics to poll.
- A measure is composed of either scalar or indexed metrics as defined by the `is_indexed` flag. It is not possible to mix both types of metrics.
- The `index_metric_id` references the metric to use as index for indexed measures.
- The `filter_metric_id` references the metric over which to filter the measure.
- The `filter_pattern` defines the regex pattern used to filter the measure. The measure is kept only if the filter\_metric's results matches this pattern.
- It is possible to invert the filter match result with the `invert_filter_match` flag.
- Measures and metrics have a N:N relationship defined in the `measure_metrics` table.

## profiles table

- A profile is defined by the tuple (category, vendor, model) that is affected to a device. It allows to easily define a list of measures common to a group of devices (routers, switch, etc.)
- Profiles and measures have a N:N relationship defined in the `profile_measures` table.

## reports table

This table keeps a list of ongoing polling jobs. When a report is received from an agent, the entry is removed if there was no error. Otherwise, the poll error is saved for inspection.
Rows whose `requested_at` field is older than a defined delay are periodically removed by the dispatcher (parametrable via `--poll-error-retention-period` param).

## metric\_poll\_times table

This is an internal table that keeps the last polling date for each metric on each device.

## measure\_metrics table

This table defines the N:N relation between measures and metrics.

## profile\_measures table

This table defines the N:N relation between profiles and measures.
