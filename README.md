# Horus

Horus is a distributed tool that collects snmp and ping data from various network equipments and exports them to Kafka, Prometheus or InfluxDB.

Horus' main distinguishing features compared to other snmp collectors are:

- a distributed architecture composed of a dispatcher and multiple distributed agents
- supports pushing metric results to Kafka, Prometheus and InfluxDB in parallel or selectively
- devices, metrics and agents are defined on a postgres db and can be updated in real time
- only the dispatcher is connected to the db
- can make ping statistics a la smokeping (with fping) in addition to snmp polling
- the agents receive their job requests from the controller over http and post their results directly to Kafka and the TSDB
- composite OID indexes are supported: index position is defined with a regex
- It is possible to use an alternate community for some metrics on the same device
- related snmp metrics can be grouped as measures
- profiles can be defined to group a list of measures specific to a type of device

Horus is currently used at [Kosc Telecom](https://www.kosc-telecom.fr/en/home/) to poll 2K+ various devices (switches, routers, DSLAM, OLT) every 5 minutes, with up to 27K metrics per device.


## Architecture overview

![](./doc/horus-architecture.svg)


## Install

### Building from source

To build Horus from source, you need Go compiler (version 1.13 or later). You can clone the repository and build it with the Makefile:

```
$ cd $HOME/go/src # or $GOPATH/src
$ git clone https://github.com/kosctelecom/horus.git
$ cd horus
$ make all
$ ./cmd/bin/horus-dispatcher -h
$ ./cmd/bin/horus-agent -h
```


## Usage

The Horus project compilation results in 3 binaries located in the cmd/bin directory:

- [horus-dispatcher(1)](./doc/horus-dispatcher.1.md): the dispatcher that retrieves available jobs from db and send them to agents
- [horus-agent(1)](./doc/horus-agent.1.md): the agent that performs the snmp or ping requests and sends the result to kafka, Prometheus and influxDB
- [horus-query(1)](./doc/horus-query.1.md): test command that polls a device and prints the json result to stdout


## Database creation

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

We can also import some sample metrics:

```
$ sudo -u postgres psql -d horus < metrics-sample.sql
```


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
    scrape_timeout: 15s
    metrics_path: /metrics
    static_configs:
    - targets: ['localhost:8001']

  # snmp metrics
  - job_name: 'snmp'
    scrape_interval: 5m
    scrape_timeout: 2m
    metrics_path: /snmpmetrics
    static_configs:
    - targets: ['localhost:8001']
    metric_relabel_configs:
    - source_labels: [id]
      target_label: instance

  # ping metrics
  - job_name: 'ping'
    scrape_interval: 1m
    scrape_timeout: 15s
    metrics_path: /pingmetrics
    static_configs:
    - targets: ['localhost:8001']
    metric_relabel_configs:
    - source_labels: [id]
      target_label: instance
```

## Contributing

Bugs reports and Pull Requests are welcome!


## License

Apache License 2.0, see [LICENSE](./LICENSE).
