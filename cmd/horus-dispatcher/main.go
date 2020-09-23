// Copyright 2019-2020 Kosc Telecom.
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
	"net/http"
	_ "net/http/pprof"
	"os"
	"os/signal"
	"strconv"
	"strings"
	"syscall"
	"time"

	"github.com/kosctelecom/horus/dispatcher"
	"github.com/kosctelecom/horus/log"
	"github.com/kosctelecom/horus/model"
	_ "github.com/lib/pq"
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

	debug           = getopt.IntLong("debug", 'd', 0, "debug level")
	showVersion     = getopt.BoolLong("version", 'v', "Print version and build date")
	localIP         = getopt.StringLong("ip", 'i', dispatcher.LocalIP, "API & report web server local listen IP, must be non-zero", "address")
	port            = getopt.IntLong("port", 'p', dispatcher.Port, "API & report web server listen port", "port")
	dsn             = getopt.StringLong("dsn", 'c', "", "postgres db DSN", "url")
	unlockFreq      = getopt.IntLong("device-unlock-freq", 'u', 600, "device unlocker frequency (resets the db is_polling flag)", "seconds")
	keepAliveFreq   = getopt.IntLong("agent-keepalive-freq", 'k', 30, "agent keep-alive frequency", "seconds")
	dbSnmpQueryFreq = getopt.IntLong("db-snmp-freq", 'q', 30, "db query frequency for available polling jobs (0 to disable snmp)", "seconds")
	dbPingQueryFreq = getopt.IntLong("db-ping-freq", 'g', 10, "db query frequency for available ping jobs (0 to disable ping)", "seconds")
	pingBatchCount  = getopt.IntLong("ping-batch-count", 0, 100, "number of hosts per fping process")
	dbPollErrRP     = getopt.IntLong("error-flush-freq", 'r', 4, "how long to keep poll errors in reports table (0 is forever)", "hours")
	dbFlusherFreq   = getopt.IntLong("report-flush-freq", 0, 2, "db reports table flush frequency (all entries with report_received_at=null older than this period are deleted)", "hours")
	logDir          = getopt.StringLong("log", 0, "", "directory for log files. If empty, all log goes to stderr", "dir")
	snmpLoadAvgWin  = getopt.IntLong("load-avg-window", 'w', 30, "SNMP load avg calculation window", "sec")
	lockID          = getopt.IntLong("lock-id", 'l', 0, "pg advisory lock id to ensure single running process (0 to disable)")
)

