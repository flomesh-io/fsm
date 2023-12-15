((
  { config, isDebugEnabled } = pipy.solve('config.js'),

  makeServiceHandler = serviceName => (
    config?.Services?.[serviceName] ? (
      config.Services[serviceName].name = serviceName,
      config.Services[serviceName]
    ) : null
  ),

  serviceHandlers = new algo.Cache(makeServiceHandler),

) => pipy({
  _xff: null,
  _serviceName: null,
  _unauthorized: undefined,
})

.export('service', {
  __service: null,
})

.import({
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