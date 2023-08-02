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
  cacheTTL = val => (
    val?.indexOf('s') > 0 && (
      val.replace('s', '')
    ) ||
    val?.indexOf('m') > 0 && (
      val.replace('m', '') * 60
    ) ||
    val?.indexOf('h') > 0 && (
      val.replace('h', '') * 3600
    ) ||
    val?.indexOf('d') > 0 && (
      val.replace('d', '') * 86400
    ) ||
    0
  ),
  originMatch = origin => (
    (origin || []).map(
      o => (
        o?.exact && (
          url => url === o.exact
        ) ||
        o?.prefix && (
          url => url.startsWith(o.prefix)
        ) ||
        o?.regex && (
          (match = new RegExp(o.regex)) => (
            url => match.test(url)
          )
        )()
      )
    )
  ),
  configCache = new algo.Cache(
    pluginConfig => (
      (originHeaders = {}, optionsHeaders = {}) => (
        pluginConfig?.allowCredentials && (
          originHeaders['access-control-allow-credentials'] = 'true'
        ),
        pluginConfig?.exposeHeaders && (
          originHeaders['access-control-expose-headers'] = pluginConfig.exposeHeaders.join()
        ),
        pluginConfig?.allowMethods && (
          optionsHeaders['access-control-allow-methods'] = pluginConfig.allowMethods.join()
        ),
        pluginConfig?.allowHeaders && (
          optionsHeaders['access-control-allow-headers'] = pluginConfig.allowHeaders.join()
        ),
        pluginConfig?.maxAge && (cacheTTL(pluginConfig?.maxAge) > 0) && (
          optionsHeaders['access-control-max-age'] = cacheTTL(pluginConfig?.maxAge)
        ),
        {
          originHeaders,
          optionsHeaders,
          matchingMap: originMatch(pluginConfig?.allowOrigins)
        }
      )
    )()
  ),
) => pipy({
  _pluginName: '',
  _pluginConfig: null,
  _corsHeaders: null,
  _matchingMap: null,
  _matching: false,
  _isOptions: false,
  _origin: undefined,
})
.import({
  __service: 'service',
})
.pipeline()
.onStart(
  () => void (
    _pluginName = __filename.slice(9, -3),
    _pluginConfig = __service?.Plugins?.[_pluginName],
    _corsHeaders = configCache.get(_pluginConfig),
    _matchingMap = _corsHeaders?.matchingMap
  )
)
.branch(
  () => _matchingMap, (
    $=>$
    .handleMessageStart(
      msg => (
        (_origin = msg?.head?.headers?.origin) && (_matching = _matchingMap.find(o => o(_origin))) && (
          _isOptions = (msg?.head?.method === 'OPTIONS')
        )
      )
    )
  ), (
    $=>$
  )
)
.branch(
  () => _matching, (
    $=>$
    .branch(
      () => _isOptions, (
        $=>$
        .replaceMessage(
          () => (
            new Message({ status: 200, headers: { ..._corsHeaders.originHeaders, ..._corsHeaders.optionsHeaders, 'access-control-allow-origin': _origin } })
          )
        )
      ), (
        $=>$
        .chain()
        .handleMessageStart(
          msg => (
            Object.keys(_corsHeaders.originHeaders).forEach(
              key => msg.head.headers[key] = _corsHeaders.originHeaders[key]
            ),
            msg.head.headers['access-control-allow-origin'] = _origin
          )
        )
      )
    )
  ), (
    $=>$.chain()
  )
)
)()