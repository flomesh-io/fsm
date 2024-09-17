export default function (config) {
  var log = console.log
  var $ctx

  return pipeline($=>$
    .onStart(c => { $ctx = c })
    .pipeNext()
    .handleMessageEnd(() => {
      var inbound = $ctx.parent.inbound
      var headers = $ctx.head.headers
      var response = $ctx.response
      var target = $ctx.target
      log({
        protocol: $ctx.head.protocol || '',
        upstream_service_time: response.headTime,
        upstream_local_address: target || '',
        duration: response.tailTime - $ctx.headTime,
        upstream_transport_failure_reason: '',
        route_name: '',
        downstream_local_address: inbound.localAddress,
        user_agent: headers['user-agent'] || '',
        response_code: response.head?.status,
        response_flags: '',
        start_time: $ctx.headTime,
        method: $ctx.head.method || '',
        request_id: $ctx.id,
        upstream_host: target,
        x_forward_for: headers['x-forwarded-for'] || '',
        client_ip: inbound.remoteAddress,
        requested_server_name: '',
        bytes_received: $ctx.tail.headSize + $ctx.tail.bodySize,
        bytes_sent: response.tail ? response.tail.headSize + response.tail.bodySize : 0,
        upstream_cluster: $ctx.backendResource?.metadata?.name,
        downstream_remote_address: inbound.remoteAddress,
        authority: $ctx.head.authority || '',
        path: $ctx.path,
        response_code_details: response.head?.statusText,
        trace_id: headers['x-b3-traceid'] || '',
      })
    })
  )
}
