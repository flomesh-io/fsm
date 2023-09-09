package utils

const (
	logLevelTrace = "trace"
	logLevelDebug = "debug"
	logLevelInfo  = "info"
	logLevelWarn  = "warn"
	logLevelError = "error"
	logLevelFatal = "fatal"
	logLevelPanic = "panic"
)

func PipyLogLevelByVerbosity(verbosity string) string {
	switch verbosity {
	case logLevelTrace:
		return logLevelDebug
	case logLevelFatal:
		return logLevelError
	case logLevelPanic:
		return logLevelError
	case logLevelDebug, logLevelInfo, logLevelWarn, logLevelError:
		return verbosity
	default:
		// default to error if verbosity is not recognized
		return logLevelError
	}
}
