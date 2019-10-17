# Horus

Horus is a distributed tool that collects snmp and ping data from various network equipments and exports them to Kafka, Prometheus or InfluxDB.

Horus' main distinguishing features compared to other snmp collectors are:

- a distributed architecture composed of a dispatcher and multiple distributed agents
- supports pushing results to Kafka, Prometheus and InfluxDB in parallel
- devices, metrics and agents are defined on a postgres db and can be updated in real time
- the dispatcher is the only one connected the db
- can make ping (via fping) and snmp queries
- the agents receive their job requests from the controller over http and post their results directly to Kafka and the TSDB
- composite OID indexes are supported: index position is defined with a regex
- related snmp metrics can be grouped as measures
- profiles can be defined to group a list of measures specific to a type of device


## Architecture overview

![](./doc/horus-architecture.svg)


## Install

### Building from source

To build Horus from source, you need a working Go environment (version 1.13 or later). You can clone the repository and build it with the Makefile:

```
$ git clone https://github.com/kosctelecom/horus.git
$ cd horus
$ make all
$ ./cmd/bin/horus-dispatcher -h
$ ./cmd/bin/horus-agent -h
```


## Usage

The Horus project compilation results in 3 binaries located in the cmd/bin directory:

- `horus-dispatcher`: the dispatcher that retrieves available jobs from db and send them to agents
- `horus-agent`: the agent that performs the snmp or ping requests and sends the result to kafka, Prometheus and influxDB
- `horus-query`: polls a given device id from db and prints the results as a json on stdout

The detailed usage of each binary is available in the [doc/](./doc/) folder.


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
