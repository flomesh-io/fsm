/*
 * MIT License
 *
 * Copyright (c) since 2021,  flomesh.io Authors.
 *
 * Permission is hereby granted, free of charge, to any person obtaining a copy
 * of this software and associated documentation files (the "Software"), to deal
 * in the Software without restriction, including without limitation the rights
 * to use, copy, modify, merge, publish, distribute, sublicense, and/or sell
 * copies of the Software, and to permit persons to whom the Software is
 * furnished to do so, subject to the following conditions:
 *
 * The above copyright notice and this permission notice shall be included in all
 * copies or substantial portions of the Software.
 *
 * THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
 * IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY,
 * FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE
 * AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER
 * LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM,
 * OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN THE
 * SOFTWARE.
 */

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