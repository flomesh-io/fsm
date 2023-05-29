(
  (
    config = pipy.solve('config.js'),

    {
      namespace,
      kind,
      name,
      pod,
    } = pipy.solve('utils.js'),

    identity = namespace + ',' + kind + ',' + name + ',' + pod,

    sendBytesTotalCounter = new stats.Counter('sidecar_cluster_upstream_cx_tx_bytes_total', [
      'sidecar_cluster_name'
    ]),
    receiveBytesTotalCounter = new stats.Counter('sidecar_cluster_upstream_cx_rx_bytes_total', [
      'sidecar_cluster_name'
    ]),
    activeConnectionGauge = new stats.Gauge('sidecar_cluster_upstream_cx_active', [
      'sidecar_cluster_name'
    ]),
    upstreamCompletedCount = new stats.Counter('sidecar_cluster_external_upstream_rq_completed', [
      'sidecar_cluster_name'
    ]),
    destroyRemoteActiveCounter = new stats.Counter('sidecar_cluster_upstream_cx_destroy_remote_with_active_rq', [
      'sidecar_cluster_name'
    ]),
    destroyLocalActiveCounter = new stats.Counter('sidecar_cluster_upstream_cx_destroy_local_with_active_rq', [
      'sidecar_cluster_name'
    ]),
    connectTimeoutCounter = new stats.Counter('sidecar_cluster_upstream_cx_connect_timeout', [
      'sidecar_cluster_name'
    ]),
    pendingFailureEjectCounter = new stats.Counter('sidecar_cluster_upstream_rq_pending_failure_eject', [
      'sidecar_cluster_name'
    ]),
    pendingOverflowCounter = new stats.Counter('sidecar_cluster_upstream_rq_pending_overflow', [
      'sidecar_cluster_name'
    ]),
    requestTimeoutCounter = new stats.Counter('sidecar_cluster_upstream_rq_timeout', [
      'sidecar_cluster_name'
    ]),
    requestReceiveResetCounter = new stats.Counter('sidecar_cluster_upstream_rq_rx_reset', [
      'sidecar_cluster_name'
    ]),
    requestSendResetCounter = new stats.Counter('sidecar_cluster_upstream_rq_tx_reset', [
      'sidecar_cluster_name'
    ]),
    upstreamCodeCount = new stats.Counter('sidecar_cluster_external_upstream_rq', [
      'sidecar_cluster_name',
      'sidecar_response_code'
    ]),
    upstreamCodeXCount = new stats.Counter('sidecar_cluster_external_upstream_rq_xx', [
      'sidecar_cluster_name',
      'sidecar_response_code_class'
    ]),
    upstreamResponseTotal = new stats.Counter('sidecar_cluster_upstream_rq_total', [
      'source_namespace',
      'source_workload_kind',
      'source_workload_name',
      'source_workload_pod',
      'sidecar_cluster_name'
    ]),
    upstreamResponseCode = new stats.Counter('sidecar_cluster_upstream_rq_xx', [
      'source_namespace',
      'source_workload_kind',
      'source_workload_name',
      'source_workload_pod',
      'sidecar_cluster_name',
      'sidecar_response_code_class'
    ]),

    fsmRequestDurationHist = new stats.Histogram('fsm_request_duration_ms', [
      5, 10, 25, 50, 100, 250, 500, 1000, 2500, 5000, 10000, 30000, 60000, 300000, 600000, 1800000, 3600000, Infinity
    ], [
      'source_namespace',
      'source_kind',
      'source_name',
      'source_pod',
      'destination_namespace',
      'destination_kind',
      'destination_name',
      'destination_pod'
    ]),

    metricsCache = new algo.Cache(clusterName => (
      {
        sendBytesTotalCounter: sendBytesTotalCounter.withLabels(clusterName),
        receiveBytesTotalCounter: receiveBytesTotalCounter.withLabels(clusterName),
        activeConnectionGauge: activeConnectionGauge.withLabels(clusterName),
        upstreamCompletedCount: upstreamCompletedCount.withLabels(clusterName),
        destroyRemoteActiveCounter: destroyRemoteActiveCounter.withLabels(clusterName),
        destroyLocalActiveCounter: destroyLocalActiveCounter.withLabels(clusterName),
        connectTimeoutCounter: connectTimeoutCounter.withLabels(clusterName),
        pendingFailureEjectCounter: pendingFailureEjectCounter.withLabels(clusterName),
        pendingOverflowCounter: pendingOverflowCounter.withLabels(clusterName),
        requestTimeoutCounter: requestTimeoutCounter.withLabels(clusterName),
        requestReceiveResetCounter: requestReceiveResetCounter.withLabels(clusterName),
        requestSendResetCounter: requestSendResetCounter.withLabels(clusterName),
        upstreamCodeCount: upstreamCodeCount.withLabels(clusterName),
        upstreamCodeXCount: upstreamCodeXCount.withLabels(clusterName),
        upstreamResponseTotal: upstreamResponseTotal.withLabels(namespace, kind, name, pod, clusterName),
        upstreamResponseCode: upstreamResponseCode.withLabels(namespace, kind, name, pod, clusterName),
      }
    )),

    identityCache = new algo.Cache(identity => (
      (
        items = identity?.split?.(','),
      ) => (
        items?.length === 4 ? fsmRequestDurationHist.withLabels(namespace, kind, name, pod, items[0], items[1], items[2], items[3]) : null
      )
    )()),

    serverLiveGauge = new stats.Gauge('sidecar_server_live'),
  ) => (

    Object.keys(config?.Inbound?.ClustersConfigs || {}).concat(Object.keys(config?.Outbound?.ClustersConfigs || {})).forEach(
      clusterName => (
        (
          metrics = metricsCache.get(clusterName),
        ) => (
          metrics.upstreamResponseTotal.zero(),
          metrics.upstreamResponseCode.withLabels('5').zero(),
          metrics.activeConnectionGauge.zero(),
          metrics.receiveBytesTotalCounter.zero(),
          metrics.sendBytesTotalCounter.zero(),
          metrics.connectTimeoutCounter.zero(),
          metrics.destroyLocalActiveCounter.zero(),
          metrics.destroyRemoteActiveCounter.zero(),
          metrics.pendingFailureEjectCounter.zero(),
          metrics.pendingOverflowCounter.zero(),
          metrics.requestTimeoutCounter.zero(),
          metrics.requestReceiveResetCounter.zero(),
          metrics.requestSendResetCounter.zero()
        )
      )()
    ),

    // Turn On Activity Metrics
    serverLiveGauge.increase(),

    {
      identity,
      metricsCache,
      identityCache,
      rateLimitCounter: new stats.Counter('http_local_rate_limiter', [
        'http_local_rate_limit'
      ]),
    }
  )

)()
