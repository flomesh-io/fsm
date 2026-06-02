export default function (config, resources) {
  var backendRef = config.requestMirror.backendRef
  var percent = config.requestMirror.percent
  var fraction = config.requestMirror.fraction
  var filterKey = config.key

  console.log('[RequestMirror] filter key:', filterKey, 'backendRef:', JSON.stringify(backendRef))

  if (backendRef) {
    var kind = backendRef.kind || 'Backend'
    var name = backendRef.name
    var backend = resources.list(kind).find(r => r.metadata.name === name)
    var target = backend?.spec?.targets?.[0]

    if (backend) {
      console.log('[RequestMirror] backend found:', name, 'targets:', backend.spec?.targets?.length || 0)
    } else {
      console.log('[RequestMirror] backend NOT found:', kind, name)
    }

    if (target) {
      console.log('[RequestMirror] mirror target:', target.address, target.port)
    } else {
      console.log('[RequestMirror] no target available, mirror disabled')
    }
  } else {
    console.log('[RequestMirror] no backendRef in config, mirror disabled')
  }

  if (percent || fraction) {
    var weight0 = percent ? 100 - percent : fraction.denominator - fraction.numerator
    var weight1 = percent ? percent : fraction.numerator
    var sampler = new algo.LoadBalancer([true, false], { weight: t => t ? weight1 : weight0 })
    console.log('[RequestMirror] sampling enabled, percent:', percent, 'fraction:', fraction)
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
      console.log('[RequestMirror] mirror active with sampling')
      return pipeline($=>$
        .pipe(() => sampler.allocate().target ? mirror : bypass)
        .pipeNext()
      )
    } else {
      console.log('[RequestMirror] mirror active, all requests mirrored')
      return pipeline($=>$
        .pipe(mirror)
        .pipeNext()
      )
    }
  } else {
    console.log('[RequestMirror] mirror bypassed, no valid target')
    return pipeline($=>$.pipeNext())
  }
}
