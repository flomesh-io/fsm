((
  { config, isDebugEnabled } = pipy.solve('config.js'),

  { healthCheckTargets, healthCheckServices } = pipy.solve('common/variables.js'),

  hcLogging = config?.Configs?.HealthCheckLog?.StorageAddress && new logging.JSONLogger('health-check-logger').toHTTP(config.Configs.HealthCheckLog.StorageAddress, {
    batch: {
      timeout: 1,
      interval: 1,
      prefix: '[',
      postfix: ']',
      separator: ','
    },
    headers: {
      'Content-Type': 'application/json',
      'Authorization': config.Configs.HealthCheckLog.Authorization || ''
    }
  }).log,

  k8s_cluster = os.env.PIPY_K8S_CLUSTER || '',
  code_base = pipy.source || '',
  pipy_id = pipy.name || '',

  { metrics } = pipy.solve('lib/metrics.js'),

  makeHealthCheck = (serviceConfig) => (
    serviceConfig?.HealthCheck && (
      (
        name = serviceConfig.name,
        interval = serviceConfig.HealthCheck?.Interval, // || 15, active
        maxFails = serviceConfig.HealthCheck?.MaxFails, // || 3, both
        failTimeout = serviceConfig.HealthCheck?.FailTimeout, // || 300, passivity
        path = serviceConfig.HealthCheck?.Path, // for HTTP
        matches = serviceConfig.HealthCheck?.Matches || [{ StatusCodes: [200] }], // for HTTP
        type = path ? 'HTTP' : 'TCP',
      ) => (
        {
          name,
          interval,
          maxFails,
          failTimeout,
          path,
          matches,

          toString: () => (
            'service: ' + name + ', interval: ' + interval + ', maxFails: ' + maxFails + ', path: ' + path + ', matches: ' + matches
          ),

          ok: target => (
            (target.alive === 0) ? (
              target.alive = 1,
              target.errorCount = 0,
              healthCheckServices[name] && healthCheckServices[name].get(target.target) && (
                healthCheckServices[name].remove(target.target)
              ),
              isDebugEnabled && (
                console.log('[health-check] ok - service, type, target:', name, type, target)
              ),
              _changed = 1
            ) : (
              _changed = 0
            ),
            metrics.fgwUpstreamStatus.withLabels(
              name,
              target.ip,
              target.port,
              target.reason = 'ok',
              target.http_status || '',
              _changed
            ).increase(),
            hcLogging?.({
              k8s_cluster,
              code_base,
              pipy_id,
              upstream_ip: target.ip,
              upstream_port: target.port,
              type: 'ok',
              http_status: target.http_status || '',
              change_status: _changed
            })
          ),

          fail: target => (
            (++target.errorCount >= maxFails && target.alive) ? (
              target.alive = 0,
              target.failTick = 0,
              !healthCheckServices[name] ? (
                healthCheckServices[name] = new algo.Cache(),
                healthCheckServices[name].set(target.target, true)
              ) : (
                !healthCheckServices[name].get(target.target) && (
                  healthCheckServices[name].set(target.target, true)
                )
              ),
              isDebugEnabled && (
                console.log('[health-check] fail - service, type, target:', name, type, target)
              ),
              _changed = -1
            ) : (
              _changed = 0
            ),
            metrics.fgwUpstreamStatus.withLabels(
              name,
              target.ip,
              target.port,
              target.reason || 'fail',
              target.http_status || '',
              _changed
            ).decrease(),
            hcLogging?.({
              k8s_cluster,
              code_base,
              pipy_id,
              upstream_ip: target.ip,
              upstream_port: target.port,
              type: target.reason || 'fail',
              http_status: target.http_status || '',
              change_status: _changed
            })
          ),

          available: target => (
            target.alive > 0
          ),

          match: (
            (
              rules,
              match_rules = matches.map(
                m => (
                  rules = [],
                  m?.StatusCodes && (
                    rules.push(
                      msg => (
                        m.StatusCodes.includes(msg?.head?.status) || -1
                      )
                    )
                  ),
                  m?.Body && (
                    rules.push(
                      msg => (
                        msg?.body?.toString?.()?.includes(m.Body) || -2
                      )
                    )
                  ),
                  m?.Headers && (
                    rules.push(
                      Object.entries(m.Headers).map(
                        ([k, v]) => (
                          msg => (
                            (msg?.head?.headers?.[k.toLowerCase?.()] == v) || -3
                          )
                        )
                      )
                    )
                  ),
                  (rules.length === 0) ? (
                    () => -100
                  ) : (
                    rules.flat()
                  )
                )
              ).flat()
            ) => (
              msg => (
                (
                  err = 0
                ) => (
                  match_rules.every(
                    m => (
                      (err = m(msg)) > 0
                    )
                  ) || err
                )
              )()
            )
          )(),

          check: target => (
            new http.Agent(target.target).request('GET', path).then(
              result => (
                (
                  code = target.service.match(result)
                ) => (
                  target.http_status = result?.head?.status,
                  (code > 0) ? (
                    target.service.ok(target)
                  ) : (
                    (code == -1) ? (
                      target.reason = "Bad-StatusCode"
                    ) : (code == -2) ? (
                      target.reason = "Bad-Body"
                    ) : (code == -3) ? (
                      target.reason = "Bad-Header"
                    ) : (
                      target.reason = "Bad-Matches"
                    ),
                    target.service.fail(target)
                  ),
                  {}
                )
              )()
            )
          ),
        }
      )
    )()
  ),

  healthCheckCache = new algo.Cache(makeHealthCheck),
) => pipy({
  _idx: 0,
  _changed: 0,
  _service: null,
  _target: null,
  _resolve: null,
  _tcpTargets: null,
  _targetPromises: null,
})

.pipeline()
.onStart(
  () => void (
    Object.keys(config?.Services || {}).forEach(
      name => (
        (config.Services[name]?.HealthCheck?.MaxFails > 0) && (
          config.Services[name].HealthCheck.Interval > 0 || config.Services[name].HealthCheck.FailTimeout > 0
        ) && (
          config.Services[name].name = name,
          (_service = healthCheckCache.get(config.Services[name])) && (
            Object.keys(config.Services[name].Endpoints || {}).forEach(
              target => (
                _idx = target.lastIndexOf(':'),
                healthCheckTargets[target + '@' + name] = {
                  ip: target.substring(0, _idx),
                  port: target.substring(_idx + 1),
                  target,
                  service: _service,
                  alive: 1,
                  errorCount: 0,
                  failTick: 0,
                  tick: 0,
                }
              )
            )
          )
        )
      )
    )
  )
)

.task('1s')
.onStart(
  () => new Message
)
.replaceMessage(
  msg => (
    _tcpTargets = [],
    _targetPromises = [],
    Object.values(healthCheckTargets).forEach(
      target => (
        (target.service.interval > 0 && ++target.tick >= target.service.interval) && (
          target.tick = 0,
          target.service.path ? ( // for HTTP
            target.service.check(target)
          ) : ( // for TCP
            _targetPromises.push(new Promise(r => _resolve = r)),
            _tcpTargets.push(new Message({ target, resolve: _resolve }))
          )
        )
      )
    ),
    _tcpTargets.length > 0 ? _tcpTargets : msg
  )
)
.branch(
  () => _tcpTargets.length > 0, (
    $=>$
    .demux().to(
      $=>$.replaceMessage(
        msg => (
          _target = msg.head.target,
          _resolve = msg.head.resolve,
          new Data
        )
      )
      .connect(() => _target.target,
        {
          connectTimeout: 0.1,
          readTimeout: 0.1,
          idleTimeout: 0.1,
        }
      )
      .replaceData(
        () => new Data
      )
      .replaceStreamEnd(
        e => (
          (!e.error || e.error === "ReadTimeout" || e.error === "IdleTimeout") ? (
            _target.service.ok(_target)
          ) : (
            _target.reason = 'ConnectionRefused',
            _target.service.fail(_target)
          ),
          _resolve(),
          new Message
        )
      )
    )
    .wait(
      () => Promise.all(_targetPromises)
    )
  ), (
    $=>$
  )
)
.replaceMessage(
  () => new StreamEnd
)

.task('1s')
.onStart(
  () => new Message
)
.replaceMessage(
  () => (
    Object.values(healthCheckTargets).forEach(
      target => (
        (target.alive === 0 && target.service.failTimeout > 0 && !(target.service.interval > 0)) && (
          (++target.failTick >= target.service.failTimeout) && (
            isDebugEnabled && (
              console.log('[health-check] reset - service, type, target, failTick:', target.name, type, target.target, target.failTick)
            ),
            target.service.ok(target)
          )
        )
      )
    ),
    new StreamEnd
  )
)

)()