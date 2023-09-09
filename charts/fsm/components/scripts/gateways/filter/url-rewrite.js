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

  getPrefix = uri => (
    (
      path = uri?.split?.('?')[0] || '',
      elts = path.split('/'),
    ) => (
      (elts[0] === '' && elts.length > 1) ? ('/' + (elts[1] || '')) : elts[0]
    )
  )(),

  pathPrefix = (path, prefix) => (
    path.startsWith(prefix) && (
      prefix.endsWith('/') || (
        (
          lastChar = path.charAt(prefix.length),
        ) => (
          lastChar === '' || lastChar === '/'
        )
      )()
    )
  ),

  makeHeadHandler = cfg => (
    (cfg?.path?.type === 'ReplacePrefixMatch') ? (
      head => (
        (
          match = pathPrefix(head?.path, cfg?.path?.replacePrefixMatch),
          suffix = match && (head?.path || '').substring(cfg.path.replacePrefixMatch.length),
          replace = resolvVar(cfg?.path?.replacePrefix || '/'),
        ) => (
          match && (
            replace.endsWith('/') ? (
              suffix.startsWith('/') ? (
                head.path = replace + suffix.substring(1)
              ) : (
                head.path = replace + suffix
              )
            ) : (
              suffix.startsWith('/') ? (
                head.path = replace + suffix
              ) : (
                head.path = replace + '/' + suffix
              )
            ),
            cfg?.hostname && head.headers && (
              head.headers.host = cfg.hostname
            )
          )
        )
      )()
    ) : (
      (cfg?.path?.type === 'ReplaceFullPath') ? (
        head => (
          (
            prefix = (head?.path || '').split('?')[0],
            suffix = (head?.path || '').substring(prefix.length),
          ) => (
            head.path = resolvVar(cfg?.path?.replaceFullPath) + suffix,
            cfg?.hostname && head.headers && (
              head.headers.host = cfg.hostname
            )
          )
        )()
      ) : null
    )
  ),

  headHandlers = new algo.Cache(makeHeadHandler),

  makeRewriteHandler = cfg => (
    (
      handlers = (cfg?.Filters || []).filter(
        e => e?.Type === 'HTTPURLRewriteFilter'
      ).map(
        e => headHandlers.get(e)
      ).filter(
        e => e
      )
    ) => (
      handlers.length > 0 ? handlers : null
    )
  )(),

  rewriteHandlersCache = new algo.Cache(makeRewriteHandler),

) => pipy({
  _rewriteHandlers: null,
})

.import({
  __service: 'service',
  __http: 'http',
  __consumer: 'consumer',
})

.pipeline()
.onStart(
  () => void (
    _rewriteHandlers = rewriteHandlersCache.get(__service)
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