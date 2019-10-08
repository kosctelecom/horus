package log

// Writer is a Writer used by httplogger.CommonLogger for http logging
type Writer struct{}

// Write implements the writer interface and
// sends the input to the logger at debug level.
func (l Writer) Write(b []byte) (int, error) {
	Debug(string(b))
	return len(b), nil
}
