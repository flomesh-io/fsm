((
  config = pipy.solve('config.js'),

  certChain = config?.Certificate?.CertChain,
  privateKey = config?.Certificate?.PrivateKey,
  issuingCA = config?.Certificate?.IssuingCA,

  sourceIPRangesCache = new algo.Cache((sourceIPRanges) => (
    sourceIPRanges ? (
      Object.entries(sourceIPRanges).map(
        (
          [k, v]
        ) => (
          {
            netmask: new Netmask(k),
            mTLS: v?.mTLS,
            skipClientCertValidation: v?.SkipClientCertValidation,
            authenticatedPrincipals: v?.AuthenticatedPrincipals && Object.fromEntries(v.AuthenticatedPrincipals.map(e => [e, true])),
          }
        )
      )
    ) : null
  )),

) => pipy({
  _tlsConfig: null,
  _forbiddenTLS: false,
})

.import({
  __port: 'inbound',
  __protocol: 'inbound',
})

.pipeline()
.branch(
  () => certChain && __port && (
    _tlsConfig = sourceIPRangesCache.get(__port?.SourceIPRanges)?.find?.(o => o.netmask.contains(__inbound.remoteAddress || '127.0.0.1')),
    !_tlsConfig || _tlsConfig?.mTLS), (
    $=>$.acceptTLS({
      certificate: () => ({
        cert: new crypto.Certificate(certChain),
        key: new crypto.PrivateKey(privateKey),
      }),
      trusted: issuingCA ? [new crypto.Certificate(issuingCA)] : [],
      verify: (ok, cert) => (
        _tlsConfig?.mTLS && !_tlsConfig?.skipClientCertValidation && (
          _tlsConfig?.authenticatedPrincipals && (_forbiddenTLS = true),
          (_tlsConfig?.authenticatedPrincipals?.[cert?.subject?.commonName] || (
            cert?.subjectAltNames && cert.subjectAltNames.find(o => _tlsConfig?.authenticatedPrincipals?.[o]))) && (
            _forbiddenTLS = false
          ),
          _forbiddenTLS && (__protocol !== 'http') && (
            ok = false
          )
        ),
        ok
      )
    }).to(
      $=>$.branch(
        () => _forbiddenTLS, (
          $=>$.demuxHTTP().to(
            $=>$.replaceMessage(
              new Message({ status: 403 }, 'Access denied')
            )
          )
        ), (
          $=>$.chain()
        )
      )
    )
  ), (
    $=>$.chain()
  )
)

)()