func main() {
	getopt.FlagLong(&dispatcher.MaxLoadDelta, "max-load-delta", 0, "max load delta allowed between agents before `unsticking` a device from its agent")
	getopt.SetParameters("")
	getopt.Parse()

	if len(os.Args) == 1 {
		getopt.PrintUsage(os.Stderr)
		os.Exit(1)
	}

	glog.WithConf(glog.Conf{Verbosity: *debug, LogDir: *logDir, PrintLocation: *debug > 0})

	if *showVersion {
		fmt.Printf("Revision:%s Branch:%s Build:%s\n", Revision, Branch, Build)
		return
	}

	if !strings.HasPrefix(*dsn, "postgres://") {
		glog.Exit("pgdsn must start with `postgres://`")
	}

	if *pingBatchCount == 0 && *dbPingQueryFreq > 0 {
		glog.Exit("ping-batch-count cannot be 0 when db-ping-freq is > 0")
	}

	ctx, cancel := context.WithCancel(context.Background())
	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt, syscall.SIGTERM)
	go func() {
		select {
		case <-c:
			glog.Info("interrupted, sending cancel...")
			cancel()
		case <-ctx.Done():
		}
	}()

	dispatcher.LocalIP, dispatcher.Port = *localIP, *port
	if err := dispatcher.ConnectDB(*dsn); err != nil {
		glog.Exitf("connect db: %v", err)
	}
	defer dispatcher.ReleaseDB()

	if *lockID > 0 {
		if err := dispatcher.AcquireLock(ctx, *lockID); err != nil {
			if strings.Contains(err.Error(), "cancel") {
				return
			}
			glog.Exitf("acquire lock: %v", err)
		}
	}

	if err := dispatcher.PrepareQueries(); err != nil {
		glog.Exitf("prepare queries: %v", err)
	}

	dispatcher.LoadAvgWindow = time.Duration(*snmpLoadAvgWin) * time.Second

	if err := dispatcher.LoadAgents(); err != nil {
		glog.Exitf("error loading agents: %v", err)
	}

	if *keepAliveFreq > 0 {
		log.Debug("starting agent checker goroutine")
		go func() {
			keepAliveTick := time.NewTicker(time.Duration(*keepAliveFreq) * time.Second)
			defer keepAliveTick.Stop()
			var loops int
			for range keepAliveTick.C {
				loops++
				if loops%10 == 0 {
					// reload agents from db every 10 keep alives
					dispatcher.LoadAgents()
				}
				dispatcher.CheckAgents()
			}
		}()
	}

	if *dbSnmpQueryFreq > 0 {
		log.Debug("starting poller goroutine")
		go func() {
			pollTick := time.NewTicker(time.Duration(*dbSnmpQueryFreq) * time.Second)
			defer pollTick.Stop()
			for {
				dispatcher.SendPollingJobs(ctx)
				select {
				case <-ctx.Done():
					log.Debugf("interrupted, exiting")
					os.Exit(0)
				case <-pollTick.C:
				}
			}
		}()
	} else {
		log.Info("snmp requests disabled")
	}

	if *dbPingQueryFreq > 0 {
		dispatcher.PingBatchCount = *pingBatchCount
		log.Debug("starting pinger goroutine")
		go func() {
			pingTick := time.NewTicker(time.Duration(*dbPingQueryFreq) * time.Second)
			defer pingTick.Stop()
			for {
				dispatcher.SendPingRequests(ctx)
				select {
				case <-ctx.Done():
					log.Debugf("interrupted, exiting")
					os.Exit(0)
				case <-pingTick.C:
				}
			}
		}()
	} else {
		log.Info("ping requests disabled")
	}

	if *unlockFreq > 0 {
		log.Debug("starting device unlocker goroutine")
		go func() {
			unlockTick := time.NewTicker(time.Duration(*unlockFreq) * time.Second)
			defer unlockTick.Stop()
			for range unlockTick.C {
				dispatcher.UnlockDevices()
			}
		}()
	}

	if *dbPollErrRP > 0 {
		log.Debug("starting reports flusher goroutine")
		go func() {
			flushTick := time.NewTicker(time.Duration(*dbFlusherFreq) * time.Hour)
			defer flushTick.Stop()
			for range flushTick.C {
				dispatcher.FlushReports(*dbPollErrRP, *dbFlusherFreq)
			}
		}()
	}

	log.Debugf("starting report web server on %s:%d", *localIP, *port)
	http.HandleFunc(model.ReportURI, dispatcher.HandleReport)
	http.HandleFunc(dispatcher.DeviceListURI, dispatcher.HandleDeviceList)
	http.HandleFunc(dispatcher.DeviceCreateURI, dispatcher.HandleDeviceCreate)
	http.HandleFunc(dispatcher.DeviceUpdateURI, dispatcher.HandleDeviceUpdate)
	http.HandleFunc(dispatcher.DeviceUpsertURI, dispatcher.HandleDeviceUpsert)
	http.HandleFunc(dispatcher.DeviceDeleteURI, dispatcher.HandleDeviceDelete)
	http.HandleFunc("/-/debug", handleDebugLevel)
	logger := httplogger.CommonLogger(log.Writer{})
	glog.Fatal(http.ListenAndServe(fmt.Sprintf("%s:%d", *localIP, *port), logger(http.DefaultServeMux)))
}

func handleDebugLevel(w http.ResponseWriter, r *http.Request) {
	level := r.FormValue("level")
	if level == "" {
		fmt.Fprintf(w, "level=%d", glog.GetLevel())
		return
	}
	dbgLevel, err := strconv.Atoi(level)
	if err != nil || dbgLevel < 0 || dbgLevel > 3 {
		log.Errorf("invalid level %s", level)
		http.Error(w, "invalid debug level "+level, 400)
		return
	}
	glog.SetLevel(int32(dbgLevel))
	w.WriteHeader(http.StatusOK)
}
