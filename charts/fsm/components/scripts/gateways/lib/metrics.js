/*
 * MIT License
 *
 * Copyright (c) since 2021,  flomesh.io Authors.
 *
 * Permission is hereby granted, free of charge, to any person obtaining a copy
 * of this software and associated documentation files (the "Software"), to deal
 * in the Software without restriction, including without limitation the rights
 * to use, copy, modify, merge, publish, distribute, sublicense, and/or sell
 * copies of the Software, and to permit persons to whom the Software is
 * furnished to do so, subject to the following conditions:
 *
 * The above copyright notice and this permission notice shall be included in all
 * copies or substantial portions of the Software.
 *
 * THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
 * IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY,
 * FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE
 * AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER
 * LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM,
 * OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN THE
 * SOFTWARE.
 */

(
  (
    config = pipy.solve('config.js'),

    {
      namespace,
      kind,
      name,
      pod,
    } = pipy.solve('lib/utils.js'),

    sendBytesTotalCounter = new stats.Counter('fgw_service_upstream_cx_tx_bytes_total', [
      'fgw_service_name'
    ]),
    receiveBytesTotalCounter = new stats.Counter('fgw_service_upstream_cx_rx_bytes_total', [
      'fgw_service_name'
    ]),
    activeConnectionGauge = new stats.Gauge('fgw_service_upstream_cx_active', [
      'fgw_service_name'
    ]),
    upstreamCompletedCount = new stats.Counter('fgw_service_external_upstream_rq_completed', [
      'fgw_service_name'
    ]),
    destroyRemoteActiveCounter = new stats.Counter('fgw_service_upstream_cx_destroy_remote_with_active_rq', [
      'fgw_service_name'
    ]),
    destroyLocalActiveCounter = new stats.Counter('fgw_service_upstream_cx_destroy_local_with_active_rq', [
      'fgw_service_name'
    ]),
    connectTimeoutCounter = new stats.Counter('fgw_service_upstream_cx_connect_timeout', [
      'fgw_service_name'
    ]),
    pendingFailureEjectCounter = new stats.Counter('fgw_service_upstream_rq_pending_failure_eject', [
      'fgw_service_name'
    ]),
    pendingOverflowCounter = new stats.Counter('fgw_service_upstream_rq_pending_overflow', [
      'fgw_service_name'
    ]),
    requestTimeoutCounter = new stats.Counter('fgw_service_upstream_rq_timeout', [
      'fgw_service_name'
    ]),
    requestReceiveResetCounter = new stats.Counter('fgw_service_upstream_rq_rx_reset', [
      'fgw_service_name'
    ]),
    requestSendResetCounter = new stats.Counter('fgw_service_upstream_rq_tx_reset', [
      'fgw_service_name'
    ]),
    upstreamCodeCount = new stats.Counter('fgw_service_external_upstream_rq', [
      'fgw_service_name',
      'fgw_response_code'
    ]),
    upstreamCodeXCount = new stats.Counter('fgw_service_external_upstream_rq_xx', [
      'fgw_service_name',
      'fgw_response_code_class'
    ]),
    upstreamResponseTotal = new stats.Counter('fgw_service_upstream_rq_total', [
      'source_namespace',
      'source_workload_kind',
      'source_workload_name',
      'source_workload_pod',
      'fgw_service_name'
    ]),
    upstreamResponseCode = new stats.Counter('fgw_service_upstream_rq_xx', [
      'source_namespace',
      'source_workload_kind',
      'source_workload_name',
      'source_workload_pod',
      'fgw_service_name',
      'fgw_response_code_class'
    ]),

    fgwRequestDurationHist = new stats.Histogram('fgw_request_duration_ms', [
      5, 10, 25, 50, 100, 250, 500, 1000, 2500, 5000, 10000, 30000, 60000, 300000, 600000, 1800000, 3600000, Infinity
    ], [
      'source_namespace',
      'source_kind',
      'source_name',
      'source_pod',
      'fgw_service_name'
    ]),

    metricsCache = new algo.Cache(serviceName => (
      {
        sendBytesTotalCounter: sendBytesTotalCounter.withLabels(serviceName),
        receiveBytesTotalCounter: receiveBytesTotalCounter.withLabels(serviceName),
        activeConnectionGauge: activeConnectionGauge.withLabels(serviceName),
        upstreamCompletedCount: upstreamCompletedCount.withLabels(serviceName),
        destroyRemoteActiveCounter: destroyRemoteActiveCounter.withLabels(serviceName),
        destroyLocalActiveCounter: destroyLocalActiveCounter.withLabels(serviceName),
        connectTimeoutCounter: connectTimeoutCounter.withLabels(serviceName),
        pendingFailureEjectCounter: pendingFailureEjectCounter.withLabels(serviceName),
        pendingOverflowCounter: pendingOverflowCounter.withLabels(serviceName),
        requestTimeoutCounter: requestTimeoutCounter.withLabels(serviceName),
        requestReceiveResetCounter: requestReceiveResetCounter.withLabels(serviceName),
        requestSendResetCounter: requestSendResetCounter.withLabels(serviceName),
        upstreamCodeCount: upstreamCodeCount.withLabels(serviceName),
        upstreamCodeXCount: upstreamCodeXCount.withLabels(serviceName),
        upstreamResponseTotal: upstreamResponseTotal.withLabels(namespace, kind, name, pod, serviceName),
        upstreamResponseCode: upstreamResponseCode.withLabels(namespace, kind, name, pod, serviceName),
      }
    )),

    durationCache = new algo.Cache(serviceName => (
      fgwRequestDurationHist.withLabels(namespace, kind, name, pod, serviceName)
    )),

  ) => (

    Object.keys(config?.Services || {}).forEach(
      serviceName => (
        (
          metrics = metricsCache.get(serviceName),
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

    {
      metricsCache,
      durationCache,
      rateLimitCounter: new stats.Counter('http_local_rate_limiter', [
        'http_local_rate_limit'
      ]),
    }
  )

)()