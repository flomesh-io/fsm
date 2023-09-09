((
  { config } = pipy.solve('config.js'),

  serviceCache = new algo.Cache(
    serviceName => (
      config?.Services?.[serviceName] ? (
        (
          serviceConfig = Object.assign({}, config.Services[serviceName]),
        ) => (
          serviceConfig.name = serviceName,
          serviceConfig.RetryPolicy = { NumRetries: 0 },
          serviceConfig
        )
      )() : null
    )
  ),

  randomInt63 = () => (
    new Int("u64", new Data(algo.uuid().substring(0, 18).replaceAll('-', ''), "hex").toArray()).toString() / 2
  ),
  samplingRange = fraction => (fraction > 0 ? fraction : 0) * Math.pow(2, 63),
  configCache = new algo.Cache(
    pluginConfig => pluginConfig && (
      {
        samplingRange: pluginConfig?.percentage > 0 ? samplingRange(pluginConfig.percentage) : 0,
        serviceName: pluginConfig?.serviceName,
        host: pluginConfig?.host,
      }
    )
  ),
) => pipy({
  _pluginName: '',
  _pluginConfig: null,
  _mirrorConfig: null,
  _randomVal: 0,
  _mirrorService: undefined,
})
.import({
  __service: 'service',
})

.pipeline()
.onStart(
  () => void (
    _pluginName = __filename.slice(9, -3),
    _pluginConfig = __service?.Plugins?.[_pluginName],
    (_mirrorConfig = configCache.get(_pluginConfig)) && (
      _mirrorService = serviceCache.get(_mirrorConfig.serviceName)
    )
  )
)
.handleMessageStart(
  () => (
    _mirrorService && (
      _randomVal = randomInt63(),
      (_randomVal < _mirrorConfig.samplingRange) || (
        _mirrorService = undefined
      )
    )
  )
)
.branch(
  () => _mirrorService, (
    $=>$
    .fork().to('mirror-service')
    .chain()
  ), (
    $=>$.chain()
  )
)

.pipeline('mirror-service')
.replaceMessage(
  msg => (
    (
      mirrorMsg = new Message(Object.assign({}, msg.head), msg.body),
    ) => (
      _mirrorConfig.host && (
        mirrorMsg.head.headers = Object.assign({}, msg.head.headers),
        mirrorMsg.head.headers.host = _mirrorConfig.host
      ),
      [mirrorMsg]
    )
  )()
)
.demux().to(
  $=>$
  .handleMessageStart(
    () => (
      __service = _mirrorService
    )
  )
  .chain([
    "http/forward.js",
    "http/default.js"
  ])
)
.dummy()
)()