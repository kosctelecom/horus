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
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"os/signal"
	"strconv"
	"syscall"

	"github.com/kosctelecom/horus/agent"
	"github.com/kosctelecom/horus/dispatcher"
	"github.com/kosctelecom/horus/model"
	_ "github.com/lib/pq"
	"github.com/vma/getopt"
	"github.com/vma/glog"
)

var (
	// Revision is the git revision, set at compilation
	Revision string

	// Build is the build time, set at compilation
	Build string

	// Branch is the git branch, set at compilation
	Branch string

	showVersion = getopt.BoolLong("version", 'v', "Print version and build date")
	jsonf       = getopt.StringLong("request", 'r', "", "request json file", "json")
	debug       = getopt.IntLong("debug", 'd', 0, "debug level")
	devID       = getopt.IntLong("id", 'i', 0, "id of the device to query")
	dsn         = getopt.StringLong("dsn", 0, "postgres://horus:horus@localhost/horus", "postgres db DSN", "url")
	compact     = getopt.BoolLong("compact", 'c', "print compacted json result")
	printQuery  = getopt.BoolLong("print-query", 'p', "print the json query before executing it")
	scalarMeas  = getopt.ListLong("scalar", 's', "id of scalar measures to query: all if empty, none if 0", "id,...")
	indexedMeas = getopt.ListLong("indexed", 't', "id of indexed measures to query: all if empty, none if 0", "id,...")
	prune       = getopt.BoolLong("prune", 0, "prune result to keep only metrics to be exported to kafka")
)

func main() {
	log.SetFlags(log.LstdFlags | log.Lmicroseconds | log.Lshortfile)
	getopt.SetParameters("")
	getopt.Parse()

	if len(os.Args) == 1 {
		getopt.PrintUsage(os.Stderr)
		os.Exit(1)
	}

	glog.WithConf(glog.Conf{Verbosity: *debug})

	if *showVersion {
		fmt.Printf("Revision:%s Branch:%s Build:%s\n", Revision, Branch, Build)
		return
	}

	if *jsonf == "" && *devID == 0 {
		log.Fatalf("ERR: json or device id required")
	}

	ctx, cancel := context.WithCancel(context.Background())
	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt, syscall.SIGTERM)
	go func() {
		select {
		case <-c:
			cancel()
		case <-ctx.Done():
			return
		}
	}()

	agent.StopCtx = ctx

	var data []byte
	var err error
	if *jsonf != "" {
		data, err = ioutil.ReadFile(*jsonf)
		if err != nil {
			log.Fatalf("ERR: read %s: %v", *jsonf, err)
		}
	}

	if *devID != 0 {
		if err = dispatcher.InitDB(*dsn); err != nil {
			log.Fatalf("ERR: init db: %v", err)
		}
		var req model.SnmpRequest
		req, err = dispatcher.RequestFromDB(*devID)
		if err != nil {
			log.Fatalf("ERR: request from db: %v", err)
		}
		dispatcher.ReleaseDB()
		data, err = json.Marshal(req)
		if err != nil {
			log.Fatalf("marshal req: %v", err)
		}
	}
	var req agent.SnmpRequest
	if err := json.Unmarshal(data, &req); err != nil {
		log.Fatalf("ERR: snmp request from json: %v", err)
	}
	if len(*scalarMeas) > 0 {
		log.Printf("filtering scalar meas to keep %v", *scalarMeas)
		var scalar []model.ScalarMeasure
		for _, meas := range req.ScalarMeasures {
			if inArray(*scalarMeas, meas.ID) {
				scalar = append(scalar, meas)
			}
		}
		req.ScalarMeasures = scalar
	}
	if len(*indexedMeas) > 0 {
		log.Printf("filtering indexed meas to keep %v", *indexedMeas)
		var indexed []model.IndexedMeasure
		for _, meas := range req.IndexedMeasures {
			if inArray(*indexedMeas, meas.ID) {
				indexed = append(indexed, meas)
			}
		}
		req.IndexedMeasures = indexed
	}
	if *printQuery {
		jq, _ := json.MarshalIndent(req, "", " ")
		log.Printf("req = %s\n--\n", jq)
	}

	if err := req.Dial(ctx); err != nil {
		log.Fatalf("Dial: %v", err)
	}
	defer req.Close()
	res := req.Poll(ctx)
	if res.PollErr != "" {
		log.Printf("Poll error: %v", res.PollErr)
		return
	}
	if *prune {
		res.PruneForKafka()
	}
	for i := range res.Indexed {
		res.Indexed[i].DedupDesc()
	}

	var payload []byte
	if *compact {
		payload, _ = json.Marshal(res)
	} else {
		payload, _ = json.MarshalIndent(res, "", "  ")
	}
	fmt.Printf("result:\n%s\n", payload)
}

// inArray checks if the given integer entry is present in the string array.
func inArray(arr []string, entry int) bool {
	sentry := strconv.Itoa(entry)
	for _, s := range arr {
		if s == sentry {
			return true
		}
	}
	return false
}
