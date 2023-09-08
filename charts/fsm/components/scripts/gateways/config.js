((
  config = JSON.decode(pipy.load('config.json')),
  routeRules = {},
  hostRules = null,
) => (
  Object.keys(config?.RouteRules || {}).forEach(
    ports => (
      hostRules = {},
      Object.keys(config.RouteRules[ports] || {}).forEach(
        hosts => (
          hosts.split(',').forEach(
            host => (hostRules[host.trim()] = config.RouteRules[ports][hosts])
          )
        )
      ),
      config.RouteRules[ports] = hostRules
    )
  ),
  Object.keys(config?.RouteRules || {}).forEach(
    ports => (
      ports.split(',').forEach(
        port => (routeRules[port.trim()] = config.RouteRules[ports])
      )
    )
  ),
  config.RouteRules = routeRules,
  {
    config,
    isDebugEnabled: Boolean(config?.Configs?.EnableDebug),
    socketTimeoutOptions: (config?.Configs?.SocketTimeout > 0) ? (
      {
        connectTimeout: config.Configs.SocketTimeout,
        readTimeout: config.Configs.SocketTimeout,
        writeTimeout: config.Configs.SocketTimeout,
        idleTimeout: config.Configs.SocketTimeout,
      }
    ) : {},
  }
))()