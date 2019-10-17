// Copyright 2019 Kosc Telecom.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package main

import (
	"context"
	"fmt"
	"horus/agent"
	"horus/log"
	"horus/model"
	"net/http"
	"os"
	"os/signal"
	"strconv"
	"syscall"
	"time"

	_ "net/http/pprof"

	"github.com/vma/getopt"
	"github.com/vma/glog"
	"github.com/vma/httplogger"
)

var (
	// Revision is the git revision, set at compilation
	Revision string

	// Build is the build time, set at compilation
	Build string

	// Branch is the git branch, set at compilation
	Branch string

	debug          = getopt.IntLong("debug", 'd', 0, "debug level")
	port           = getopt.Int16Long("port", 'p', 8080, "API webserver listen port", "port")
	showVersion    = getopt.BoolLong("version", 'v', "Print version and build date")
	snmpJobCount   = getopt.IntLong("snmp-jobs", 'j', 1, "Number of simultaneous snmp jobs", "count")
	mock           = getopt.BoolLong("mock", 0, "Run the agent in mock mode (no actual snmp query)")
	statUpdFreq    = getopt.IntLong("stat-frequency", 's', 0, "Agent stats update frequency (disabled if 0)", "sec")
	interPollDelay = getopt.IntLong("inter-poll-delay", 't', 100, "time to wait between successive poll start", "msec")
	logDir         = getopt.StringLong("log", 0, "", "directory for log files, disabled if empty (all log goes to stderr)", "dir")

	// prometheus conf
	maxResAge = getopt.IntLong("prom-max-age", 0, 0, "Maximum time to keep prometheus samples in mem, disabled if 0", "sec")
	sweepFreq = getopt.IntLong("prom-sweep-frequency", 0, 120, "Prometheus old samples cleaning frequency", "sec")

	// influx conf
	influxHost    = getopt.StringLong("influx-host", 0, "", "influx server address (influx push disabled if empty)")
	influxUser    = getopt.StringLong("influx-user", 0, "", "influx user")
	influxPasswd  = getopt.StringLong("influx-password", 0, "", "influx user password")
	influxDB      = getopt.StringLong("influx-db", 0, "", "influx database")
	influxRP      = getopt.StringLong("influx-rp", 0, "autogen", "influx retention policy for pushed data")
	influxTimeout = getopt.IntLong("influx-timeout", 0, 5, "influx write timeout in second")
	influxRetries = getopt.IntLong("influx-retries", 0, 2, "influx write retries in case of error")

	// kafka conf
	kafkaHost      = getopt.StringLong("kafka-host", 0, "", "kafka broker ip address (kafka push disabled if empty)")
	kafkaTopic     = getopt.StringLong("kafka-topic", 0, "", "kafka snmp results topic")
	kafkaPartition = getopt.IntLong("kafka-partition", 0, 0, "kafka write partition")

	// fping conf
	pingPacketCount = getopt.IntLong("fping-packet-count", 0, 15, "number of ping requests sent to each host")
	maxPingProcs    = getopt.IntLong("fping-max-procs", 0, 5, "max number of simultaneous fping processes")
	fpingExec       = getopt.StringLong("fping-path", 0, agent.DefaultFpingExec, "specify the location of the fping program")
)

