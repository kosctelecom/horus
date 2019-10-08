// Go support for leveled logs, analogous to https://code.google.com/p/google-glog/
//
// Copyright 2013 Google Inc. All Rights Reserved.
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

// File I/O for logs.

package glog

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"
)

var (
	pid     = os.Getpid()
	program = filepath.Base(os.Args[0])
	host    = "unknownhost"
)

func init() {
	h, err := os.Hostname()
	if err == nil {
		host = shortHostname(h)
	}
}

// shortHostname returns its argument, truncating at the first period.
// For instance, given "www.google.com" it returns "www".
func shortHostname(hostname string) string {
	if i := strings.Index(hostname, "."); i >= 0 {
		return hostname[:i]
	}
	return hostname
}

// logName returns a new log file name with start time t
func logName(t time.Time) string {
	name := fmt.Sprintf("%s.%s.%d.%04d%02d%02dT%02d%02d%02d.log",
		program,
		host,
		pid,
		t.Year(),
		t.Month(),
		t.Day(),
		t.Hour(),
		t.Minute(),
		t.Second())
	return name
}

// create creates a new log file and returns the file and its filename.
func create(t time.Time) (*os.File, string, error) {
	if logDir == "" {
		return nil, "", errors.New("log: no log dirs")
	}
	name := logName(t)
	fname := filepath.Join(logDir, name)
	f, err := os.Create(fname)
	if err != nil {
		return nil, "", fmt.Errorf("log: cannot create file: %v", err)
	}
	return f, fname, nil
}
