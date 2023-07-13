((
  config = pipy.solve('config.js'),
) => pipy()

.branch(
  Boolean(config?.Inbound?.TrafficMatches), (
    $=>$
    .listen(15003, { transparent: true })
    .onStart(() => new Data)
    .use('modules/inbound-main.js')
  )
)

.branch(
  Boolean(config?.Outbound || config?.Spec?.Traffic?.EnableEgress), (
    $=>$
    .listen(15001, { transparent: true })
    .onStart(() => new Data)
    .use('modules/outbound-main.js')
  )
)

.branch(
  config?.Spec?.Probes?.LivenessProbes?.[0]?.httpGet?.port === 15901,
  $=>$.listen(15901).use('probes.js', 'liveness')
)

.branch(
  config?.Spec?.Probes?.ReadinessProbes?.[0]?.httpGet?.port === 15902,
  $=>$.listen(15902).use('probes.js', 'readiness')
)

.branch(
  config?.Spec?.Probes?.StartupProbes?.[0]?.httpGet?.port === 15903,
  $=>$.listen(15903).use('probes.js', 'startup')
)

.listen(15010)
.use('stats.js', 'prometheus')

.listen(':::15000')
.use('stats.js', 'fsm-stats')

//
// Local DNS server
//
.branch(
  Boolean(os.env.LOCAL_DNS_PROXY), (
    $=>$
    .listen('127.0.0.153:5300', { protocol: 'udp' })
    .chain(['dns-main.js'])
  )
)

)()
