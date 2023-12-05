((
  { config, isDebugEnabled } = pipy.solve('config.js'),

  makeServiceHandler = serviceName => (
    config?.Services?.[serviceName] ? (
      config.Services[serviceName].name = serviceName,
      config.Services[serviceName]
    ) : null
  ),

  serviceHandlers = new algo.Cache(makeServiceHandler),

  resolvVar = val => (
    val?.startsWith('$') ? (
      (
        pos = val.indexOf('_'),
        name,
        member,
        content = val,
      ) => (
        (pos > 0) && (
          name = val.substring(1, pos),
          member = val.substring(pos + 1),
          (name === 'http') && (
            content = __http?.headers?.[member] || __http?.[member] || val
          ) || (name === 'consumer') && (
            content = __consumer?.[member] || val
          )
        ),
        content
      )
    )() : val
  ),

) => pipy({
  _xff: null,
  _serviceName: null,
  _unauthorized: undefined,
})

.export('service', {
  __service: null,
})

.import({
  __http: 'http',
  __route: 'route',
  __root: 'web-server',
  __consumer: 'consumer',
})

.pipeline()
.handleMessageStart(
  msg => (
    __route?.config?.EnableHeadersAuthorization && (
      (!__consumer || !__consumer?.['Headers-Authorization']) ? (_unauthorized = true) : (_unauthorized = false)
    ),
    __route?.virtualService ? (
      _serviceName = resolvVar(__route.virtualService),
      !(__service = serviceHandlers.get(_serviceName)) && config?.Services && (
        config.Services[_serviceName] = { "Endpoints": { [_serviceName]: { "Weight": 1 } } },
        serviceHandlers.remove(_serviceName),
        __service = serviceHandlers.get(_serviceName)
      )
    ) :
    __route?.serverRoot ? (
      __root = __route.serverRoot
    ) : (
      (_serviceName = __route?.backendServiceBalancer?.borrow?.({})?.id) && (
        (__service = serviceHandlers.get(_serviceName)) && msg?.head?.headers && (
          (_xff = msg.head.headers['x-forwarded-for']) ? (
            msg.head.headers['x-forwarded-for'] = _xff + ', ' + __inbound.localAddress
          ) : (
            msg.head.headers['x-forwarded-for'] = __inbound.localAddress
          )
        )
      )
    )
  )
)
.branch(
  isDebugEnabled, (
    $=>$.handleStreamStart(
      () => (
        console.log('[service] name, root, endpoints, unauthorized:', _serviceName, __root, Object.keys(__service?.Endpoints || {}), _unauthorized)
      )
    )
  )
)
.branch(
  () => _unauthorized, (
    $=>$.replaceMessage(
      () => (
        __route?.config?.HeadersAuthorizationType === 'Basic' ? (
          new Message({ status: 401, headers: { 'WWW-Authenticate': 'Basic realm=fgw' } })
        ) : new Message({ status: 401 })
      )
    )
  ),
  () => __root, (
    $=>$
    .use('http/error-page.js', 'request')
    .use('server/web-server.js')
    .use('http/error-page.js', 'response')
  ),
  () => __service, (
    $=>$.chain()
  ), (
    $=>$.use('http/default.js')
  )
)

)()