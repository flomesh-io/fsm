export default function (config) {
  var log = console.log
  var $ctx

  return pipeline($=>$
    .onStart(c => { $ctx = c })
    .pipeNext()
    .handleMessageEnd(() => {
      var inbound = $ctx.parent.inbound
      var headers = $ctx.head.headers || {}
      var response = $ctx.response
      var target = $ctx.target

      // Log the request and response
      // Format: remoteAddress (traceId) - [timestamp] "method path protocol" statusCode statusText "user-agent" responseTime requestSize responseSize "upstream" "backend"
      log(
        `${inbound.remoteAddress}`,
        `(${headers['x-b3-traceid'] || ''})`,
        `-`,
        `[${new Date().toISOString()}]`,
        `"${$ctx.head?.method || ''} ${$ctx.path} ${$ctx.head?.protocol || ''}"`,
        `${response.head?.status}`,
        `${response.head?.statusText || ''}`,
        `"${headers['user-agent'] || ''}"`,
        `${response.headTime - $ctx.headTime}ms`,
        `${$ctx.tail ? ($ctx.tail?.headSize || 0 + $ctx.tail?.bodySize || 0) : 0}`,
        `${response.tail ? (response.tail?.headSize || 0 + response.tail?.bodySize || 0) : 0}`,
        `"${$ctx.backendResource?.metadata?.name || ''}"`,
        `"${target || ''}"`
      )
    })
  )
}
