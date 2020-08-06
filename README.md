# Horus

Horus is a distributed tool that collects snmp and ping data from various network equipments and exports them to Kafka, Prometheus, NATS.io or InfluxDB.

Horus' main distinguishing features compared to other snmp collectors are:

- a distributed architecture composed of a dispatcher and multiple distributed agents
- supports pushing metric results to Kafka, Prometheus, NATS, and InfluxDB in parallel or selectively
- devices, metrics and agents are defined on a postgres db and can be updated in real time
- only the dispatcher is connected to the db
- can make ping statistics a la smokeping (with fping) in addition to snmp polling
- the agents receive their job requests from the controller over http and post their results directly to the message bus or TSDB
- composite OID indexes are supported: index position is defined with a regex
- It is possible to use an alternate community for some metrics on the same device
- related snmp metrics can be grouped as measures
- profiles can be defined to group a list of measures specific to a type of device
- retrieved metrics can be post-processed to change scale or convert hex/string values to numeric ones

Horus is currently used at [Kosc Telecom](https://www.kosc-telecom.fr/en/home/) to poll 2K+ various devices (switches, routers, DSLAM, OLT) every 1 to 5 minutes,
with up to 27K metrics per device.  The polling is dispatched over 4 agents collecting each about 1M metrics, using less than 3GB memory and 2 cpu cores.


## Architecture overview

![](./doc/horus-architecture.svg)


## Install

### Building from source

To build Horus from source, you need Go compiler (version 1.14 or later). You can clone the repository and build it with the Makefile:

```
$ cd $HOME/go/src # or $GOPATH/src
$ git clone https://github.com/kosctelecom/horus.git
$ cd horus
$ make all
$ ./cmd/bin/horus-dispatcher -h
$ ./cmd/bin/horus-agent -h
```

The project compilation results in 3 binaries located in the cmd/bin directory:

- [horus-dispatcher(1)](./doc/horus-dispatcher.1.md): the dispatcher that retrieves available jobs from db and send them to agents
- [horus-agent(1)](./doc/horus-agent.1.md): the agent that performs the snmp or ping requests and sends the result to select message buses and TSDB
- [horus-query(1)](./doc/horus-query.1.md): test command that polls a device and prints the json result to stdout


## Creating and populating the database

We first need to create a postgres user and database. In the psql admin console, run:

```
postgres=# CREATE ROLE horus WITH LOGIN ENCRYPTED PASSWORD 'secret';
postgres=# CREATE DATABASE horus WITH OWNER horus;
postgres=# GRANT ALL PRIVILEGES ON DATABASE horus TO horus;
```

Then we can import the table schema:

```
$ sudo -u postgres psql -d horus < horus.sql
```

See [doc/database.md](./doc/database.md) for a detailed description of each table.

Then we can create a local agent running on port 8000:

```
 horus=# INSERT INTO agents (id, ip_address, port, active) VALUES (1, '127.0.0.1', 8000, true);
```

and a device to poll:

```
horus=# INSERT INTO devices (id, profile_id, active, hostname, ip_address, snmp_version, snmp_community, polling_frequency, ping_frequency)
             VALUES (1, 1, true, 'switch-01.lan', '10.0.0.1', '2c', 'mycommunity', 60, 30);
```

and import some sample metrics:

```
$ sudo -u postgres psql -d horus < metrics-sample.sql
```

This script defines:
- a profile for a generic switch
- a scalar measure for device info (name, uptime, etc.)
- 3 indexed measures for each interface status, inbound and outbound counters
- the corresponding snmp metrics and relations


## Starting the agent and the dispatcher

With the previous database config, we can start an agent and the dispatcher (preferably on different shells):

```
$ ./cmd/bin/horus-agent -d1 --port 8000 --prom-max-age 900 --kafka-host kafka.kosc.local --kafka-partition 0 --kafka-topic horus
$ ./cmd/bin/horus-dispatcher -c postgres://horus:secret@localhost/horus -d1
```

You can start the agent or the dispatcher without any argument to get all options and their usage.


## Prometheus config

There are 3 scrape endpoints available to Prometheus:

- `/metrics` for agent's internal metrics (ongoing polls count, memory usage...)
- `/snmpmetrics` for snmp metrics
- `/pingmetrics` for ping metrics

Here is an example scrape config from `prometheus.yml`:

```
scrape_configs:
  # agent metrics (mem usage, ongoing count, etc.)
  - job_name: 'agent'
    scrape_interval: 30s
    scrape_timeout: 10s
    metrics_path: /metrics
    static_configs:
    - targets: ['localhost:8000']

  # snmp metrics
  - job_name: 'snmp'
    scrape_interval: 1m
    scrape_timeout: 10s
    metrics_path: /snmpmetrics
    static_configs:
    - targets: ['localhost:8000']
    metric_relabel_configs:
    - source_labels: [hostname]
      target_label: instance

  # ping metrics
  - job_name: 'ping'
    scrape_interval: 30s
    scrape_timeout: 10s
    metrics_path: /pingmetrics
    static_configs:
    - targets: ['localhost:8000']
    metric_relabel_configs:
    - source_labels: [hostname]
      target_label: instance
```

## Contributing

Bugs reports and Pull Requests are welcome!


## License

Apache License 2.0, see [LICENSE](./LICENSE).
