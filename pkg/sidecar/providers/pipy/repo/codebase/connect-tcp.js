((
  config = pipy.solve('config.js'),
  isDebugEnabled = config?.Spec?.SidecarLogLevel === 'debug',
  {
    metricsCache,
  } = pipy.solve('metrics.js'),

  connectOptions = (config?.Spec?.SidecarTimeout > 0) ? (
    {
      connectTimeout: config.Spec.SidecarTimeout,
      readTimeout: config.Spec.SidecarTimeout,
      writeTimeout: config.Spec.SidecarTimeout,
      idleTimeout: config.Spec.SidecarTimeout,
    }
  ) : {},

) => (

pipy({
  _metrics: null,
})

.export('connect-tcp', {
  __target: null,
  __metricLabel: null,
})

.pipeline()
.onStart(
  () => void (
    _metrics = metricsCache.get(__metricLabel),
    _metrics.activeConnectionGauge.increase()
  )
)
.onEnd(
  () => void (
    _metrics.activeConnectionGauge.decrease()
  )
)
.branch(
  isDebugEnabled, (
    $=>$
    .handleStreamStart(
      () => (
        console.log('connect-tcp # metrics/target :', __metricLabel, __target)
      )
    )
  )
)
.handleData(
  data => (
    _metrics.sendBytesTotalCounter.increase(data.size)
  )
)
.branch(
  () => __target.startsWith('127.0.0.1:'), (
    $=>$.connect(() => __target, { bind: '127.0.0.6', ...connectOptions })
  ),
  (
    $=>$.connect(() => __target, connectOptions)
  )
)
.handleData(
  data => (
    _metrics.receiveBytesTotalCounter.increase(data.size)
  )
)

))()
