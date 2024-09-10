import resources from '../resources.js'
import { log, findPolicies } from '../utils.js'

var cache = new algo.Cache(backendName => {
  var healthCheckPolicy = null
  var backendResource = null
  var allTargets = []
  var unhealthySet = new algo.SharedMap('HealthCheck:' + backendName)

  function watch() {
    resources.setUpdater('HealthCheckPolicy', backendName, update)
    resources.addUpdater('Backend', backendName, update)
  }

  function update() {
    healthCheckPolicy = null
    backendResource = findBackendResource(backendName)
    if (backendResource) {
      var policies = findPolicies('HealthCheckPolicy', backendResource)
      healthCheckPolicy = policies[0]
      allTargets = []
      findTargets()
      watch()
    } else {
      cache.remove(backendName)
    }
  }

  function findTargets() {
    if (!healthCheckPolicy) return
    var targets = backendResource.spec.targets
    var ports = healthCheckPolicy.spec.ports
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
          unhealthySet.delete(targetAddress)
        }

        function fail() {
          failCount++
          failTime = Date.now() / 1000
          if (failCount >= healthCheck.maxFails) {
            isHealthy = false
            unhealthySet.set(targetAddress, true)
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
  }

  function healthCheckAll() {
    if (!healthCheckPolicy) return
    Promise.all(allTargets.map(
      target => target.check(Date.now() / 1000)
    )).then(
      () => new Timeout(1).wait()
    ).then(healthCheckAll)
  }

  if (pipy.thread.id === 0) {
    update()
    healthCheckAll()
  }

  function isHealthy(target) {
    return !healthCheckPolicy || !unhealthySet.has(target)
  }

  return { isHealthy }
})

function findBackendResource(backendName) {
  return resources.list('Backend').find(
    r => r.metadata?.name === backendName
  )
}

export default function(backendRef, backendResource) {
  return cache.get(backendResource.metadata.name)
}
