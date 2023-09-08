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

  makeModifierHandler = cfg => (
    (
      set = cfg?.set,
      add = cfg?.add,
      remove = cfg?.remove,
    ) => (
      (set || add || remove) && (
        msg => (
          set && set.forEach(
            e => (msg[e.name] = resolvVar(e.value))
          ),
          add && add.forEach(
            e => (
              msg[e.name] ? (
                msg[e.name] = msg[e.name] + ',' + resolvVar(e.value)
              ) : (
                msg[e.name] = resolvVar(e.value)
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

  modifierHandlers = new algo.Cache(makeModifierHandler),

  makeRequestModifierHandler = cfg => (
    (
      handlers = (cfg?.Filters || []).filter(
        e => e?.Type === 'RequestHeaderModifier'
      ).map(
        e => modifierHandlers.get(e)
      ).filter(
        e => e
      )
    ) => (
      handlers.length > 0 ? handlers : null
    )
  )(),

  requestModifierHandlers = new algo.Cache(makeRequestModifierHandler),

  makeResponseModifierHandler = cfg => (
    (
      handlers = (cfg?.Filters || []).filter(
        e => e?.Type === 'ResponseHeaderModifier'
      ).map(
        e => modifierHandlers.get(e)
      ).filter(
        e => e
      )
    ) => (
      handlers.length > 0 ? handlers : null
    )
  )(),

  responseModifierHandlers = new algo.Cache(makeResponseModifierHandler),

) => pipy({
  _requestHandlers: null,
  _responseHandlers: null,
})

.import({
  __service: 'service',
  __http: 'http',
  __consumer: 'consumer',
})

.pipeline()
.onStart(
  () => void (
    _requestHandlers = requestModifierHandlers.get(__service),
    _responseHandlers = responseModifierHandlers.get(__service)
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