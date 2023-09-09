((
  {
    tracingEnabled,
    initTracingHeaders,
    makeZipKinData,
    saveTracing,
  } = pipy.solve('lib/tracing.js'),
  sampledCounter0 = new stats.Counter('fgw_tracing_sampled_0'),
  sampledCounter1 = new stats.Counter('fgw_tracing_sampled_1'),
) => (

pipy({
  _sampled: true,
  _zipkinData: null,
  _httpBytesStruct: null,
})

.import({
  __port: 'listener',
  __service: 'service',
  __target: 'connect-tcp',
})

.pipeline()
.handleMessage(
  (msg) => (
    tracingEnabled && (
      (_sampled = initTracingHeaders(msg.head.headers, __port?.Protocol)) && (
        _httpBytesStruct = {},
        _httpBytesStruct.requestSize = msg?.body?.size,
        _zipkinData = makeZipKinData(msg, msg.head.headers, __service?.name)
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