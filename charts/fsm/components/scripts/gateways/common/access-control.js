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
        blackList: parseIpList(acls?.blacklist),
        whiteList: parseIpList(acls?.whitelist),
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