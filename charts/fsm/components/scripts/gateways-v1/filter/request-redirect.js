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

  makePathHandle = (path, cfg) => (
    (cfg?.Path?.Type === 'ReplacePrefixMatch') ? (
      (cfg?.Path?.ReplacePrefixMatch !== undefined) && (
        head => (
          head?.path?.length > path.length ? (
            head.path = resolvPath(cfg.Path.ReplacePrefixMatch) + head.path.substring(path.length)
          ) : (
            head.path = resolvPath(cfg.Path.ReplacePrefixMatch)
          )
        )
      )
    ) : (cfg?.Path?.Type === 'ReplaceFullPath') && (
      (cfg?.Path?.ReplaceFullPath !== undefined) && (
        head => (
          (
            prefix = (head?.path || '').split('?')[0],
            suffix = (head?.path || '').substring(prefix.length),
          ) => (
            head.path = resolvPath(cfg.Path.ReplaceFullPath) + suffix
          )
        )()
      )
    )
  ),

  makeRedirectHandler = (path, cfg) => (
    head => cfg?.StatusCode ? (
      (
        scheme = cfg?.Scheme || head?.scheme || 'http',
        hostname = cfg?.Hostname || head?.headers?.host,
        pathHandle = makePathHandle(path, cfg),
        port = cfg?.Port,
      ) => (
        pathHandle(head),
        port && hostname && (
          hostname = hostname.split(':')[0] + ':' + port
        ),
        new Message({
          status: cfg.StatusCode,
          headers: {
            Location: scheme + '://' + hostname + head.path
          }
        })
      )
    )() : null
  ),

  makeServiceRedirectHandler = (path, cfg) => (
    (cfg?.Filters || []).filter(
      e => e?.Type === 'RequestRedirect'
    ).map(
      e => makeRedirectHandler(path, e.RequestRedirect)
    ).filter(
      e => e
    )?.[0]
  ),

  filterCache = new algo.Cache(
    route => (
      (
        config = route?.config,
        path = config?.Path?.Path || '/',
        backendService = config?.BackendService,
      ) => (
        new algo.Cache(
          service => (
            makeServiceRedirectHandler(path, backendService?.[service]) || makeServiceRedirectHandler(path, config)
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
        _redirectMessage = _redirectHandler(msg?.head)
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