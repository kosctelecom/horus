% horus-agent(1)

NAME
====

**horus-agent** - Performs snmp and ping job requests from the dispatcher and posts the results on the selected data backends.

SYNOPSIS
========

| **horus-agent** \[**-h**|**-v**] \[**-d** _level_] \[**--fping-max-procs** _value_] \[**--fping-packet-count** _count_]
|                 \[**--fping-path** _value_] \[**--influx-db** _value_]
|                 \[**--influx-host** _value_] \[**--influx-password** _value_]
|                 \[**--influx-retries** _value_] \[**--influx-rp** _value_]
|                 \[**--influx-timeout** _value_] \[**--influx-user** _value_] \[**-j** _count_]
|                 \[**--kafka-host** _value_] \[**--kafka-partition** _value_]
|                 \[**--kafka-topic** _value_] \[**--log** _dir_] \[**--mock**] \[**-p** _port_]
|                 \[**--prom-max-age** _sec_] \[**--prom-sweep-frequency** _sec_] \[**-s** _sec_]
|                 \[**-t** _msec_]

DESCRIPTION
===========

The agent receives job requests from the dispatcher over http. If it has remaining capacity, it accepts and queues the job. The job is an json document containing all
information about the device to poll, the metrics to retrieve and the backends where to send the results.

At the end of a polling job, the agent posts the results to Kafka or InfluxBD and keeps them in memory for Prometheus scraping. It also sends back a report to the dispatcher
with the polling duration and error if any. Ping results (min, max, avg, loss) are kept in memory for Prometheus scraping only and no report is sent back to the agent.

The result posted to Kafka is a big json document containing the aggregated poll results for each device. You can use **horus-query(1)** to get the same data on stdout.

The Prometheus metrics are named using the `<measure name>_<metric name>` pattern, for example: sysInfo\_sysUpTime and they have the following default labels: id, host,
vendor, model and category of the polled device.

Options
=======

General options
---------------

-d, --debug

:   Specifies the debug level from 1 to 3. Defaults to 0 (disabled).

-h, --help

:   Prints a help message.

-j, --snmp-jobs

:   Specifies the snmp polling job capacity of this agent. Defaults to 1.

    --log

:   Specifies the directory where the log files are written. The files are created and rotated by the glog lib (https://github.com/vma/glog).
    If not set, logs are written to stderr.

    --mock

:   Runs the agent in mock mod for snmp requests.

-p, --port

:   Specifies the listen port of the API web server. Defaults to 8080.

-s, --stat-frequency

:   Specifies the frequency in seconds at which gather and log agent stats (memory usage, ongoing polls, prometheus stats.) Disabled if set to 0 (default.)

-t, --inter-poll-delay

:   Specifies the time to wait in ms between each snmp poll request. It is used to smoothe the load and avoid spikes. Defaults to 100ms.

-v, --version

:   Prints the current version and build date.

Ping related options
--------------------

    --fping-max-procs

:   Specifies the max ping requests capacity for this agent (based on max simultaneous fping processes). Defaults to 5.

    --fping-packet-count

:   Specifies the number of ping requests sent to each host. Defaults to 15.

    --fping-path

:   Specifies the path of the fping binary. Defaults to `/usr/bin/fping`.


InfluxDB related options
------------------------

    --influx-host

:   Specifies the influxDB host address. Push to influxDB disabled if empty (default). The subsequent options are needed only if this one is set.

    --influx-db

:   Specifies the influxDB database.

    --influx-user

:   Specifies the influxDB user login.

    --influx-password

:    Specifies the influxDB user password.

    --influx-retries

:   Specifies the influxDB write retry count in case of error. Defaults to 2.

    --influx-rp

:   Specifies the influxDB retention policy for the pushed data. Defaults to "autogen".

    --influx-timeout

:   Specifies the influxDB write timeout in seconds. Defaults to 5s.


Kafka related options
---------------------

    --kafka-host

:   Specifies the Kafka broker IP address. Push to Kafka is disabled if empty (default).

    --kafka-partition

:   Specifies the Kafka write partition to use. Defaults to 0.

    --kafka-topic

:   Specifies the Kafka topic to use for the snmp results.


Prometheus related options
--------------------------

    --prom-max-age

:   Specifies the maximum time in second to keep Prometheus samples in memory. If set to 0 (default), Prometheus collectors are disabled.

    --prom-sweep-frequency

:   Specifies the cleaning frequency in second of old Prometheus samples. Defaults to 120s.

BUGS
====

See GitHub Issues: <https://github.com/kosctelecom/horus/issues>

AUTHOR
======

Vallimamod Abdullah <vma@sip.solutions>

SEE ALSO
========

**horus-dispatcher(1)**, **horus-query(1)**
