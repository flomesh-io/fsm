import makeHealthCheck from './health-check.js'
import makeBackendTLS from './backend-tls.js'
import makeSessionPersistence from './session-persistence.js'
import { stringifyHTTPHeaders, findPolicies } from '../utils.js'
import { log } from '../log.js'

var $ctx
var $selection

export default function (config, backendRef, backendResource, isHTTP2) {
  var hc = makeHealthCheck(config, backendRef, backendResource)
  var tls = makeBackendTLS(config, backendRef, backendResource)

  var targets = backendResource.spec.targets.map(t => {
    var port = t.port || backendRef.port
    var address = `${t.address}:${port}`
    var weight = t.weight
    return { address, weight }
  })

  var loadBalancer = new algo.LoadBalancer(
    targets, {
      key: t => t.address,
      weight: t => t.weight,
    }
  )

  var backendLBPolicies = findPolicies(config, 'BackendLBPolicy', backendResource)
  var sessionPersistenceConfig = backendLBPolicies.find(r => r.spec.sessionPersistence)?.spec?.sessionPersistence
  var sessionPersistence = sessionPersistenceConfig && makeSessionPersistence(sessionPersistenceConfig)

  var retryPolices = findPolicies(config, 'RetryPolicy', backendResource)
  var retryConfig = retryPolices?.[0]?.spec?.retry

  if (sessionPersistence) {
    var restoreSession = sessionPersistence.restore
    var targetSelector = function (req) {
      $selection = loadBalancer.allocate(
        restoreSession(req.head),
        target => hc.isHealthy(target.address)
      )
    }
  } else {
    var targetSelector = function () {
      $selection = loadBalancer.allocate(null, target => hc.isHealthy(target.address))
    }
  }

  return pipeline($=>{
    $.onStart(c => void ($ctx = c))
    $.pipe(evt => {
      if (evt instanceof MessageStart) {
        targetSelector(evt)
        log?.(
          `Inb #${$ctx.parent.inbound.id} Req #${$ctx.id}`, evt.head.method, evt.head.path,
          `forward ${$selection?.target?.address}`,
          `headers ${stringifyHTTPHeaders(evt.head.headers)}`,
        )
        return $selection ? forward : reject
      }
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
      $.muxHTTP(() => $selection, { version: isHTTP2 ? 2 : 1 }).to($=>{
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
          res => preserveSession(res.head, $selection.target.address)
        )
      }

      $.onEnd(() => $selection.free())
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
          .replaceMessageStart(
            function (msg) {
              if (retryCodes[msg.head.status]) {
                if (++$retryCounter < retryConfig.numRetries) {
                  log?.(`Inb #${$ctx.parent.inbound.id} Req #${$ctx.id} retry ${$retryCounter} status ${msg.head.status}`)
                  return new StreamEnd
                } else {
                  log?.(`Inb #${$ctx.parent.inbound.id} Req #${$ctx.id} retry ${$retryCounter} status ${msg.head.status} gave up`)
                  $retryCounter = 0
                }
              }
              return msg
            }
          )
        )
      )
    }

    function connect($) {
      $.connect(() => $selection.target.address)
    }
  })
}
