((
  { config, socketTimeoutOptions } = pipy.solve('config.js'),
  { metrics } = pipy.solve('lib/metrics.js'),
  listeners = {},
  listenPort = 0,
) => pipy()

.export('listener', {
  __port: null,
})

.branch(
  config?.Configs?.PidFile, (
    $=>$
    .task()
    .onStart(
      () => void (
        os.writeFile(config.Configs.PidFile, '' + pipy.pid)
      )
    )
    .exit()
    .onStart(
      () => void (
        os.unlink(config.Configs.PidFile)
      )
    )
  )
)
.task()
.onStart(
  () => void (
    metrics.fgwMetaInfo.withLabels(pipy.uuid || '', pipy.name || '', pipy.source || '', os.env.PIPY_K8S_CLUSTER || '').increase()
  )
)
.branch(
  (config?.Configs?.ResourceUsage?.ScrapeInterval > 0), (
    $=>$
    .task()
    .use('common/resource-usage.js')
  )
)

.repeat(
  (config.Listeners || []),
  ($, l) => $.listen(
    (
      listenPort = (l.Listen || l.Port || 0),
      listeners[listenPort] = new ListenerArray([{ ...socketTimeoutOptions, ...l, port: listenPort, protocol: (l.Protocol?.toLowerCase?.() === 'udp') ? 'udp' : 'tcp' }]),
      listeners[listenPort]
    )
  )
  .onStart(
    () => (
      __port = l,
      new Data
    )
  )
  .link('launch')
)

.pipeline('launch')
.branch(
  () => (__port?.Protocol === 'HTTP'), (
    $=>$.chain(config?.Chains?.HTTPRoute || [])
  ),
  () => (__port?.Protocol === 'HTTPS'), (
    $=>$.chain(config?.Chains?.HTTPSRoute || [])
  ),
  () => (__port?.Protocol === 'TLS' && __port?.TLS?.TLSModeType === 'Passthrough'), (
    $=>$.chain(config?.Chains?.TLSPassthrough || [])
  ),
  () => (__port?.Protocol === 'TLS' && __port?.TLS?.TLSModeType === 'Terminate'), (
    $=>$.chain(config?.Chains?.TLSTerminate || [])
  ),
  () => (__port?.Protocol === 'TCP'), (
    $=>$.chain(config?.Chains?.TCPRoute || [])
  ),
  (
    $=>$.replaceStreamStart(new StreamEnd)
  )
)

.task()
.onStart(new Data)
.use('common/health-check.js')

)()
