package routes

func isSupportedAppProtocol(appProtocol string, allowedAppProtocols []string) bool {
	for _, allowed := range allowedAppProtocols {
		if appProtocol == allowed {
			return true
		}
	}

	return false
}
