import { log } from '../log.js'

var $ctx
var $proto
var $hello = false

export default function (config, listener) {
  var fullnames = {}
  var postfixes = []

  listener.tls?.certificates?.forEach?.(c => {
    var crtFile = config.secrets[c['tls.crt']]
    var keyFile = config.secrets[c['tls.key']]
    var crt = new crypto.Certificate(crtFile)
    var key = new crypto.PrivateKey(keyFile)
    var certificate = { cert: crt, key }
    ;[crt.subject.commonName, ... crt.subjectAltNames].forEach(
      n => {
        if (n.startsWith('*')) {
          postfixes.push([n.substring(1).toLowerCase(), certificate])
        } else {
          fullnames[n.toLowerCase()] = certificate
        }
      }
    )
  })

  var trusted
  var frontendValidation = listener.tls?.frontendValidation
  if (frontendValidation) {
    trusted = []
    frontendValidation.caCertificates?.forEach?.(c => {
      var crtFile = config.secrets[c['ca.crt']]
      var crt = new crypto.Certificate(crtFile)
      trusted.push(crt)
    })
  }

  function findCertificate(hello) {
    var sni = hello.serverNames?.[0] || ''
    var name = sni.toLowerCase()
    var certificate = fullnames[name] || postfixes.find(([postfix]) => name.endsWith(postfix))?.[1]
    $ctx.serverName = sni
    $ctx.serverCert = certificate || null
    $hello = true
    log?.(
      `Inb #${$ctx.inbound.id}`,
      `sni ${sni} cert`, $ctx.serverCert?.cert?.subject || null
    )
  }

  return pipeline($=>$
    .onStart(c => void ($ctx = c))
    .detectProtocol(proto => void ($proto = proto))
    .pipe(
      () => {
        if ($proto !== undefined) {
          log?.(`Inb #${$ctx.inbound.id} protocol ${$proto || 'unknown'}`)
          return $proto === 'TLS' ? 'pass' : 'deny'
        }
      }, {
        'pass': ($=>$
          .handleTLSClientHello(findCertificate)
          .pipe(
            () => {
              if ($hello) {
                return $ctx.serverCert ? 'pass' : 'deny'
              }
            }, {
              'pass': ($=>$
                .acceptTLS({
                  certificate: () => $ctx.serverCert,
                  trusted,
                  onState: session => {
                    if (session.state === 'connected') {
                      $ctx.clientCert = session.peer
                    } else if (session.error) {
                      log?.(`Inb #${$ctx.inbound.id} tls error:`, session.error)
                    }
                  }
                }).to($=>$.pipeNext())
              ),
              'deny': $=>$.replaceStreamStart(new StreamEnd)
            }
          )
        ),
        'deny': $=>$.replaceStreamStart(new StreamEnd)
      }
    )
  )
}
