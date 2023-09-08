((
  { config } = pipy.solve('config.js'),

  uniqueCA = {},

  addCA = cert => (
    (
      md5 = algo.hash(cert)
    ) => (
      !uniqueCA[md5] && (
        uniqueCA[md5] = true,
        new crypto.Certificate(cert)
      )
    )
  )(),

  unionCA = Object.values(config?.Services || {}).map(
    o => (
      (
        ca,
        cas = [],
      ) => (
        o?.UpstreamCert?.IssuingCA && (ca = addCA(o.UpstreamCert.IssuingCA)) && (
          cas.push(ca)
        ),
        Object.values(o?.Endpoints || {}).map(
          e => (
            e?.UpstreamCert?.IssuingCA && (ca = addCA(e.UpstreamCert.IssuingCA)) && (
              cas.push(ca)
            )
          )
        ),
        cas
      )
    )()
  )
  .flat()
  .filter(e => e),

  globalTls = (config?.Certificate?.CertChain && config?.Certificate?.PrivateKey) ? (
    {
      cert: new crypto.Certificate(config.Certificate.CertChain),
      key: new crypto.PrivateKey(config.Certificate.PrivateKey),
      ca: config?.Certificate?.IssuingCA && unionCA.push(new crypto.Certificate(config.Certificate.IssuingCA)),
    }
  ) : null,

  certCache = new algo.Cache(
    cert => (
      cert?.CertChain && cert?.PrivateKey && (
        {
          crt: new crypto.Certificate(cert.CertChain),
          key: new crypto.PrivateKey(cert.PrivateKey),
        }
      )
    )
  ),

) => pipy({
  _tls: null,
})

.export('connect-tls', {
  __cert: null,
})

.pipeline()
.branch(
  () => (_tls = certCache.get(__cert)), (
    $=>$.connectTLS({
      certificate: () => ({
        cert: _tls.crt,
        key: _tls.key,
      }),
      trusted: unionCA,
      alpn: 'h2',
    }).to($=>$.use('lib/connect-tcp.js'))
  ), (
    $=>$.use('lib/connect-tcp.js')
  )
)

)()