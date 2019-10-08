package glog

type Logger struct {
	*loggingT
	prefix string
	parg   []interface{}
	pfmt   string
}

func New() *Logger {
	return &Logger{
		loggingT: &logging,
	}
}

func WithPrefix(pfx string) *Logger {
	var pfmt string
	var parg []interface{}

	if pfx != "" {
		parg = []interface{}{pfx, "-"}
		pfmt = pfx + " - "
	}
	return &Logger{
		loggingT: &logging,
		prefix:   pfx,
		parg:     parg,
		pfmt:     pfmt,
	}
}

func (l *Logger) Debug(lvl Level, args ...interface{}) {
	if V(lvl) {
		args = append(l.parg, args...)
		logging.println(infoLog, args...)
	}
}

func (l *Logger) Debugf(lvl Level, format string, args ...interface{}) {
	if V(lvl) {
		logging.printf(infoLog, l.pfmt+format, args...)
	}
}

func (l *Logger) Info(args ...interface{}) {
	args = append(l.parg, args...)
	logging.println(infoLog, args...)
}

func (l *Logger) Infof(format string, args ...interface{}) {
	logging.printf(infoLog, l.pfmt+format, args...)
}

func (l *Logger) InfoDepth(depth int, args ...interface{}) {
	args = append(l.parg, args...)
	logging.printDepth(infoLog, depth, args...)
}

func (l *Logger) InfofDepth(depth int, format string, args ...interface{}) {
	logging.printfDepth(infoLog, depth, l.pfmt+format, args...)
}

func (l *Logger) Warn(args ...interface{}) {
	args = append(l.parg, args...)
	logging.println(warningLog, args...)
}

func (l *Logger) Warnf(format string, args ...interface{}) {
	logging.printf(warningLog, l.pfmt+format, args...)
}

func (l *Logger) Warning(args ...interface{}) {
	args = append(l.parg, args...)
	logging.println(warningLog, args...)
}

func (l *Logger) Warningf(format string, args ...interface{}) {
	logging.printf(warningLog, l.pfmt+format, args...)
}

func (l *Logger) Error(args ...interface{}) {
	args = append(l.parg, args...)
	logging.println(errorLog, args...)
}

func (l *Logger) Errorf(format string, args ...interface{}) {
	logging.printf(errorLog, l.pfmt+format, args...)
}
