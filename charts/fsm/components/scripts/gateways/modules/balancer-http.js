import makeBackend from './backend.js'
import makeBackendTLS from './backend-tls.js'
import makeSessionPersistence from './session-persistence.js'
import makeHealthCheck from './health-check.js'
import { log, stringifyHTTPHeaders, findPolicies } from '../utils.js'

var $ctx
var $session
var $conn

export default function (backendRef, backendResource, gateway, isHTTP2) {
  var backend = makeBackend(backendResource.metadata.name)
  var balancer = backend.balancer
  var hc = makeHealthCheck(backendRef, backendResource)
  var tls = makeBackendTLS(backendRef, backendResource, gateway)

  var backendLBPolicies = findPolicies('BackendLBPolicy', backendResource)
  var sessionPersistenceConfig = backendLBPolicies.find(r => r.spec.sessionPersistence)?.spec?.sessionPersistence
  var sessionPersistence = sessionPersistenceConfig && makeSessionPersistence(sessionPersistenceConfig)

  if (sessionPersistence) {
    var restoreSession = sessionPersistence.restore
    var targetSelector = function (req) {
      $session = balancer.allocate(
        restoreSession(req.head),
        target => hc.isHealthy(target.address)
      )
    }
  } else {
    var targetSelector = function () {
      $session = balancer.allocate(null, target => hc.isHealthy(target.address))
    }
  }

  return pipeline($=>{
    $.onStart(c => { $ctx = c })
    $.pipe(evt => {
      if (evt instanceof MessageStart) {
        $ctx.backend = backend
        targetSelector(evt)
        log?.(
          `Inb #${$ctx.parent.inbound.id} Req #${$ctx.id}`, evt.head.method, evt.head.path,
          `forward ${$session?.target?.address}`,
          `headers ${stringifyHTTPHeaders(evt.head.headers)}`,
        )
        return $session ? forward : reject
      }
    })
    $.handleMessageStart(res => {
      var r = $ctx.response
      r.head = res.head
      r.headTime = Date.now()
    })
    $.handleMessageEnd(res => {
      var r = $ctx.response
      r.tail = res.tail
      r.tailTime = res.tailTime
    })

    if (log) {
      $.handleMessageStart(
        res => log?.(
          `Inb #${$ctx.parent.inbound.id} Req #${$ctx.id}`, $ctx.head.method, $ctx.head.path,
          `return ${res.head.status} ${res.head.statusText}`,
          `headers ${stringifyHTTPHeaders(res.head.headers)}`,
        )
      )
    }

    var reject = pipeline($=>$
      .replaceMessage(
        new Message({ status: 500 })
      )
    )

    var forward = pipeline($=>{
      $.onStart(() => {
        $ctx.sendTime = Date.now()
        $ctx.target = $session.target.address
        $conn = {
          protocol: 'tcp',
          target: $session.target,
        }
      })
      $.muxHTTP(() => $session, { version: () => isHTTP2 || $session.target.protocol === 'h2c' ? 2 : 1 }).to($=>{
        if (tls) {
          $.connectTLS({
            ...tls,
            onState: session => {
              if (session.error) {
                log?.(`Inb #${$ctx.parent.inbound.id} Req #${$ctx.id} tls error:`, session.error)
              }
            }
          }).to($=>$
            .pipe(backend.connect, () => $conn)
          )
        } else {
          $.pipe(backend.connect, () => $conn)
        }
      })

      if (sessionPersistence) {
        var preserveSession = sessionPersistence.preserve
        $.handleMessageStart(
          res => preserveSession(res.head, $session.target.address)
        )
      }

      $.onEnd(() => $session.free())
    })
  })
}
