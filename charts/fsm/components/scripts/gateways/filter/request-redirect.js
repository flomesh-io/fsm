((
  { isDebugEnabled } = pipy.solve('config.js'),

  makeRedirectHandler = cfg => (
    msg => cfg?.statusCode ? (
      (
        scheme = cfg?.scheme || msg?.scheme || 'http',
        hostname = cfg?.hostname || msg?.host,
        path = cfg?.path || msg?.path,
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

  redirectHandlers = new algo.Cache(makeRedirectHandler),

  makeServiceRedirectHandler = svc => (
    (svc?.Filters || []).filter(
      e => e?.Type === 'RequestRedirect'
    ).map(
      e => redirectHandlers.get(e)
    ).filter(
      e => e
    )?.[0]
  ),

  serviceRedirectHandlers = new algo.Cache(makeServiceRedirectHandler),

) => pipy({
  _redirectHandler: null,
  _redirectMessage: null,
 })

.import({
  __service: 'service',
})

.pipeline()
.onStart(
  () => void (
    _redirectHandler = serviceRedirectHandlers.get(__service)
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