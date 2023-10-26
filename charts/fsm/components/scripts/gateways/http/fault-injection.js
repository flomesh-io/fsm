((
  { config, isDebugEnabled } = pipy.solve('config.js'),

  delayFaultCache = new algo.Cache(
    route => (
      (
        range,
        min = 0,
        max = 0,
        unit = 0.001, // default ms
        delay = route?.config?.Fault?.Delay || __domain?.Fault?.Delay || config?.Configs?.Fault?.Delay,
        percent = delay?.Percent,
        fixed = delay?.Fixed,
      ) => (
        percent > 0 && (
          delay.Unit?.toLowerCase?.() === 's' ? (
            unit = 1
          ) : delay.Unit?.toLowerCase?.() === 'm' && (
            unit = 60
          ),
          delay.Range ? (
            range = delay.Range.split('-'),
            min = range[0],
            max = range[1],
            min >= 0 && max >= min && (
              () => (
                Math.floor(Math.random() * 100) <= percent ? (
                  (Math.floor(Math.random() * (max - min + 1)) + min) * unit
                ) : 0
              )
            )
          ) : fixed > 0 && (
            fixed *= unit,
            () => (
              Math.floor(Math.random() * 100) <= percent ? (
                fixed
              ) : 0
            )
          )
        ) || null
      )
    )()
  ),

  abortFaultCache = new algo.Cache(
    route => (
      (
        abort = route?.config?.Fault?.Abort || __domain?.Fault?.Abort || config?.Configs?.Fault?.Abort,
        percent = abort?.Percent,
        status = abort?.Status,
        message,
      ) => (
        percent > 0 && status > 0 && (
          __domain.RouteType === 'GRPC' ? (
            message = new Message({ status: 200, headers: { 'content-type': 'application/grpc', 'grpc-encoding': 'identity' } }, null, { headers: { 'grpc-status': status, 'grpc-message': abort?.Message || '' } })
          ) : (
            message = new Message({ status }, abort?.Message || '')
          ),
          () => (
            Math.floor(Math.random() * 100) <= percent ? (
              message
            ) : null
          )
        ) || null
      )
    )()
  ),

) => pipy({
  _timeout: 0,
  _message: null,
})

.import({
  __domain: 'route',
  __route: 'route',
})

.pipeline()
.branch(
  () => (_timeout = delayFaultCache.get(__route)?.()) > 0, (
    $=>$.handleMessageStart(
      () => new Timeout(_timeout).wait()
    )
  ), (
    $=>$
  )
)
.branch(
  () => _message = abortFaultCache.get(__route)?.(), (
    $=>$.replaceMessage(
      () => _message
    )
  ), (
    $=>$.chain()
  )
)
.branch(
  isDebugEnabled, (
    $=>$.handleMessageStart(
      () => (_timeout || _message) && (
        console.log('[fault-injection] delay, message:', _timeout, _message)
      )
    )
  )
)

)()