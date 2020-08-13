Database structure
==================

## agents table

- Lists all available agents, only `active` ones are taken in account.
- The `is_alive`, `load` and `last_checked_at` are updated on each keep-alive request.

## devices table

- Lists all devices to poll and ping. Only `active` ones are taken into account.
- Each device is part of a profile through the `profile_id` field. It defines the list of metrics to poll (see below).
- The `is_polling` flag is set by the dispatcher to lock the device during the polling, it is unlocked when the dispatcher receives the poll report. A cleaner goroutine unlocks periodically locked devices with no polling.
- The `last_polled_at` and `last_pinged_at` fields are updated by the dispatcher on each successful job submission.
- The following table lists all fields with their description and default values:

| field                      | type   | default | description
| ---------------------------| ------ | ------- | --------------------------------------------------------------
| active                     | bool   | false   | flag to activate device polling.
| hostname                   | string | -       | device hostname (fqdn)
| ip\_address                | string | -       | device IP address for snmp requests. Takes precedence over hostname; if null then hostname is used.
| is\_polling                | bool   | false   | internal field: flag telling wether there is an ongoing poll
| last\_pinged\_at           | tstamp | -       | internal field: last ping time
| last\_polled\_at           | tstamp | -       | internal field: last snmp polling time
| ping\_frequency            | int    | 0       | ping frequency in seconds for the device. The device is pinged only if the value of this field is > 0.
| polling\_frequency         | int    | 0       | snmp polling frequency in seconds for the device. The device is polled only if the value of this field is > 0.
| profile\_id                | int    | -       | the id of the device profile (see profiles table below)
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

## metrics table

This table lists all snmp metrics (OID) to poll from devices. The main fields are:

| field                 | type     | default | description
| ----------------------| -------- | ------- | ---------------------------------------------------
| name                  | string   | -       | the canonical metric name as found on the MIB files
| oid                   | string   | -       | the metric OID with the leading dot
| description           | text     | -       | description of the metric (as found in the MIB)
| export\_as\_label     | bool     | false   | flag telling wether this metric must be exported as a label. If set, the value is converted to string first.
| exported\_name        | string   | null    | name of the corresponding prometheus metric or label. Defaults to `name` if unset.
| index\_pattern        | string   | null    | applicable only to indexed metrics. It is a regexp that defines how to extract the index from the OID, if it is not at the end. It is a go compatible regexp with one group for the index position. Example: `.1.3.6.1.2.1.10.48.1.5.1.1.(\d+).2.1.\d`
| polling\_frequency    | int      | 0       | defines a specific polling frequency for this metric. It must be a multiple of the device polling frequency, allows to poll this metric less frequently.
| post\_processors      | []string | {}      | a list of post processing transformations to apply in order to the retrieved metric. See below for details.

The post processors allow to normalize a retrieved metric that has a string or numeric value. The current list is:

- For string values:
    - `trim`: trims spaces at the beginning and end. It is the default processor for all string metrics.
    - `parse-int`: parses the string to a numeric value. Typically for 64bits counters that are returned as `OctetString`.
    - `parse-hex-le`: parses the hexadecimal string as a numeric value in little-endian order.
    - `parse-hex-be`: parses the hexadecimal string as a numeric value in big-endian order.
    - `extract-int` or `extract-float`: extracts a numeric value from a string. For example: "Rx level: -12.5 dBm" returns -12.5 as a float.

- For numeric values:
    - `div-<divisor>` or `div:<divisor>`: divides the retrieved value by the divisor, a float number.
    - `mul-<multiplicator>` or `mul:<multiplicator>`: multiplies the retrieved value by the multiplicator, a float number.


## measures table

- A measure is a group of similar snmp metrics to poll. It is a list of either scalar or indexed metrics as defined by the `is_indexed` flag.
- It is not possible to mix scalar and indexed metrics in one measure.
- The `index_metric_id` references the metric to use as index for indexed measures. It is possible to have indexed metrics without a defined index metric for components that does not have a specific name like fans or PSUs.
- The `filter_metric_id` references the metric over which to filter the measure.
- The `filter_pattern` defines the regex pattern used to filter the measure. The measure is kept only if the filter\_metric's results matches this pattern.
- It is possible to invert the filter match result with the `invert_filter_match` flag.
- Measures and metrics have a N:N relationship defined in the `measure_metrics` table.
- The `use_alternate_community` flag tells to use the device's other community to poll all metrics of this measure.
- It is possible to select the export destination with `to_influx`, `to_kafka` and `to_prometheus` flags.

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
