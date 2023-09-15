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

  makeHeadHandler = (path, cfg) => (
    (cfg?.path?.type === 'ReplacePrefixMatch') ? (
      cfg?.path?.replacePrefixMatch && (
        head => (
          head?.path?.length > path.length ? (
            head.path = resolvPath(cfg.path.replacePrefixMatch) + head.path.substring(path.length)
          ) : (
            head.path = resolvPath(cfg.path.replacePrefixMatch)
          ),
          cfg?.hostname && head.headers && (
            head.headers.host = cfg.hostname
          )
        )
      )
    ) : (
      (cfg?.path?.type === 'ReplaceFullPath') ? (
        cfg?.path?.replaceFullPath && (
          head => (
            (
              prefix = (head?.path || '').split('?')[0],
              suffix = (head?.path || '').substring(prefix.length),
            ) => (
              head.path = resolvPath(cfg.path.replaceFullPath) + suffix,
              cfg?.hostname && head.headers && (
                head.headers.host = cfg.hostname
              )
            )
          )()
        )
      ) : null
    )
  ),

  makeRewriteHandler = (path, cfg) => (
    (
      handlers = (cfg?.Filters || []).filter(
        e => e?.Type === 'URLRewrite'
      ).map(
        e => makeHeadHandler(path, e)
      ).filter(
        e => e
      )
    ) => (
      handlers.length > 0 ? handlers : null
    )
  )(),

  filterCache = new algo.Cache(
    route => (
      (
        config = route?.config,
        path = config?.Path?.Path || '/',
        backendService = config?.BackendService,
      ) => (
        new algo.Cache(
          service => (
            makeRewriteHandler(path, backendService?.[service]) || makeRewriteHandler(path, config)
          )
        )
      )
    )()
  ),

) => pipy({
  _rewriteHandlers: null,
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
    _rewriteHandlers = filterCache.get(__route)?.get?.(__service?.name)
  )
)
.branch(
  () => _rewriteHandlers, (
    $=>$.handleMessageStart(
      msg => (
        msg?.head?.headers && _rewriteHandlers.forEach(
          e => e(msg.head)
        )
      )
    )
  ), (
    $=>$
  )
)
.branch(
  isDebugEnabled, (
    $=>$.handleStreamStart(
      msg => (
        console.log('[url-rewrite] message:', msg)
      )
    )
  )
)
.chain()

)()