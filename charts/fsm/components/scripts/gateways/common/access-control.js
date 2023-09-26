((
  { isDebugEnabled } = pipy.solve('config.js'),

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
        blackList: parseIpList(acls?.Blacklist),
        whiteList: parseIpList(acls?.Whitelist),
      }
    )
  ),

  checkACLs = ip => (
    (
      acls = aclsCache.get(__port?.AccessControlLists),
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
) => pipy()

.import({
  __port: 'listener',
})

.pipeline()
.branch(
  () => checkACLs(__inbound.remoteAddress), (
    $=>$.chain()
  ), (
    $=>$
    .branch(
      isDebugEnabled, (
        $=>$.handleStreamStart(
          () => (
            console.log('Blocked IP address:', __inbound.remoteAddress)
          )
        )
      )
    )
    .replaceStreamStart(
      () => (
        new StreamEnd('ConnectionReset')
      )
    )
  )
)

)()
