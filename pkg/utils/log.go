package utils

func PipyLogLevelByVerbosity(verbosity string) string {
	switch verbosity {
	case "trace":
		return "debug"
	case "fatal":
		return "error"
	case "panic":
		return "error"
	case "debug", "info", "warn", "error":
		return verbosity
	default:
		// default to error if verbosity is not recognized
		return "error"
	}
}
