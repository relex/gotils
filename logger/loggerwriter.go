package logger

type logWriter struct {
	lg    Logger
	level LogLevel
}

func newLogWriter(lg Logger, level LogLevel) *logWriter {
	return &logWriter{lg, level}
}

func (lw *logWriter) Write(p []byte) (n int, err error) {
	s := string(p)
	switch lw.level {
	case PanicLevel:
		lw.lg.Panic(s)
	case FatalLevel:
		lw.lg.Fatal(s)
	case ErrorLevel:
		lw.lg.Error(s)
	case WarnLevel:
		lw.lg.Warn(s)
	case InfoLevel:
		lw.lg.Info(s)
	case DebugLevel:
		lw.lg.Debug(s)
	case TraceLevel:
		lw.lg.Trace(s)
	}
	return len(s), nil
}
