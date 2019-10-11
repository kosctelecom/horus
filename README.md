# Horus

Horus is a distributed tool that collects snmp and ping data from various network equipments and exports them to Kafka, Prometheus or InfluxDB.

Horus' main distinguishing features, compared to other snmp collectors are:

- a distributed architecture composed of a dispatcher and multiple agents
- supports pushing results to Kafka, Prometheus and InfluxDB in parallel
- can make ping (via fping) and snmp queries
- the agents receive their job requests from the controller over http and post their results directly to the Kafka topic or the TSDB
- composite OID indexes are supported: an index can be extracted at any position with a regex
- related snmp metrics can be grouped as measures
- profiles can be defined to group a list of measures specific to a type of device


## Architecture overview

TODO


## Install

### Building from source

To build Horus from source, you will need a working Go environment (version 1.13 or later). You can clone the repository and build it with the Makefile:

```
$ git clone https://github.com/kosctelecom/horus-core.git
$ cd horus-core
$ make all
$ ./cmd/bin/horus-dispatcher -h
$ ./cmd/bin/horus-agent -h
```


## Usage

TODO


## Database schema

TODO


## Contributing

TODO


## License

TODO
