((
  { config, isDebugEnabled } = pipy.solve('config.js'),

  serviceCache = new algo.Cache(
    serviceName => (
      config?.Services?.[serviceName] ? (
        (
          serviceConfig = Object.assign({}, config.Services[serviceName]),
        ) => (
          serviceConfig.name = serviceName,
          serviceConfig.Retry = { NumRetries: 0 },
          serviceConfig
        )
      )() : null
    )
  ),

  makeMirrorServices = cfg => (
    (
      services = (cfg?.Filters || []).filter(
        e => e?.Type === 'RequestMirror'
      ).map(
        e => serviceCache.get(e.BackendService)
      ).filter(
        e => e
      )
    ) => (
      services.length > 0 ? services : null
    )
  )(),

  filterCache = new algo.Cache(
    route => (
      (
        config = route?.config,
        backendService = config?.BackendService,
      ) => (
        new algo.Cache(
          service => (
            makeMirrorServices(backendService?.[service]) || makeMirrorServices(config)
          )
        )
      )
    )()
  ),

) => pipy({
  _mirrorServices: null,
})
.import({
  __route: 'route',
  __service: 'service',
})

.pipeline()
.onStart(
  () => void (
    _mirrorServices = filterCache.get(__route)?.get?.(__service?.name)
  )
)
.branch(
  isDebugEnabled, (
    $=>$.handleMessageStart(
      () => _mirrorServices && (
        console.log('[request-mirror] backendService, mirrorServices:', __service?.name, _mirrorServices.map(svc => svc.name))
      )
    )
  )
)
.branch(
  () => _mirrorServices, (
    $=>$
    .fork()
    .to('mirror-service')
    .chain()
  ), (
    $=>$.chain()
  )
)
.pipeline('mirror-service')
.replaceMessage(
  msg => (
    _mirrorServices.map(
      service => (
        new Message({ '@service': service, ...msg.head, headers: { ...msg.head.headers } }, msg.body)
      )
    )
  )
)
.demux().to(
  $=>$
  .handleMessageStart(
    msg => (
      __service = msg.head['@service'],
      delete msg.head['@service']
    )
  )
  .chain([
    "http/forward.js",
    "http/default.js"
  ])
)
.dummy()

)()