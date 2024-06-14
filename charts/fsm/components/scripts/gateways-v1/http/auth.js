((
  { config, isDebugEnabled } = pipy.solve('config.js'),

  headersAuthorization = {},

  _0 = (config?.Consumers || []).forEach(
    c => (
      Object.entries(c['Headers-Authorization'] || {}).map(
        ([k, v]) => (
          !headersAuthorization[k] && (headersAuthorization[k] = {}),
          headersAuthorization[k][v] = c
        )
      )
    )
  ),

) => pipy({
  _consumer: null,
})

.import({
  __consumer: 'consumer',
})

.pipeline()
.handleStreamStart(
  msg => (
    Object.keys(headersAuthorization).forEach(
      h => !_consumer && (
        (_consumer = headersAuthorization[h][msg?.head?.headers?.[h]]) && (
          __consumer ? (
            Object.keys(_consumer).forEach(
              k => (__consumer[k] = _consumer[k])
            )
          ) : (
            __consumer = _consumer
          )
        )
      )
    )
  )
)
.branch(
  isDebugEnabled, (
    $=>$.handleStreamStart(
      msg => (
        console.log('[auth] consumer, msg:', __consumer, msg)
      )
    )
  )
)
.chain()

)()