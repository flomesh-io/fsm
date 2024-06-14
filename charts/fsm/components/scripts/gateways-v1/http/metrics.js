((
  { metricsCache } = pipy.solve('lib/metrics.js'),
) => (

pipy({
  _metrics: null,
  _route: null,
  _consumer: null,
  _target: null,
  _status: null,
  _requestTime: null,
  _responseTime: null,
})

.import({
  __requestTail: 'http',
  __requestTime: 'http',
  __domain: 'route',
  __route: 'route',
  __service: 'service',
  __consumer: 'consumer',
  __target: 'connect-tcp',
})

.pipeline()
.handleMessageStart(
  () => (
    _requestTime = Date.now()
  )
)
.chain()
.handleMessageStart(
  (msg) => (
    _route = __route?.config?.route || '',
    _consumer = __consumer?.name || '',
    _target = __target || '',
    _status = msg?.head?.status,
    _metrics = metricsCache.get(__service?.name),

    __requestTail && _metrics.fgwBandwidth.withLabels(
      'egress',
      _route,
      _consumer,
      // __inbound.remoteAddress || ''
      'downstream'
    ).increase(__requestTail.headSize + __requestTail.bodySize),

    _status && _metrics.fgwHttpStatus.withLabels(
      _status,
      _route,
      __route?.config?.Path?.Path || '',
      __domain?.name || '',
      _consumer,
      _target
    ).increase()
  )
)
.handleMessageEnd(
  msg => (
    _responseTime = Date.now(),
    _metrics.fgwHttpLatency.withLabels(
      _route,
      _consumer,
      'upstream',
      _target
    ).observe(_responseTime - _requestTime),
    _metrics.fgwHttpLatency.withLabels(
      _route,
      _consumer,
      'fgw',
      _target
    ).observe(_requestTime - __requestTime),
    _metrics.fgwHttpLatency.withLabels(
      _route,
      _consumer,
      'request',
      _target
    ).observe(_responseTime - __requestTime),

    msg.tail && _metrics.fgwBandwidth.withLabels(
      'ingress',
      _route,
      _consumer,
      // __inbound.remoteAddress || ''
      'downstream'
    ).increase(msg.tail.headSize + msg.tail.bodySize)
  )
)

))()
