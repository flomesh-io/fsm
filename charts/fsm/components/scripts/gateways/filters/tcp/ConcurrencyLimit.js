export default function (config) {
  var maxConnections = config.concurrencyLimit?.maxConnections | 0

  if (maxConnections > 0) {
    var quota = new algo.Quota(maxConnections, { key: config.key })

    var $rejected = false

    return pipeline($=>$
      .onStart(() => {
        if (quota.consume(1) <= 0) {
          $rejected = true
          return new StreamEnd
        }
      })
      .pipe(() => $rejected ? 'reject' : 'pass', {
        'reject': $=>$,
        'pass': $=>$.pipeNext(),
      })
      .onEnd(() => {
        console.log($rejected)
        if (!$rejected) {
          quota.produce(1)
        }
      })
    )
  } else {
    return pipeline($=>$.pipeNext())
  }
}
