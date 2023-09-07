package utils

func PipyLogLevelByVerbosity(verbosity string) string {
	switch verbosity {
	case "trace":
		return "debug"
	case "fatal":
		return "error"
	case "panic":
		return "error"
	}

	return verbosity
}