func main() {
	getopt.SetParameters("")
	getopt.Parse()

	if len(os.Args) == 1 {
		getopt.PrintUsage(os.Stderr)
		os.Exit(1)
	}

	glog.WithConf(glog.Conf{Verbosity: *debug, LogDir: *logDir, PrintLocation: *debug > 0})

	if *showVersion {
		fmt.Printf("Revision: %s\nBuild: %s\nBranch: %s\n", Revision, Build, Branch)
		return
	}

	ctx, cancel := context.WithCancel(context.Background())
	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt, syscall.SIGTERM)
	go func() {
		for {
			select {
			case <-c:
				glog.Warning("interrupt received, canceling all requests")
				cancel()
				time.Sleep(500 * time.Millisecond) // wait for all job reports to be sent
				os.Exit(0)
			case <-ctx.Done():
				return
			}
		}
	}()

	if *maxResAge == 0 && *influxHost == "" && *kafkaHost == "" {
		glog.Exitf("either prom-max-age or influx-host or kafka-host must be defined")
	}

	if *maxPingProcs > 0 {
		agent.FpingExec = *fpingExec
		if _, err := os.Stat(agent.FpingExec); os.IsNotExist(err) {
			glog.Exitf("fping binary not found at %s", agent.FpingExec)
		}
		if *pingPacketCount == 0 {
			glog.Exitf("fping-packet-count cannot be zero")
		}
	}
	agent.MockMode = *mock
	agent.MaxRequests = *snmpJobCount
	agent.StatsUpdFreq = *statUpdFreq
	agent.InterPollDelay = time.Duration(*interPollDelay) * time.Millisecond
	agent.PingPacketCount = *pingPacketCount
	agent.MaxPingProcs = *maxPingProcs
	agent.StopCtx = ctx

	if err := agent.Init(); err != nil {
		glog.Fatalf("agent init: %v", err)
	}

	if *maxResAge > 0 {
		err := agent.InitCollectors(*maxResAge, *sweepFreq)
		if err != nil {
			glog.Fatalf("prom init: %v", err)
		}
	}

	if *influxHost != "" {
		err := agent.NewInfluxClient(*influxHost, *influxUser, *influxPasswd,
			*influxDB, *influxRP, *influxTimeout, *influxRetries)
		if err != nil {
			glog.Fatalf("influx client init: %v", err)
		}
	}

	if *kafkaHost != "" {
		err := agent.NewKafkaClient(*kafkaHost, *kafkaTopic, *kafkaPartition)
		if err != nil {
			glog.Fatalf("kafka client init: %v", err)
		}
	}

	http.HandleFunc(model.SnmpJobURI, agent.HandleSnmpRequest)
	http.HandleFunc(model.CheckURI, agent.HandleCheck)
	http.HandleFunc(model.OngoingURI, agent.HandleOngoing)
	http.HandleFunc(model.PingJobURI, agent.HandlePingRequest)
	http.HandleFunc("/-/stop", handleStop)
	http.HandleFunc("/-/debug", handleDebugLevel)
	logger := httplogger.CommonLogger(log.Writer{})
	log.Infof("starting web server on port %d", *port)
	glog.Fatal(http.ListenAndServe(fmt.Sprintf(":%d", *port), logger(http.DefaultServeMux)))
}

// handleStop handles agent graceful stop. Waits for all polling
// jobs to finish and for a last prom scrape before exiting.
func handleStop(w http.ResponseWriter, r *http.Request) {
	log.Infof("** graceful stop request from %s", r.RemoteAddr)
	initialScrapeCount := agent.SnmpScrapeCount()
	agent.GracefulQuitMode = true
	if agent.CurrentLoad() == 0 {
		goto end
	}
	for agent.CurrentLoad() > 0 {
		time.Sleep(500 * time.Millisecond)
	}
	if *maxResAge > 0 {
		// wait for a final prom scrape with a 5mn timeout
		remainingLoops := 600
		for agent.SnmpScrapeCount() == initialScrapeCount && remainingLoops > 0 {
			time.Sleep(500 * time.Millisecond)
			remainingLoops--
		}
	}
end:
	_, cancel := context.WithCancel(context.Background())
	cancel()
	time.Sleep(500 * time.Millisecond)
	w.WriteHeader(http.StatusNoContent)
	os.Exit(0)
}

// handleDebugLevel sets the application debug level dynamically.
func handleDebugLevel(w http.ResponseWriter, r *http.Request) {
	lvl := r.FormValue("level")
	dbgLevel, err := strconv.Atoi(lvl)
	if err != nil || dbgLevel < 0 || dbgLevel > 3 {
		log.Errorf("invalid level %s", lvl)
		http.Error(w, "invalid debug level "+lvl, 400)
		return
	}
	glog.SetLevel(int32(dbgLevel))
	w.WriteHeader(http.StatusOK)
}
