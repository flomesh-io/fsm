export default function (config) {
  var host = config.externalRateLimit.throttleHost
  var passHeaders = config.externalRateLimit.passHeaders

  var $throttleResponse
  var $throttleResolve

  var errorResponse = new Message({ status: 503 }, 'Throttle service down')

  return pipeline($=>$
    .fork().to($=>$
      .replaceData()
      .replaceMessageStart(
        req => {
          if (passHeaders instanceof Array && passHeaders.length > 0) {
            var head = req.head
            var headers = head.headers
            var selectedHeaders = Object.fromEntries(
              passHeaders.map(k => [k, headers?.[k]])
            )
            return new MessageStart({
              method: head.method,
              path: head.path,
              headers: selectedHeaders,
            })
          } else {
            return req
          }
        }
      )
      .muxHTTP(() => 1, { version: 2 }).to($=>$
        .connect(host)
      )
      .handleMessage(
        res => {
          $throttleResponse = res?.head ? res : errorResponse
          $throttleResolve(true)
        }
      )
    )
    .wait(() => new Promise(r => { $throttleResolve = r }))
    .pipe(() => $throttleResponse.head.status === 200 ? 'pass' : 'reject', {
      'pass': $=>$.pipeNext(),
      'reject': $=>$.replaceData().replaceMessage(() => $throttleResponse)
    })
  )
}
