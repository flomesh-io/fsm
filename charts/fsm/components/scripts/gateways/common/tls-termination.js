((
  { config, isDebugEnabled } = pipy.solve('config.js'),

  uniqueCA = {},
  unionCA = (config?.Listeners || []).filter(
    o => o?.TLS?.Certificates
  ).map(
    o => o.TLS.Certificates.map(
      c => c.IssuingCA && (
        (
          md5 = algo.hash(c.IssuingCA)
        ) => (
          !uniqueCA[md5] && (
            uniqueCA[md5] = true,
            new crypto.Certificate(c.IssuingCA)
          )
        )
      )()
    ).filter(
      c => c
    )
  ).flat(),

  globalTls = (config?.Certificate?.CertChain && config?.Certificate?.PrivateKey) ? (
    {
      cert: new crypto.Certificate(config.Certificate.CertChain),
      key: new crypto.PrivateKey(config.Certificate.PrivateKey),
      ca: config?.Certificate?.IssuingCA && unionCA.push(new crypto.Certificate(config.Certificate.IssuingCA)),
    }
  ) : null,

  tlsCache = new algo.Cache(
    portCfg => (
      (
        exactNames = {},
        starNames = {},
      ) => (
        (portCfg?.TLS?.Certificates || []).forEach(
          o => (
            (
              tls = {
                cert: new crypto.Certificate(o.CertChain),
                key: new crypto.PrivateKey(o.PrivateKey),
                ca: o.IssuingCA && (new crypto.Certificate(o.IssuingCA)),
              },
              commonName = tls.cert.subject.commonName.toLowerCase(),
            ) => (
              commonName.startsWith('*') ? (
                starNames[commonName.substring(1)] = tls
              ) : (
                exactNames[commonName] = tls
              ),
              tls.cert.subjectAltNames.forEach(
                o => o.startsWith('*') ? (
                  starNames[o.substring(1).toLowerCase()] = tls
                ) : (
                  exactNames[o.toLowerCase()] = tls
                )
              )
            )
          )()
        ),
        {
          exactNames,
          starNames,
        }
      )
    )()
  ),

  matchDomainName = (portCfg, name) => (
    (
      portTls = tlsCache.get(portCfg),
      domainName = name.toLowerCase(),
      starName,
    ) => (
      portTls.exactNames[domainName] ? portTls.exactNames[domainName] : (
        starName = domainName.substring(domainName.indexOf('.')),
        portTls.starNames[starName] ? portTls.starNames[starName] : globalTls
      )
    )
  )(),

) => pipy({
  _tls: undefined,
  _domainName: undefined,
})

.import({
  __port: 'listener',
  __consumer: 'consumer',
})

.pipeline()
.handleTLSClientHello(
  hello => (
    _domainName = hello?.serverNames?.[0] || '',
    _tls = matchDomainName(__port, _domainName),
    isDebugEnabled && (
      console.log('[tls-termination] port, domainName, CN, Alts:', __port?.Port, _domainName, _tls?.cert?.subject?.commonName, _tls?.cert?.subjectAltNames)
    )
  )
)
.branch(
  () => Boolean(_tls), (
    $=>$.branch(
      () => __port?.TLS?.mTLS, (
        $=>$.acceptTLS({
          certificate: (sni, cert) => (
            __consumer = {sni, cert, mTLS: true, type: 'terminate'},
            {
              cert: _tls.cert,
              key: _tls.key,
            }
          ),
          verify: (ok, cert) => (
            __consumer && (__consumer.cert = cert),
            ok
          ),
          trusted: unionCA,
        }).to($=>$.chain())
      ), (
        $=>$.acceptTLS({
          certificate: (sni, cert) => (
            __consumer = {sni, cert, type: 'terminate'},
            {
              cert: _tls.cert,
              key: _tls.key,
            }
          ),
        }).to($=>$.chain())
      )
    )
  ),
  () => _tls === null, (
    $=>$.replaceStreamStart(
      () => (
        isDebugEnabled && (
          console.log('[tls-termination] Not match TLS cert, _domainName:', _domainName)
        ),
        new StreamEnd
      )
    )
  )
)

)()