import { findPolicies } from '../utils.js'
import { log } from '../log.js'

export default function (config, backendRef, backendResource) {
  var healthCheckPolicies = findPolicies(config, 'HealthCheckPolicy', backendResource)
  if (healthCheckPolicies.length === 0) {
    return { isHealthy: () => true }
  }

  var targets = backendResource.spec.targets
  var ports = healthCheckPolicies[0].spec.ports

  var unhealthyCache = new algo.Cache
  var allTargets = []

  if (pipy.thread.id === 0) {
    ports.forEach(({ port, healthCheck }) => {
      var checkPort = port
      targets.forEach(({ address, port }) => {
        if (port != checkPort) return

        var targetAddress = `${address}:${port}`
        var isHealthy = true
        var failCount = 0
        var failTime = 0
        var lastCheckTime = Date.now() / 1000

        var matches = healthCheck.matches.map(
          m => {
            if (m.statusCodes) return res => m.statusCodes.some(code => code == res.head.status)
            if (m.body) return res => res.body?.toString?.() === m.body
            if (m.headers) {
              var headers = Object.entries(m.headers).map(([k, v]) => [k.toLowerCase(), v])
              return res => {
                var h = res.head.headers
                return !headers.some(([k, v]) => h[k] !== v)
              }
            }
            return () => true
          }
        )

        if (healthCheck.path) {
          var hcPipeline = pipeline($=>$
            .onStart(new Message({ path: healthCheck.path }))
            .encodeHTTPRequest()
            .connect(targetAddress, { connectTimeout: 5, idleTimeout: 5 })
            .decodeHTTPResponse()
            .handleMessage(
              function (res, i) {
                if (i > 0) return
                if (matches.some(f => !f(res))) {
                  fail()
                } else {
                  reset()
                }
              }
            )
          )
        } else {
          var hcPipeline = pipeline($=>$
            .onStart(new Data)
            .connect(checkAddress, { connectTimeout: 5, idleTimeout: 5 })
            .handleStreamEnd(
              function (eos) {
                if (eos.error) {
                  fail()
                } else {
                  reset()
                }
              }
            )
          )
        }

        function reset() {
          if (!isHealthy) {
            log?.(`Health backend ${backendResource.metadata.name} reset ${targetAddress}`)
          }
          isHealthy = true
          failCount = 0
          failTime = 0
          unhealthyCache.remove(targetAddress, true)
        }

        function fail() {
          failCount++
          failTime = Date.now() / 1000
          if (failCount >= healthCheck.maxFails) {
            isHealthy = false
            unhealthyCache.set(targetAddress, true)
            log?.(`Health backend ${backendResource.metadata.name} down ${targetAddress}`)
          } else {
            log?.(`Health backend ${backendResource.metadata.name} fail ${targetAddress}`)
          }
        }

        function check(t) {
          if (healthCheck.interval) {
            if (t - lastCheckTime >= healthCheck.interval) {
              lastCheckTime = t
              return hcPipeline.spawn()
            }
          } else if (!isHealthy) {
            if (t - failTime >= healthCheck.failTimeout) {
              reset()
            }
          }
          return Promise.resolve()
        }

        allTargets.push({
          address: targetAddress,
          isHealthy: () => isHealthy,
          reset, fail, check,
        })
      })
    })
  
    function healthCheckAll() {
      Promise.all(allTargets.map(
        target => target.check(Date.now() / 1000)
      )).then(
        () => new Timeout(1).wait()
      ).then(healthCheckAll)
    }
  
    healthCheckAll()
  }

  function isHealthy(target) {
    return !unhealthyCache.has(target)
  }

  return { isHealthy }
}
