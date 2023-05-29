((
  config = pipy.solve('config.js'),

  certChain = config?.Certificate?.CertChain,
  privateKey = config?.Certificate?.PrivateKey,
  issuingCA = config?.Certificate?.IssuingCA,

  listIssuingCA = (
    (cas = []) => (
      issuingCA && cas.push(new crypto.Certificate(issuingCA)),
      Object.values(config?.Outbound?.TrafficMatches || {}).map(
        a => a.map(
          o => Object.values(o.DestinationIPRanges || {}).map(
            c => c?.SourceCert?.IssuingCA && (
              cas.push(new crypto.Certificate(c?.SourceCert?.IssuingCA))
            )
          )
        )
      ),
      Object.values(config?.Outbound?.ClustersConfigs || {}).forEach(
        c => c?.SourceCert?.IssuingCA && (
          cas.push(new crypto.Certificate(c?.SourceCert?.IssuingCA))
        )
      ),
      cas
    )
  )(),

  certCache = new algo.Cache(certString => new crypto.Certificate(certString)),
  keyCache = new algo.Cache(keyString => new crypto.PrivateKey(keyString)),
) => (

pipy({
  _cert: null,
  _key: null,
})

.import({
  __cert: 'outbound',
})

.pipeline()
.handleStreamStart(
  () => (
    __cert && (
      _cert = certCache.get(__cert.CertChain),
      _key = keyCache.get(__cert.PrivateKey),
      true
    ) || (
      _cert = certCache.get(certChain),
      _key = keyCache.get(privateKey)
    )
  )
)
.connectTLS({
  certificate: () => ({
    cert: _cert,
    key: _key,
  }),
  trusted: listIssuingCA,
}).to($=>$.use('connect-tcp.js'))

))()