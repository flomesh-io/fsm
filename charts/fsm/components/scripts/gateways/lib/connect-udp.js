((
  { isDebugEnabled } = pipy.solve('config.js'),
  { metrics, metricsCache } = pipy.solve('lib/metrics.js'),
) => pipy({
  _metrics: null,
})

.export('connect-udp', {
  __target: null,
  __metricLabel: null,
})

.pipeline()
.onStart(
  () => void (
    _metrics = metricsCache.get(__metricLabel),
    _metrics.activeConnectionGauge.increase(),
    metrics.fgwStreamConnectionTotal.withLabels(__metricLabel).increase()
  )
)
.onEnd(
  () => void (
    _metrics.activeConnectionGauge.decrease()
  )
)
.branch(
  isDebugEnabled, (
    $=>$.handleStreamStart(
      () => (
        console.log('[connect-udp] metrics, target :', __metricLabel, __target)
      )
    )
  )
)
.handleData(
  data => (
    _metrics.sendBytesTotalCounter.increase(data.size)
  )
)
.connect(() => __target, { protocol: 'udp' })
.handleData(
  data => (
    _metrics.receiveBytesTotalCounter.increase(data.size)
  )
)

)()