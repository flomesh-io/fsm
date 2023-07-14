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
  hexChar = { '0': 0, '1': 1, '2': 2, '3': 3, '4': 4, '5': 5, '6': 6, '7': 7, '8': 8, '9': 9, 'a': 10, 'b': 11, 'c': 12, 'd': 13, 'e': 14, 'f': 15 },
  toInt63 = str => (
    (
      value = str.split('').reduce((calc, char) => (calc * 16) + hexChar[char], 0),
    ) => value / 2
  )(),
  traceId = () => algo.uuid().substring(0, 18).replaceAll('-', ''),

) => (
  {
    namespace: (os.env.POD_NAMESPACE || 'default'),
    kind: (os.env.POD_CONTROLLER_KIND || 'Deployment'),
    name: (os.env.SERVICE_ACCOUNT || 'fgw'),
    pod: (os.env.POD_NAME || ''),

    toInt63,
    traceId,

    initRateLimit: rateLimit => (
      rateLimit?.Local ? (
        {
          count: 0,
          backlog: rateLimit.Local.Backlog || 0,
          quota: new algo.Quota(
            rateLimit.Local.Burst || rateLimit.Local.Requests || 0,
            {
              produce: rateLimit.Local.Requests || 0,
              per: rateLimit.Local.StatTimeWindow || 0,
            }
          ),
          response: new Message({
            status: rateLimit.Local.ResponseStatusCode || 429,
            headers: Object.fromEntries((rateLimit.Local.ResponseHeadersToAdd || []).map(({ Name, Value }) => [Name, Value])),
          }),
        }
      ) : null
    ),

    shuffle: arg => (
      (
        sort = a => (a.map(e => e).map(() => a.splice(Math.random() * a.length | 0, 1)[0])),
      ) => (
        arg ? Object.fromEntries(sort(sort(Object.entries(arg)))) : {}
      )
    )(),

    failover: json => (
      json ? ((obj = null) => (
        obj = Object.fromEntries(
          Object.entries(json).map(
            ([k, v]) => (
              (v === 0) ? ([k, 1]) : null
            )
          ).filter(e => e)
        ),
        Object.keys(obj).length === 0 ? null : new algo.RoundRobinLoadBalancer(obj)
      ))() : null
    ),

  }
))()