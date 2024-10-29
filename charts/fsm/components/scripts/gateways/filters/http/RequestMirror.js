export default function (config, resources) {
  var backendRef = config.requestMirror.backendRef
  var percent = config.requestMirror.percent
  var fraction = config.requestMirror.fraction

  if (backendRef) {
    var kind = backendRef.kind || 'Backend'
    var name = backendRef.name
    var backend = resources.list(kind).find(r => r.metadata.name === name)
    var target = backend?.spec?.targets?.[0]
  }

  if (percent || fraction) {
    var weight0 = percent ? 100 - percent : fraction.denominator - fraction.numerator
    var weight1 = percent ? percent : fraction.numerator
    var sampler = new algo.LoadBalancer([true, false], { weight: t => t ? weight1 : weight0 })
  }

  if (target) {
    var mirror = pipeline($=>$
      .fork().to($=>$
        .muxHTTP().to($=>$
          .connect(`${target.address}:${target.port}`)
        )
      )
    )

    var bypass = pipeline($=>$)

    if (sampler) {
      return pipeline($=>$
        .pipe(() => sampler.allocate().target ? mirror : bypass)
        .pipeNext()
      )
    } else {
      return pipeline($=>$
        .pipe(mirror)
        .pipeNext()
      )
    }
  } else {
    return pipeline($=>$.pipeNext())
  }
}
