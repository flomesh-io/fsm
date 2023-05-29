((
  {
    loggingEnabled,
    makeLoggingData,
    saveLoggingData,
  } = pipy.solve('logging.js'),
  sampledCounter0 = new stats.Counter('outbound_http_logging_sampled_0'),
  sampledCounter1 = new stats.Counter('outbound_http_logging_sampled_1'),
) => (

pipy({
  _loggingData: null
})

.import({
  __isEgress: 'outbound',
  __target: 'connect-tcp',
  __service: 'outbound-http-routing',
})

.pipeline()
.handleMessage(
  msg => (
    loggingEnabled && (
      _loggingData = makeLoggingData(msg, __inbound.destinationAddress, __inbound.destinationPort, __inbound.remoteAddress, __inbound.remotePort),
      _loggingData ? sampledCounter1.increase() : sampledCounter0.increase()
    )
  )
)
.chain()
.handleMessage(
  msg => (
    loggingEnabled && _loggingData && (
      saveLoggingData(_loggingData, msg, __service?.name, __target, false, __isEgress, 'outbound')
    )
  )
)

))()