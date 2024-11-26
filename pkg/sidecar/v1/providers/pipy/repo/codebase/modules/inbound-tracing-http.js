((
  {
    tracingEnabled,
    makeZipKinData,
    saveTracing,
  } = pipy.solve('tracing.js'),
  sampledCounter0 = new stats.Counter('inbound_http_tracing_sampled_0'),
  sampledCounter1 = new stats.Counter('inbound_http_tracing_sampled_1'),
) => (

pipy({
  _sampled: true,
  _zipkinData: null,
  _httpBytesStruct: null,
})

.import({
  __cluster: 'inbound-http-routing',
  __target: 'connect-tcp',
})

.pipeline()
.handleMessage(
  (msg) => (
    tracingEnabled && (
      (_sampled = (msg?.head?.headers?.['x-b3-sampled'] === '1')) && (
        _httpBytesStruct = {},
        _httpBytesStruct.requestSize = msg?.body?.size,
        _zipkinData = makeZipKinData(msg, msg.head.headers, __cluster?.name, 'SERVER', true)
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