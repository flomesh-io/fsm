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
            content = _http?.headers?.[member] || _http?.[member] || val
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

  makeModifierHandler = cfg => (
    (
      set = cfg?.set,
      add = cfg?.add,
      remove = cfg?.remove,
    ) => (
      (set || add || remove) && (
        msg => (
          _http = (cfg.Type === 'RequestHeaderModifier') ? __request?.head : __response?.head,
          set && set.forEach(
            e => (msg[e.name] = resolvPath(e.value))
          ),
          add && add.forEach(
            e => (
              msg[e.name] ? (
                msg[e.name] = msg[e.name] + ',' + resolvPath(e.value)
              ) : (
                msg[e.name] = resolvPath(e.value)
              )
            )
          ),
          remove && remove.forEach(
            e => delete msg[e]
          )
        )
      )
    )
  )(),

  makeRequestModifierHandler = cfg => (
    (
      handlers = (cfg?.Filters || []).filter(
        e => e?.Type === 'RequestHeaderModifier'
      ).map(
        e => makeModifierHandler(e)
      ).filter(
        e => e
      )
    ) => (
      handlers.length > 0 ? handlers : null
    )
  )(),

  requestFilterCache = new algo.Cache(
    route => (
      (
        config = route?.config,
        backendService = config?.BackendService,
      ) => (
        new algo.Cache(
          service => (
            makeRequestModifierHandler(backendService?.[service]) || makeRequestModifierHandler(config)
          )
        )
      )
    )()
  ),

  makeResponseModifierHandler = cfg => (
    (
      handlers = (cfg?.Filters || []).filter(
        e => e?.Type === 'ResponseHeaderModifier'
      ).map(
        e => makeModifierHandler(e)
      ).filter(
        e => e
      )
    ) => (
      handlers.length > 0 ? handlers : null
    )
  )(),

  responseFilterCache = new algo.Cache(
    route => (
      (
        config = route?.config,
        backendService = config?.BackendService,
      ) => (
        new algo.Cache(
          service => (
            makeResponseModifierHandler(backendService?.[service]) || makeResponseModifierHandler(config)
          )
        )
      )
    )()
  ),

) => pipy({
  _http: null,
  _requestHandlers: null,
  _responseHandlers: null,
})

.import({
  __route: 'route',
  __service: 'service',
  __request: 'http',
  __response: 'http',
  __consumer: 'consumer',
})

.pipeline()
.onStart(
  () => void (
    _requestHandlers = requestFilterCache.get(__route)?.get?.(__service?.name),
    _responseHandlers = responseFilterCache.get(__route)?.get?.(__service?.name)
  )
)
.branch(
  () => _requestHandlers, (
    $=>$.handleMessageStart(
      msg => (
        msg?.head?.headers && _requestHandlers.forEach(
          e => e(msg.head.headers)
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
        console.log('[header-modifier] request message:', msg)
      )
    )
  )
)
.chain()
.branch(
  () => _responseHandlers, (
    $=>$.handleMessageStart(
      msg => (
        msg?.head?.headers && _responseHandlers.forEach(
          e => e(msg.head.headers)
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
        console.log('[header-modifier] response message:', msg)
      )
    )
  )
)

)()