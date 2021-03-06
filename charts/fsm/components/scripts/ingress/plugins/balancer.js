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

(config =>

  pipy({
    _services: (
      Object.fromEntries(
        Object.entries(config.services).map(
          ([k, v]) => (
            ((balancer => (
              balancer = new algo[v.balancer ? v.balancer : 'RoundRobinLoadBalancer'](v.targets),
              [k, {
                balancer,
                cache: v.sticky && new algo.Cache(
                  () => balancer.select()
                )
              }]
            ))()
            )
          )
        )
      )
    ),
    _requestCounter: new stats.Counter('http_requests_count', ['method', 'status', "host", "path"]),
    _requestLatency: new stats.Histogram('http_request_latency', [1, 2, 5, 7, 10, 15, 20, 25, 30, 40, 50, 60, 70,
      80, 90, 100, 200, 300, 400, 500, 1000,
      2000, 5000, 10000, 30000, 60000, Number.POSITIVE_INFINITY]),
    _reqHead: null,
    _resHead: null,
    _reqTime: 0,
    _service: null,
    _serviceCache: null,
    _target: '',
    _targetCache: null,

    _g: {
      connectionID: 0,
    },

    _connectionPool: new algo.ResourcePool(
      () => ++_g.connectionID
    ),

    _selectKey: null,
    _select: (service, key) => (
      service.cache && key ? (
        service.cache.get(key)
      ) : (
        service.balancer.select()
      )
    ),
  })

  .import({
      __turnDown: 'main',
      __service: 'router',
    })

  .pipeline('session')
    .handleStreamStart(
      () => (
        _serviceCache = new algo.Cache(
          // k is a balancer, v is a target
          (k) => _select(k, _selectKey),
          (k, v) => k.balancer.deselect(v),
        ),
        _targetCache = new algo.Cache(
          // k is a target, v is a connection ID
          (k) => _connectionPool.allocate(k),
          (k, v) => _connectionPool.free(v),
        )
      )
    )
    .handleStreamEnd(
      () => (
        _targetCache.clear(),
        _serviceCache.clear()
      )
    )

  .pipeline('request')
    .handleMessageStart(
      (msg) => (
        _selectKey = msg.head?.headers?.['authorization'],
        _service = _services[__service],
        _service && (_target = _serviceCache.get(_service)),
        _target && (msg.head.headers['host'] = _target.split(":")[0]),
        _target && (__turnDown = true)
      )
    )
    .link(
      'forward', () => Boolean(_target),
      ''
    )

  .pipeline('forward')
    .handleMessageStart(
      msg => (
        _reqHead = msg.head,
        _reqTime = Date.now()
      )
    )
    .muxHTTP(
      'connection',
      () => _targetCache.get(_target)
    )
    .handleMessageStart(
      msg => (
        _resHead = msg.head,
        _requestCounter.withLabels(_reqHead.method, _resHead.status, _reqHead.headers.host, _reqHead.path).increase(),
        _requestLatency.observe(Date.now() - _reqTime)
      )
    )

  .pipeline('connection')
    .handleMessage(
      msg => (console.log('Ingress connection: ' + _target))
    )
    .connect(
      () => _target
    )

)(JSON.decode(pipy.load('config/balancer.json')))
