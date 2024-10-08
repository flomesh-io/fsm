export default function (config) {
  var proxyTag = (config.proxyTag?.dstHostHeader || 'proxy-tag').toLowerCase()
  var origHost = (config.proxyTag?.srcHostHeader || 'orig-host').toLowerCase()

  return pipeline($=>$
    .demuxHTTP().to($=>$
      .handleMessageStart(
        msg => {
          var headers = msg.head.headers
          var tag = headers[proxyTag]
          if (tag) {
            headers[origHost] = headers.host
            headers.host = tag
          } else if (headers['fgw-target-service']) {
            headers[proxyTag] = headers['fgw-target-service']
          } else {
            headers[proxyTag] = headers.host
          }
        }
      )
      .pipeNext()
    )
  )
}
