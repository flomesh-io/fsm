import resources from '../resources.js'
import { findPolicies } from '../utils.js'

var cache = new algo.Cache(
  backendName => {
    var ztm = resources.ztm
    var backendResource = findBackendResource(backendName)
    var backendLBPolicies = findPolicies('BackendLBPolicy', backendResource)
    var algorithm = backendLBPolicies.find(r => r.spec.algorithm)?.spec?.algorithm
    var targets = getTargets(backendResource)

    if (algorithm === 'LeastLoad') {
      algorithm = 'least-load'
    } else {
      algorithm = 'round-robin'
    }

    var balancer = new algo.LoadBalancer(
      targets, {
        algorithm,
        key: t => t.address,
        weight: t => t.weight,
      }
    )

    var $protocol
    var $target
    var $endpoint

    var connect = pipeline($=>{
      $.onStart(({ protocol, target }) => {
        $protocol = protocol
        $target = target
        backend.concurrency++
        target.concurrency++
        if (ztm) {
          var ep = backendResource?.spec?.ztm?.endpoint
          var id = ep?.id
          if (id) {
            $endpoint = id
          } else {
            var name = ep?.name
            if (name) {
              return ztm.mesh.discover().then(endpoints => {
                var epList = endpoints.filter(ep => ep.name === name)
                if (epList.length > 0) {
                  $endpoint = epList[Math.floor(Math.random() * epList.length)].id
                } else {
                  ztm.app.log(`Endpoint ${name} not found for backend ${backendName}`)
                }
              })
            }
          }
          if (!$endpoint) {
            ztm.app.log(`Endpoints not found for backend ${backendName}`)
          }
        }
      })

      if (ztm) {
        $.pipe(() => {
          var isUDP = ($protocol === 'udp')
          if (!$endpoint) return 'deny'
          if ($endpoint === ztm.app.endpoint.id) return isUDP ? 'selfUDP' : 'selfTCP'
          return isUDP ? 'peerUDP' : 'peerTCP'
        }, {
          'peerTCP': ($=>$
            .connectHTTPTunnel(() => new Message({
              method: 'CONNECT',
              path: `/backends/tcp/${backendName}/${$target.address}`,
            })).to($=>$
              .muxHTTP({ version: 2 }).to($=>$
                .pipe(() => ztm.mesh.connect($endpoint))
              )
            )
          ),
          'peerUDP': ($=>$
            .replaceData(data => new Message(data))
            .encodeWebSocket()
            .connectHTTPTunnel(() => new Message({
              method: 'CONNECT',
              path: `/backends/udp/${backendName}/${$target.address}`,
            })).to($=>$
              .muxHTTP({ version: 2 }).to($=>$
                .pipe(() => ztm.mesh.connect($endpoint))
              )
            )
            .decodeWebSocket()
            .replaceMessage(msg => msg.body)
          ),
          'selfTCP': $=>$.connect(() => $target.address),
          'selfUDP': $=>$.connect(() => $target.address, { protocol: 'udp' }),
          'deny': $=>$.replaceStreamStart(new StreamEnd)
        })
      } else {
        $.pipe(() => $protocol, {
          'tcp': $=>$.connect(() => $target.address),
          'udp': $=>$.connect(() => $target.address, { protocol: 'udp' }),
        })
      }

      $.onEnd(() => {
        $target.concurrency--
        backend.concurrency--
      })
    })

    var backend = {
      name: backendName,
      concurrency: 0,
      targets,
      balancer,
      connect,
    }

    function watch() {
      resources.addUpdater('Backend', backendName, () => {
        backendResource = findBackendResource(backendName)
        if (backendResource) {
          targets = getTargets(backendResource)
          balancer.provision(targets)
          watch()
        } else {
          cache.remove(backendName)
        }
      })
    }

    watch()

    return backend
  }
)

function findBackendResource(backendName) {
  return resources.list('Backend').find(
    r => r.metadata?.name === backendName
  )
}

function getTargets(backendResource) {
  if (!backendResource?.spec?.targets) return []
  return backendResource.spec.targets.map(t => {
    var port = t.port || backendRef.port
    var address = `${t.address}:${port}`
    var protocol = t.appProtocol || backendResource.spec.appProtocol
    var weight = t.weight
    return { address, protocol, weight, concurrency: 0 }
  })
}

export default function (backendName) {
  return cache.get(backendName)
}
