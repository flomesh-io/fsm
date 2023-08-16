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
    ingress = pipy.solve('ingress.js'),
    upstreamMapIssuingCA = {},
    upstreamIssuingCAs = [],
    addUpstreamIssuingCA = (ca) => (
      (md5 => (
        md5 = '' + algo.hash(ca),
        !upstreamMapIssuingCA[md5] && (
          upstreamIssuingCAs.push(new crypto.Certificate(ca)),
          upstreamMapIssuingCA[md5] = true
        )
      ))()
    ),
    balancers = {
      'round-robin': algo.RoundRobinLoadBalancer,
      'least-work': algo.LeastWorkLoadBalancer,
      'hashing': algo.HashingLoadBalancer,
    },
    services = (
      Object.fromEntries(
        Object.entries(ingress.services).map(
          ([k, v]) =>(
            ((targets, balancer, balancerInst) => (
              targets = v?.upstream?.endpoints?.map?.(ep => `${ep.ip}:${ep.port}`),
              v?.upstream?.sslCert?.ca && (
                addUpstreamIssuingCA(v.upstream.sslCert.ca)
              ),
              balancer = balancers[v?.balancer || 'round-robin'] || balancers['round-robin'],
              balancerInst = new balancer(targets || []),

              [k, {
                balancer: balancerInst,
                cache: v?.sticky && new algo.Cache(
                  () => balancerInst.next()
                ),
                upstreamSSLName: v?.upstream?.sslName || null,
                upstreamSSLVerify: v?.upstream?.sslVerify || false,
                cert: v?.upstream?.sslCert?.cert || null,
                key: v?.upstream?.sslCert?.key || null,
                isTLS: Boolean(v?.upstream?.sslCert?.ca),
                isMTLS: Boolean(v?.upstream?.sslCert?.cert) && Boolean(v?.upstream?.sslCert?.key) && Boolean(v?.upstream?.sslCert?.ca),
                protocol: v?.upstream?.proto || 'HTTP'
              }]
            ))()
          )
        )
      )
    ),

  ) => pipy({
    _target: undefined,
    _service: null,
    _serviceSNI: null,
    _serviceVerify: null,
    _serviceCertChain: null,
    _servicePrivateKey: null,
    _connectTLS: false,
    _mTLS: false,
    _isOutboundGRPC: false,

    _serviceCache: null,
    _targetCache: null,

    _sourceIP: null,

    _g: {
      connectionID: 0,
    },

    _connectionPool: new algo.ResourcePool(
      () => ++_g.connectionID
    ),
    
    _select: (service, key) => (
      service?.cache && key ? (
        service?.cache?.get(key)
      ) : (
        service?.balancer?.next({})
      )
    ),
  })

  .import({
    __route: 'main',
    __isInboundHTTP2: 'proto',
    __isInboundGRPC: 'proto',
  })

  .pipeline()
    .handleStreamStart(
      () => (
        _serviceCache = new algo.Cache(
          // k is a balancer, v is a target
          (k) => _select(k, _sourceIP),
          (k, v) => k.balancer.deselect(v?.id),
        ),
        _targetCache = new algo.Cache(
          // k is a target, v is a connection ID
          (k) => _connectionPool.allocate(k),
          (k, v) => _connectionPool.free(v),
        )
      )
    )
    .handleStreamEnd(
      () => (
        _targetCache.clear(),
        _serviceCache.clear()
      )
    )
    .link('outbound-http')

  .pipeline('outbound-http')
    .handleMessageStart(
      (msg) => (
        _sourceIP = __inbound.remoteAddress,
        _service = services[__route],
        _service && (
          _serviceSNI = _service?.upstreamSSLName,
          _serviceVerify = _service?.upstreamSSLVerify,
          _serviceCertChain = _service?.cert,
          _servicePrivateKey = _service?.key,
          _target = _serviceCache.get(_service),
          _connectTLS = _service?.isTLS,
          _mTLS = _service?.isMTLS,
          _isOutboundGRPC = __isInboundGRPC || _service?.protocol === 'GRPC'
        ),

        console.log("[balancer] _sourceIP", _sourceIP),
        console.log("[balancer] _connectTLS", _connectTLS),
        console.log("[balancer] _mTLS", _mTLS),
        console.log("[balancer] _target.id", (_target || {id : ''}).id),
        console.log("[balancer] _isOutboundGRPC", _isOutboundGRPC)
      )
    )
    .branch(
      () => Boolean(_target) && !_connectTLS, (
        $=>$.muxHTTP(() => _targetCache.get(_target), { version: () => _isOutboundGRPC ? 2 : 1 }).to(
          $=>$.connect(() => _target.id)
        )
      ), () => Boolean(_target) && _connectTLS && !_isOutboundGRPC, (
        $=>$.muxHTTP(() => _targetCache.get(_target)).to(
          $=>$.connectTLS({
            certificate: () => (_mTLS ? {
              cert: new crypto.Certificate(_serviceCertChain),
              key: new crypto.PrivateKey(_servicePrivateKey),
            } : undefined),
            trusted: upstreamIssuingCAs,
            sni: () => _serviceSNI || undefined,
            verify: (ok, cert) => (
              !_serviceVerify && (ok = true),
              ok
            )
          }).to(
            $=>$.connect(() => _target.id)
          )
        )
      ), () => Boolean(_target) && _connectTLS && _isOutboundGRPC, (
        $=>$.muxHTTP(() => _targetCache.get(_target), { version: 2 }).to(
          $=>$.connectTLS({
            certificate: () => (_mTLS ? {
              cert: new crypto.Certificate(_serviceCertChain),
              key: new crypto.PrivateKey(_servicePrivateKey),
            } : undefined),
            trusted: upstreamIssuingCAs,
            sni: () => _serviceSNI || undefined,
            alpn: 'h2',
            verify: (ok, cert) => (
              !_serviceVerify && (ok = true),
                ok
            )
          }).to(
            $=>$.connect(() => _target.id)
          )
        )
      ), (
        $=>$.chain()
      )
    )
)()