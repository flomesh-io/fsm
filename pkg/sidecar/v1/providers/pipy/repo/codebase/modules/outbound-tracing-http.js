((
  {
    tracingEnabled,
    initTracingHeaders,
    makeZipKinData,
    saveTracing,
  } = pipy.solve('tracing.js'),
  sampledCounter0 = new stats.Counter('outbound_http_tracing_sampled_0'),
  sampledCounter1 = new stats.Counter('outbound_http_tracing_sampled_1'),
) => (

pipy({
  _sampled: true,
  _zipkinData: null,
  _httpBytesStruct: null,
})

.import({
  __protocol: 'outbound',
  __cluster: 'outbound-http-routing',
  __target: 'connect-tcp',
})

.pipeline()
.handleMessage(
  (msg) => (
    tracingEnabled && (
      (_sampled = initTracingHeaders(msg.head.headers, __protocol)) && (
        _httpBytesStruct = {},
        _httpBytesStruct.requestSize = msg?.body?.size,
        _zipkinData = makeZipKinData(msg, msg.head.headers, __cluster?.name, 'CLIENT', false)
      ),
      _sampled ? sampledCounter1.increase() : sampledCounter0.increase()
    )
  )
)
.chain()
.handleMessage(
  (msg) => (
    tracingEnabled && _sampled && (
      _httpBytesStruct.responseSize = msg?.body?.size,
      saveTracing(_zipkinData, msg?.head, _httpBytesStruct, __target)
    )
  )
)

))()