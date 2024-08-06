import makeBackend from './backend.js'
import makeBackendTLS from './backend-tls.js'
import makeSessionPersistence from './session-persistence.js'
import makeHealthCheck from './health-check.js'
import { log, stringifyHTTPHeaders, findPolicies } from '../utils.js'

var $ctx
var $session

export default function (backendRef, backendResource, isHTTP2) {
  var name = backendResource.metadata.name
  var backend = makeBackend(name)
  var balancer = backend.balancer
  var hc = makeHealthCheck(backendRef, backendResource)
  var tls = makeBackendTLS(backendRef, backendResource)

  var backendLBPolicies = findPolicies('BackendLBPolicy', backendResource)
  var sessionPersistenceConfig = backendLBPolicies.find(r => r.spec.sessionPersistence)?.spec?.sessionPersistence
  var sessionPersistence = sessionPersistenceConfig && makeSessionPersistence(sessionPersistenceConfig)

  var retryPolices = findPolicies('RetryPolicy', backendResource)
  var retryConfig = retryPolices?.[0]?.spec?.retry

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
      })
      $.muxHTTP(() => $session, { version: isHTTP2 ? 2 : 1 }).to($=>{
        if (tls) {
          $.connectTLS({
            ...tls,
            onState: session => {
              if (session.error) {
                log?.(`Inb #${$ctx.parent.inbound.id} Req #${$ctx.id} tls error:`, session.error)
              }
            }
          }).to(connect)
        } else {
          connect($)
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

    if (retryConfig) {
      var retryCodes = {}
      ;(retryConfig.retryOn || ['5xx']).forEach(code => {
        if (code.toString().substring(1) === 'xx') {
          var base = Number.parseInt(code.toString().charAt(0) + '00')
          new Array(100).fill().forEach((_, i) => retryCodes[base + i] = true)
        } else {
          retryCodes[Number.parseInt(code)] = true
        }
      })

      var $retryCounter = 0
      forward = pipeline($=>$
        .repeat(() => {
          if ($retryCounter > 0) {
            return new Timeout(retryConfig.backoffBaseInterval * Math.pow(2, $retryCounter - 1)).wait().then(true)
          } else {
            return false
          }
        }).to($=>$
          .pipe(forward)
          .pipe(evt => {
            if (evt instanceof MessageStart) {
              var needRetry = (evt.head.status in retryCodes)
              var wasRetry = ($retryCounter > 0)
              if (wasRetry) {
                $ctx.retries.push({
                  target: $ctx.target,
                  succeeded: !needRetry,
                })
              }
              if (needRetry) {
                if (++$retryCounter < retryConfig.numRetries) {
                  log?.(`Inb #${$ctx.parent.inbound.id} Req #${$ctx.id} retry ${$retryCounter} status ${evt.head.status}`)
                  return 'retry'
                } else {
                  log?.(`Inb #${$ctx.parent.inbound.id} Req #${$ctx.id} retry ${$retryCounter} status ${evt.head.status} gave up`)
                  $retryCounter = 0
                  return 'conclude'
                }
              } else {
                return 'conclude'
              }
            }
          }, {
            'retry': $=>$.replaceData().replaceMessage(new StreamEnd),
            'conclude': $=>$,
          })
        )
      )
    }

    function connect($) {
      $.onStart(() => {
        var t = (backend.targets[$session.target.address] ??= { concurrency: 0 })
        t.concurrency++
      })
      $.connect(() => $session.target.address)
      $.onEnd(() => {
        backend.targets[$session.target.address].concurrency--
        backend.concurrency--
      })
    }
  })
}
