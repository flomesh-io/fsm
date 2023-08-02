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
  { config, isDebugEnabled } = pipy.solve('config.js'),

  makeMatchDomainHandler = portRouteRules => (
    (
      domains = {},
      starDomains = [],
      lowercaseDomain,
    ) => (
      Object.keys(portRouteRules || {}).forEach(
        domain => (
          portRouteRules[domain].name = domain,
          lowercaseDomain = domain.toLowerCase(),
          domain.startsWith('*') ? (
            portRouteRules[domain].starName = lowercaseDomain.substring(1),
            starDomains.push(portRouteRules[domain])
          ) : (
            domains[lowercaseDomain] = portRouteRules[domain]
          )
        )
      ),
      domain => (
        domains[domain] ? (
          domains[domain]
        ) : (
          starDomains.find(
            d => (
              domain.endsWith(d.starName) ? d : null
            )
          )
        )
      )
    )
  )(),

  matchDomainHandlers = new algo.Cache(makeMatchDomainHandler),

  getParameters = path => (
    (
      params = {},
      qsa,
      qs,
      arr,
      kv,
    ) => (
      path && (
        (qsa = path.split('?')[1]) && (
          (qs = qsa.split('#')[0]) && (
            (arr = qs.split('&')) && (
              arr.forEach(
                p => (
                  kv = p.split('='),
                  params[kv[0]] = kv[1]
                )
              )
            )
          )
        )
      ),
      params
    )
  )(),

  makeDictionaryMatches = dictionary => (
    (
      tests = Object.entries(dictionary || {}).map(
        ([type, dict]) => (
          (type === 'Exact') ? (
            Object.keys(dict || {}).map(
              k => (obj => obj?.[k] === dict[k])
            )
          ) : (
            (type === 'Regex') ? (
              Object.keys(dict || {}).map(
                k => (
                  (
                    regex = new RegExp(dict[k])
                  ) => (
                    obj => regex.test(obj?.[k] || '')
                  )
                )()
              )
            ) : [() => false]
          )
        )
      )
    ) => (
      (tests.length > 0) && (
        obj => tests.every(a => a.every(f => f(obj)))
      )
    )
  )(),

  pathPrefix = (path, prefix) => (
    path.startsWith(prefix) && (
      prefix.endsWith('/') || (
        (
          lastChar = path.charAt(prefix.length),
        ) => (
          lastChar === '' || lastChar === '/'
        )
      )()
    )
  ),

  makeHttpMatches = rule => (
    (
      matchPath = (
        (rule?.Path?.Type === 'Regex') && (
          ((match = null) => (
            match = new RegExp(rule?.Path?.Path),
            (path) => match.test(path)
          ))()
        ) || (rule?.Path?.Type === 'Exact') && (
          (path) => path === rule?.Path?.Path
        ) || (rule?.Path?.Type === 'Prefix') && (
          (path) => pathPrefix(path, rule?.Path?.Path)
        ) || rule?.Path?.Type && (
          () => false
        )
      ),
      matchHeaders = makeDictionaryMatches(rule?.Headers),
      matchMethod = (
        rule?.Methods && Object.fromEntries((rule.Methods).map(m => [m, true]))
      ),
      matchParams = makeDictionaryMatches(rule?.QueryParams),
    ) => (
      {
        config: rule,
        match: message => (
          (!matchMethod || matchMethod[message?.head?.method]) && (
            (!matchPath || matchPath(message?.head?.path?.split('?')[0])) && (
              (!matchHeaders || matchHeaders(message?.head?.headers)) && (
                (!matchParams || matchParams(getParameters(message?.head?.path)))
              )
            )
          )
        ),
        backendServiceBalancer: new algo.RoundRobinLoadBalancer(rule?.BackendService || {}),
        ...(rule?.ServerRoot && { serverRoot: rule.ServerRoot })
      }
    )
  )(),

  makeGrpcMatches = rule => (
    (
      matchHeaders = makeDictionaryMatches(rule?.Headers),
      matchMethod = (
        rule?.Method?.Type === 'Exact' && (
          path => (
            (
              grpc = (path || '').split('/'),
            ) => (
              (path === '/grpc.reflection.v1alpha.ServerReflection/ServerReflectionInfo') || (
                (rule?.Method?.Service === grpc[1]) && (rule?.Method?.Method === grpc[2])
              )
            )
          )()
        )
      ),
    ) => (
      {
        config: rule,
        match: message => (
          (!matchHeaders || matchHeaders(message?.head?.headers)) && (
            (!matchMethod || matchMethod(message?.head?.path))
          )
        ),
        backendServiceBalancer: new algo.RoundRobinLoadBalancer(rule?.BackendService || {})
      }
    )
  )(),

  makeRouteMatchesHandler = routeTypeMatches => (
    (
      matches = [],
    ) => (
      (!routeTypeMatches?.RouteType || routeTypeMatches?.RouteType === 'HTTP' || routeTypeMatches?.RouteType === 'HTTP2') && (
        matches = (routeTypeMatches?.Matches || []).map(
          m => makeHttpMatches(m)
        )
      ),
      (routeTypeMatches?.RouteType === 'GRPC') && (
        matches = (routeTypeMatches?.Matches || []).map(
          m => makeGrpcMatches(m)
        )
      ),
      message => (
        matches.find(
          m => m.match(message)
        )
      )
    )
  )(),

  routeMatchesHandlers = new algo.Cache(makeRouteMatchesHandler),

  hostHandlers = new algo.Cache(
    fullHost => (
      (
        host = fullHost.split('@')[0],
        routeRules = config?.RouteRules?.[__port?.Port],
        matchDomain = matchDomainHandlers.get(routeRules),
        domain = matchDomain(host.toLowerCase()),
        messageHandler = routeMatchesHandlers.get(domain),
      ) => (
        message => (
          _host = host,
          __domain = domain,
          messageHandler && (
            __route = messageHandler(message)
          )
        )
      )
    )()
  ),

  handleMessage = (host, msg) => (
    host && (
      (
        fullHost = host + '@' + __port?.Port,
        handler = hostHandlers.get(fullHost),
      ) => (
        handler && handler(msg)
      )
    )()
  ),

) => pipy({
  _host: undefined,
})

.export('route', {
  __domain: null,
  __route: null,
})

.import({
  __port: 'listener',
  __consumer: 'consumer',
})

.pipeline()
.handleMessageStart(
  msg => (
    handleMessage(msg?.head?.headers?.host, msg),
    !__domain && __consumer?.sni && (
      handleMessage(__consumer.sni, msg)
    ),
    !__domain && config?.Configs?.StripAnyHostPort && (
      handleMessage(msg?.head?.headers?.host?.split(':')?.[0], msg)
    )
  )
)
.branch(
  isDebugEnabled, (
    $=>$.handleStreamStart(
      () => (
        console.log('[route] port, host, __domain.name, __route.path:', __port?.Port, _host, __domain?.name, __route?.config?.Path?.Path || __route?.config?.Method)
      )
    )
  )
)
.chain()

)()