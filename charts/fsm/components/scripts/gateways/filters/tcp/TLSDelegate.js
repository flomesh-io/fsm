export default function ({ tlsDelegate }, resources) {
  var ca = new crypto.Certificate(resources.secrets[tlsDelegate.certificate['ca.crt']])
  var caKey = new crypto.PrivateKey(resources.secrets[tlsDelegate.certificate['ca.key']])
  var key = new crypto.PrivateKey({ type: 'rsa', bits: 2048 })
  var pkey = new crypto.PublicKey(key)

  var cache = new algo.Cache(
    domain => new crypto.Certificate({
      subject: { CN: domain },
      extensions: { subjectAltName: `DNS:${domain}` },
      days: 365,
      timeOffset: -3600,
      issuer: ca,
      privateKey: caKey,
      publicKey: pkey,
    }),
    null, { ttl: 60*60 }
  )

  var $ctx
  var $proto

  return pipeline($=>$
    .onStart(c => { $ctx = c })
    .pipe(evt => evt instanceof MessageStart ? 'L7' : 'L4', {
      'L7': ($=>$.pipeNext()),
      'L4': ($=>$
        .detectProtocol(p => { $proto = p })
        .pipe(
          () => {
            if ($proto !== undefined) {
              return ($proto === 'TLS' ? 'tls' : 'tcp')
            }
          }, {
            'tcp': ($=>$.pipeNext()),
            'tls': ($=>$
              .acceptTLS({
                certificate: s => {
                  if (!s) return
                  $ctx.originalServerName = s
                  return { key, cert: cache.get(s) }
                }
              }).to($=>$.pipeNext())
            )
          }
        )
      )
    })
  )
}
