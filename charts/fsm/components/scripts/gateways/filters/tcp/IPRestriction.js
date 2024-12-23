export default function (config) {
  var allowed = config.ipRestriction?.allowed || []
  var forbidden = config.ipRestriction?.forbidden || []

  allowed = allowed.map(s => {
    if (s.indexOf('/') < 0) s += '/32'
    return new IPMask(s)
  })

  forbidden = forbidden.map(s => {
    if (s.indexOf('/') < 0) s += '/32'
    return new IPMask(s)
  })

  var $rejected = false

  return pipeline($=>$
    .onStart(ctx => {
      var ip = ctx.inbound.remoteAddress
      if (allowed.length > 0) {
        if (!allowed.some(m => m.contains(ip))) {
          $rejected = true
          return new StreamEnd
        }
      }
      if (forbidden.length > 0) {
        if (forbidden.some(m => m.contains(ip))) {
          $rejected = true
          return new StreamEnd
        }
      }
    })
    .pipe(() => $rejected ? 'reject' : 'pass', {
      'reject': $=>$,
      'pass': $=>$.pipeNext(),
    })
  )
}
