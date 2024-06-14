((
  { config, isDebugEnabled } = pipy.solve('config.js'),

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

  proxyRedirectCache = new algo.Cache(
    route => (
      (
        pr = route?.config?.ProxyRedirect || __domain?.ProxyRedirect || config?.Configs?.ProxyRedirect
      ) => (
        pr && Object.entries(pr)
      )
    )()
  ),

) => pipy({
  _proxyRedirect: null,
  _location: null,
  _refresh: null,
  _from: null,
  _to: null,
})

.import({
  __http: 'http',
  __domain: 'route',
  __route: 'route',
  __consumer: 'consumer',
})

.pipeline()
.onStart(
  () => void (
    _proxyRedirect = proxyRedirectCache.get(__route)
  )
)
.chain()
.branch(
  () => _proxyRedirect, (
    $=>$
    .handleMessageStart(
      msg => (
        msg?.head?.headers?.location && (
          _location = msg.head.headers.location,
          _proxyRedirect.find(([k, v]) => _location.startsWith(k) && (_from = k, _to = v)) && (
            _to = resolvPath(_to),
            msg.head.headers.location = _location.replace(_from, _to)
          )
        ),
        msg?.head?.headers?.refresh && (
          _refresh = msg.head.headers.refresh,
          _proxyRedirect.find(([k, v]) => (_refresh.indexOf('url=' + k) >= 0) && (_from = k, _to = v)) && (
            _to = resolvPath(_to),
            msg.head.headers.refresh = _refresh.replace(_from, _to)
          )
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
      msg => _proxyRedirect && (
        console.log('[proxy-redirect] location, refresh, response:', _location, _refresh, msg)
      )
    )
  )
)

)()