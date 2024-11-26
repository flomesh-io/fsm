((
  config = pipy.solve('config.js'),
  inboundTarget,
  parseProbe,
  livenessProbe,
  readinessProbe,
  startupProbe,
) => (
  (
    config?.Inbound?.TrafficMatches && Object.entries(config.Inbound.TrafficMatches).map(
      ([port, match]) => (
        (match.Protocol === 'http') && (!inboundTarget || !match.SourceIPRanges) && (inboundTarget = '127.0.0.1:' + port)
      )
    ),

    parseProbe = (config, port) => (
      (path, target = inboundTarget, scheme) => (
        (config?.httpGet?.port === port) && (
          scheme = config?.httpGet?.scheme,
          (config?.httpGet?.httpHeaders || []).forEach(
            h => (
              (h.name === 'Original-Http-Port') && (target = '127.0.0.1:' + h.value),
              (h.name === 'Original-Tcp-Port') && (
                target = '127.0.0.1:15904',
                scheme = 'TCP'
              ),
              (h.name === 'Original-Http-Path') && (path = h.value)
            )
          ),
          !target && (
            (scheme === 'HTTP') && (target = '127.0.0.1:80'),
            (scheme === 'HTTPS') && (target = '127.0.0.1:443')
          ),
          !path && (path = '/')
        ),
        { path, target, scheme }
      )
    )(),

    livenessProbe = parseProbe(config?.Spec?.Probes?.LivenessProbes?.[0], 15901),
    readinessProbe = parseProbe(config?.Spec?.Probes?.ReadinessProbes?.[0], 15902),
    startupProbe = parseProbe(config?.Spec?.Probes?.StartupProbes?.[0], 15903)

    // , console.log('=== probes ===', livenessProbe, readinessProbe, startupProbe)

  ),

  pipy()

  .pipeline('liveness')
  .branch(
    () => livenessProbe?.scheme === 'HTTP', $=>$
      .demuxHTTP().to($=>$
        .handleMessageStart(
          msg => (
            msg.head.path = livenessProbe?.path
          )
        )
        .muxHTTP(() => livenessProbe?.target).to($=>$
          .connect(() => livenessProbe?.target)
        )
      ),
    () => Boolean(livenessProbe?.target), $=>$
      .connect(() => livenessProbe?.target),
      $=>$
      .replaceStreamStart(
        new StreamEnd('ConnectionReset')
      )
  )

  .pipeline('readiness')
  .branch(
    () => readinessProbe?.scheme === 'HTTP', $=>$
      .demuxHTTP().to($=>$
        .handleMessageStart(
          msg => (
            msg.head.path = readinessProbe?.path
          )
        )
        .muxHTTP(() => readinessProbe?.target).to($=>$
            .connect(() => readinessProbe?.target)
        )
      ),
    () => Boolean(readinessProbe?.target), $=>$
      .connect(() => readinessProbe?.target),
      $=>$
      .replaceStreamStart(
        new StreamEnd('ConnectionReset')
      )
  )

  .pipeline('startup')
  .branch(
    () => startupProbe?.scheme === 'HTTP', $=>$
      .demuxHTTP().to($=>$
        .handleMessageStart(
          msg => (
            msg.head.path = startupProbe?.path
          )
        )
        .muxHTTP(() => startupProbe?.target).to($=>$
          .connect(() => startupProbe?.target)
        )
      ),
    () => Boolean(startupProbe?.target), $=>$
      .connect(() => startupProbe?.target),
      $=>$
      .replaceStreamStart(
        new StreamEnd('ConnectionReset')
      )
  )

))()

