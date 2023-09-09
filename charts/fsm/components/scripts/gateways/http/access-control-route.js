((
  { isDebugEnabled } = pipy.solve('config.js'),

  { aclCounter } = pipy.solve('lib/metrics.js'),
  aclRouteCounter = aclCounter.withLabels('route'),

  parseIpList = ipList => (
    (ips, ipRanges) => (
      (ipList || []).forEach(
        o => (
          o.indexOf('/') > 0 ? (
            !ipRanges && (ipRanges = []),
            ipRanges.push(new Netmask(o))
          ) : (
            !ips && (ips = {}),
            ips[o] = true
          )
        )
      ),
      (ips || ipRanges) ? ({ ips, ipRanges }) : undefined
    )
  )(),

  aclsCache = new algo.Cache(
    acls => (
      {
        blackList: parseIpList(acls?.blacklist),
        whiteList: parseIpList(acls?.whitelist),
      }
    )
  ),

  checkACLs = ip => (
    (
      acls = aclsCache.get(__route?.config?.AccessControlLists),
      white = acls?.whiteList,
      black = acls?.blackList,
      blackMode = true,
      block = false,
      pass = false,
    ) => (
      white && (
        blackMode = false,
        (white?.ips?.[ip] && (pass = true)) || (
          pass = white?.ipRanges?.find?.(r => r.contains(ip))
        )
      ),
      blackMode && (
        (black?.ips?.[ip] && (block = true)) || (
          block = black?.ipRanges?.find?.(r => r.contains(ip))
        )
      ),
      blackMode ? Boolean(!block) : Boolean(pass)
    )
  )(),
) => pipy({
  _ips: null,
  _pass: false,
})

.import({
  __route: 'route',
})

.pipeline()
.handleMessageStart(
  msg => (
    __route?.config?.AccessControlLists?.enableXFF && (
      _ips = msg.head?.headers['x-forwarded-for']
    ),
    _ips ? (
      _pass = _ips.split(',').every(ip => checkACLs(ip.trim()))
    ) : (
      _pass = checkACLs(__inbound.remoteAddress)
    )
  )
)
.branch(
  () => _pass, (
    $=>$.chain()
  ), (
    $=>$
    .branch(
      isDebugEnabled, (
        $=>$.handleStreamStart(
          () => (
            console.log('[access-control-route] blocked XFF, IP address:', _ips, __inbound.remoteAddress, _pass)
          )
        )
      )
    )
    .replaceMessage(
      () => (
        aclRouteCounter.increase(),
        new Message({ status: __route?.config?.AccessControlLists?.status || 403 }, __route?.config?.AccessControlLists?.message || '')
      )
    )
  )
)

)()