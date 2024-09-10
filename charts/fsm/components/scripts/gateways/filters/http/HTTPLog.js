export default function (config) {
  var c = config.httpLog
  var b = c.batch

  var logger = new logging.JSONLogger(config.key).toHTTP(
    c.target, {
      method: c.method || 'POST',
      headers: c.headers,
      batch: b ? {
        size: b.size,
        timeout: 1,
        interval: b.interval || 1,
        prefix: b.prefix,
        postfix: b.postfix,
        separator: b.separator,
      } : null,
      bufferLimit: c.bufferLimit || 8*1024*1024,
    }
  )

  var log = logger.log

  var meshName = os.env.MESH_NAME || ''

  var node = {
    ip: os.env.POD_IP || '127.0.0.1',
    name: os.env.HOSTNAME || 'localhost',
  }

  var pod = {
    ns: os.env.POD_NAMESPACE || 'default',
    ip: os.env.POD_IP || '127.0.0.1',
    name: os.env.POD_NAME || os.env.HOSTNAME || 'localhost',
  }

  var $ctx

  return pipeline($=>$
    .onStart(c => { $ctx = c })
    .pipeNext()
    .handleMessageEnd(() => {
      var inbound = $ctx.parent.inbound
      var reqHead = $ctx.head
      var reqTail = $ctx.tail
      var headers = reqHead.headers
      var response = $ctx.response
      var resHead = response.head
      var resTail = response.tail
      log({
        type: 'fgw',
        meshName, node, pod,
        localAddr: inbound.localAddr,
        localPort: inbound.localPort,
        remoteAddr: inbound.remoteAddr,
        remotePort: inbound.remotePort,
        reqTime: $ctx.headTime,
        resTime: response.headTime,
        endTime: response.tailTime,
        req: {
          protocol: reqHead.protocol,
          scheme: reqHead.scheme,
          authority: reqHead.authority,
          method: reqHead.method,
          path: reqHead.path,
          headers: $ctx.head.headers,
          reqSize: reqTail.bodySize,
        },
        res: {
          protocol: resHead.protocol,
          status: resHead.status,
          statusText: resHead.statusText,
          resSize: resTail?.bodySize || 0,
        },
        backend: $ctx.backend?.name,
        target: $ctx.target,
        retries: $ctx.retries,
        trace: headers['x-b3-sampled'] === '1' ? {
          id: headers['x-b3-traceid'],
          span: headers['x-b3-spanid'],
          parent: headers['x-b3-parentspanid'],
          sampled: '1',
        } : null
      })
    })
  )
}
