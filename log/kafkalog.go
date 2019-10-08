package log

import (
	"fmt"

	"github.com/vma/glog"
)

// Klogger extends glog.Logger and implements
// the github.com/optiopay/kafka.Logger interface.
type Klogger struct {
	*glog.Logger
}

// Debug logs a Klogger debug message
func (l Klogger) Debug(msg string, args ...interface{}) {
	if glog.V(1) {
		glog.InfoDepth(1, join(msg, args...))
	}
}

// Info logs a Klogger info message
func (l Klogger) Info(msg string, args ...interface{}) {
	glog.InfoDepth(1, join(msg, args...))
}

// Warn logs a Klogger warning message
func (l Klogger) Warn(msg string, args ...interface{}) {
	glog.WarningDepth(1, join(msg, args...))
}

// Error logs a Klogger error message
func (l Klogger) Error(msg string, args ...interface{}) {
	glog.ErrorDepth(1, join(msg, args...))
}

func join(msg string, args ...interface{}) (s string) {
	s += "kafka: " + msg + ": "
	for i, arg := range args {
		s += fmt.Sprintf("%v", arg)
		if i < len(args)-1 {
			s += " "
		}
	}
	return s
}
