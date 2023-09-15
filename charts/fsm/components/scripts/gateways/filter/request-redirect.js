((
  { isDebugEnabled } = pipy.solve('config.js'),

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

  resolvPath = path => (
    path && path.split('/').map(
      s => resolvVar(s)
    ).join('/')
  ),

  makeRedirectHandler = cfg => (
    msg => cfg?.statusCode ? (
      (
        scheme = cfg?.scheme || msg?.scheme || 'http',
        hostname = cfg?.hostname || msg?.host,
        path = resolvPath(cfg?.path) || msg?.path,
        port = cfg?.port,
      ) => (
        port && hostname && (
          hostname = hostname.split(':')[0] + ':' + port
        ),
        hostname && path ? (
          new Message({
            status: cfg.statusCode,
            headers: {
              Location: scheme + '://' + hostname + path
            }
          })
        ) : null
      )
    )() : null
  ),

  makeServiceRedirectHandler = svc => (
    (svc?.Filters || []).filter(
      e => e?.Type === 'RequestRedirect'
    ).map(
      e => makeRedirectHandler(e)
    ).filter(
      e => e
    )?.[0]
  ),

  filterCache = new algo.Cache(
    route => (
      (
        config = route?.config,
        backendService = config?.BackendService,
      ) => (
        new algo.Cache(
          service => (
            makeServiceRedirectHandler(backendService?.[service]) || makeServiceRedirectHandler(config)
          )
        )
      )
    )()
  ),

) => pipy({
  _redirectHandler: null,
  _redirectMessage: null,
})

.import({
  __route: 'route',
  __service: 'service',
  __http: 'http',
  __consumer: 'consumer',
})

.pipeline()
.onStart(
  () => void (
    _redirectHandler = filterCache.get(__route)?.get?.(__service?.name)
  )
)
.branch(
  () => _redirectHandler, (
    $=>$.handleMessageStart(
      msg => (
        _redirectMessage = _redirectHandler(msg?.head?.headers)
      )
    )
    .branch(
      () => _redirectMessage, (
        $=>$.replaceMessage(
          msg => (
            isDebugEnabled && (
              console.log('[request-redirect] messages:', msg, _redirectMessage)
            ),
            _redirectMessage
          )
        )
      ), (
        $=>$.chain()
      )
    )
  ), (
    $=>$.chain()
  )
)

)()