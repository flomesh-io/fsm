export default function () {
  var $ctx
  var $proto

  return pipeline($=>$
    .onStart(c => { $ctx = c })
    .detectProtocol(p => { $proto = p })
    .pipe(
      () => {
        if ($proto !== undefined) {
          return $proto === 'HTTP' ? 'http' : 'socks'
        }
      }, {
        'http': ($=>$
          .demuxHTTP().to($=>$
            .pipe(
              function (evt) {
                if (evt instanceof MessageStart) {
                  return evt.head.method === 'CONNECT' ? 'tunnel' : 'forward'
                }
              }, {
                'tunnel': ($=>$
                  .acceptHTTPTunnel(
                    function (req) {
                      $ctx.originalTarget = req.head.path
                      return new Message({ status: 200 })
                    }
                  ).to($=>$.pipeNext())
                ),
                'forward': ($=>$
                  .handleMessageStart(
                    function (req) {
                      var url = new URL(req.head.path)
                      $ctx.originalTarget = `${url.hostname}:${url.port}`
                      req.head.path = url.path
                    }
                  )
                  .pipeNext()
                ),
              }
            )
          )
        ),
        'socks': ($=>$
          .acceptSOCKS(
            function (req) {
              $ctx.originalTarget = `${req.domain || req.ip}:${req.port}`
              return true
            }
          ).to($=>$.pipeNext())
        )
      }
    )
  )
}
