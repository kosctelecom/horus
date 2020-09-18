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

package log

import (
	"fmt"

	"github.com/vma/glog"
)

// Logger extends glog.Logger and implements the gosnmp.Logger interface.
type Logger struct {
	*glog.Logger
}

// WithPrefix returns a new Logger with the given prefix
func WithPrefix(pfx string) Logger {
	return Logger{glog.WithPrefix(pfx)}
}

// Print implements the gosnmp.Logger interface.
// Logs the gosnmp messages at debug level 4.
func (l Logger) Print(v ...interface{}) {
	l.Debug(glog.Level(4), v...)
}

// Printf implements the gosnmp.Logger interface.
// Logs the gosnmp messages at debug level 4.
func (l Logger) Printf(fmt string, v ...interface{}) {
	l.Debugf(glog.Level(4), fmt, v...)
}

// Info prints the input at info level.
func Info(args ...interface{}) {
	glog.InfoDepth(1, args...)
}

// Infof prints the formatted input at info level.
func Infof(format string, args ...interface{}) {
	glog.InfofDepth(1, format, args...)
}

// Debug prints the input at debug level 1.
func Debug(args ...interface{}) {
	if glog.V(1) {
		glog.InfoDepth(1, args...)
	}
}

// Debugf prints the formatted input at debug level 1.
func Debugf(format string, args ...interface{}) {
	if glog.V(1) {
		glog.InfofDepth(1, format, args...)
	}
}

// Debug2 prints the input at debug level 2.
func Debug2(args ...interface{}) {
	if glog.V(2) {
		glog.InfoDepth(1, args...)
	}
}

// Debug2f prints the formatted input at debug level 2.
func Debug2f(format string, args ...interface{}) {
	if glog.V(2) {
		glog.InfofDepth(1, format, args...)
	}
}

// Debug3 prints the input at debug level 3.
func Debug3(args ...interface{}) {
	if glog.V(3) {
		glog.InfoDepth(1, args...)
	}
}

// Debug3f prints the formatted input at debug level 3.
func Debug3f(format string, args ...interface{}) {
	if glog.V(3) {
		glog.InfofDepth(1, format, args...)
	}
}

// Warning prints the input at warning level.
func Warning(args ...interface{}) {
	glog.WarningDepth(1, args...)
}

// Warningf prints the formatted input at warning level.
func Warningf(format string, args ...interface{}) {
	glog.WarningfDepth(1, format, args...)
}

// Error prints the input at error level.
func Error(args ...interface{}) {
	glog.ErrorDepth(1, args...)
}

// Errorf prints the formatted input at error level.
func Errorf(format string, args ...interface{}) {
	glog.ErrorfDepth(1, format, args...)
}

func Exitf(format string, args ...interface{}) {
	glog.ExitDepth(1, fmt.Sprintf(format, args...))
}
