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

((
  { config, isDebugEnabled } = pipy.solve('config.js'),

  {
    shuffle,
    failover,
  } = pipy.solve('lib/utils.js'),

  makeServiceHandler = serviceName => (
    config?.Services?.[serviceName] ? (
      config.Services[serviceName].name = serviceName,
      config.Services[serviceName]
    ) : null
  ),

  serviceHandlers = new algo.Cache(makeServiceHandler),

  makeServiceConfig = (serviceConfig) => (
    serviceConfig && (
      (
        endpointAttributes = {},
        obj = {
          targetBalancer: serviceConfig.Endpoints && new algo.RoundRobinLoadBalancer(
            shuffle(Object.fromEntries(Object.entries(serviceConfig.Endpoints)
              .map(([k, v]) => (endpointAttributes[k] = v, [k, v.Weight]))
              .filter(([k, v]) => v > 0)
            ))
          ),
          endpointAttributes,
          failoverBalancer: serviceConfig.Endpoints && failover(Object.fromEntries(Object.entries(serviceConfig.Endpoints).map(([k, v]) => [k, v.Weight]))),
        },
      ) => (
        obj
      )
    )()
  ),

  serviceConfigs = new algo.Cache(makeServiceConfig),

  portHandlers = new algo.Cache(
    port => (
      (
        routeRules = config?.RouteRules?.[port],
      ) => (
        routeRules && (new algo.RoundRobinLoadBalancer(routeRules))
      )
    )()
  ),

) => pipy({
  _balancer: null,
  _serviceConfig: null,
  _unhealthCache: null,
  _healthCheckTarget: null,
})

.export('tcp-forward', {
  __service: null,
})

.import({
  __port: 'listener',
  __cert: 'connect-tls',
  __target: 'connect-tcp',
  __metricLabel: 'connect-tcp',
  __healthCheckTargets: 'health-check',
  __healthCheckServices: 'health-check',
})

.pipeline()
.handleStreamStart(
  () => (
    (_balancer = portHandlers.get(__port?.Port)) && (
      (__service = serviceHandlers.get(_balancer.borrow({}).id)) && (
        (_serviceConfig = serviceConfigs.get(__service)) && (
          __metricLabel = __service.name,
          _unhealthCache = __healthCheckServices?.[__service.name],
          (__target = _serviceConfig.targetBalancer?.borrow?.({}, undefined, _unhealthCache)?.id) && (
            (
              attrs = _serviceConfig?.endpointAttributes?.[__target]
            ) => (
              attrs?.UpstreamCert ? (
                __cert = attrs?.UpstreamCert
              ) : (
                __cert = __service?.UpstreamCert
              )
            )
          )()
        )
      )
    )
  )
)
.branch(
  isDebugEnabled, (
    $=>$.handleStreamStart(
      () => (
        console.log('[tcp-forward] port, target, cert:', __port?.Port, __target, Boolean(__cert))
      )
    )
  )
)
.branch(
  () => !__target, (
    $=>$.replaceStreamStart(new StreamEnd)
  ),
  (
    $=>$.branch(
      () => __cert, (
        $=>$.use('lib/connect-tls.js')
      ), (
        $=>$.use('lib/connect-tcp.js')
      )
    )
    .handleStreamEnd(
      e => (
        (_healthCheckTarget = __healthCheckTargets?.[__target + '@' + __service.name]) && (
          (!e.error || e.error === "ReadTimeout" || e.error === "WriteTimeout" || e.error === "IdleTimeout") ? (
            _healthCheckTarget.service.ok(_healthCheckTarget)
          ) : (
            _healthCheckTarget.service.fail(_healthCheckTarget)
          )
        )
      )
    )
  )
)

)()