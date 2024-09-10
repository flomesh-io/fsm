var PIPY_UUID = pipy.uuid || ''
var PIPY_NAME = pipy.name || ''
var PIPY_SOURCE = pipy.source || ''
var K8S_CLUSTER = os.env.PIPY_K8S_CLUSTER || ''

var LATENCY_BUCKETS = [
  1, 5, 10, 25, 50, 100, 250, 500, 1000, 2500, 5000,
  10000, 30000, 60000, 300000, 600000, 1800000, 3600000,
  Infinity
]

var backends = new Set

var meta_info = new stats.Gauge('fgw_meta_info', ['uuid', 'name', 'codebase', 'cluster'])
meta_info.withLabels(PIPY_UUID, PIPY_NAME, PIPY_SOURCE, K8S_CLUSTER).set(1)

var bandwidth = new stats.Counter('fgw_bandwidth', ['type', 'backend', 'target', 'route'])
var bandwidth_ingress = bandwidth.withLabels('ingress')
var bandwidth_egress = bandwidth.withLabels('egress')
var backend_connection_total = new stats.Gauge('fgw_backend_connection_total', ['backend'])

var http_request_total = new stats.Counter('fgw_http_request_total')
var http_status = new stats.Counter('fgw_http_status', ['backend', 'target', 'route', 'consumer', 'matched_host', 'matched_uri', 'code'])
var http_latency = new stats.Histogram('fgw_http_latency', LATENCY_BUCKETS, ['backend', 'target', 'route', 'consumer', 'type'])

var http_retry = new stats.Counter('fgw_http_retry', ['backend'])
var http_retry_success = new stats.Counter('fgw_http_retry_success', ['backend'])
var http_retry_limit_exceeded = new stats.Counter('fgw_http_retry_limit_exceeded', ['backend'])
var http_retry_backoff_exponential = new stats.Counter('fgw_http_retry_backoff_exponential', ['backend'])

export default function (config) {
  var sampleInterval = Number.parseFloat(config.metrics?.sampleInterval) || 5

  function updateUpstreamStats() {
    new Timeout(sampleInterval).wait().then(() => {
      backends.forEach(be => {
        backend_connection_total.withLabels(be.name).set(be.concurrency)
      })
      updateUpstreamStats()
    })
  }

  updateUpstreamStats()

  var $ctx

  return pipeline($=>$
    .onStart(c => { $ctx = c })
    .handleMessageStart(() => {
      http_request_total.increase()
    })
    .pipeNext()
    .handleMessageEnd(() => {
      var host = $ctx.host
      var backend = $ctx.backendResource?.metadata?.name
      var route = $ctx.routeResource.metadata?.name
      var target = $ctx.target
      var consumer = $ctx.consumer
      var response = $ctx.response
      var status = response.head?.status
      var ingress = $ctx.tail.headSize + $ctx.tail.bodySize
      var egress = response.tail ? response.tail.headSize + response.tail.bodySize : 0

      http_status.withLabels(backend, target, route, consumer, host, $ctx.basePath, status).increase()
      var rx = bandwidth_ingress.withLabels(backend)
      rx.increase(ingress)
      rx.withLabels(target, route).increase(ingress)
      var tx = bandwidth_egress.withLabels(backend)
      tx.increase(egress)
      tx.withLabels(target, route).increase(egress)

      var l = http_latency.withLabels(target, backend, route, consumer)
      l.withLabels('upstream').observe(response.headTime - $ctx.sendTime)
      l.withLabels('fgw').observe($ctx.sendTime - $ctx.headTime)
      l.withLabels('request').observe(Date.now() - $ctx.headTime)

      var retries = $ctx.retries
      if (retries.length > 0) {
        http_retry.withLabels(backend).increase()
        if (retries[retries.length - 1].succeeded) {
          http_retry_success.withLabels(backend).increase()
        } else {
          http_retry_limit_exceeded.withLabels(backend).increase()
        }
        if (retries.length > 1) {
          http_retry_backoff_exponential.withLabels(backend).increase()
        }
      }
      if ($ctx.backend) {
        backends.add($ctx.backend)
      }
    })
  )
}
