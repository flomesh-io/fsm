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

  makeServiceHandler = serviceName => (
    config?.Services?.[serviceName] ? (
      config.Services[serviceName].name = serviceName,
      config.Services[serviceName]
    ) : null
  ),

  serviceHandlers = new algo.Cache(makeServiceHandler),

) => pipy({
  _serviceName: null,
  _unauthorized: undefined,
})

.export('service', {
  __service: null,
})

.import({
  __route: 'route',
  __root: 'web-server',
  __consumer: 'consumer',
})

.pipeline()
.handleMessageStart(
  () => (
    __route?.config?.EnableHeadersAuthorization && (
      (!__consumer || !__consumer?.['Headers-Authorization']) ? (_unauthorized = true) : (_unauthorized = false)
    ),
    __route?.serverRoot ? (
      __root = __route.serverRoot
    ) : (
      (_serviceName = __route?.backendServiceBalancer?.borrow?.({})?.id) && (
        __service = serviceHandlers.get(_serviceName)
      )
    )
  )
)
.branch(
  isDebugEnabled, (
    $=>$.handleStreamStart(
      () => (
        console.log('[service] name, root, endpoints, unauthorized:', _serviceName, __root, Object.keys(__service?.Endpoints || {}), _unauthorized)
      )
    )
  )
)
.branch(
  () => _unauthorized, (
    $=>$.replaceMessage(
      () => new Message({status: 401})
    )
  ),
  () => __root, (
    $=>$.use('server/web-server.js')
  ),
  () => __service, (
    $=>$.chain()
  ), (
    $=>$.use('http/default.js')
  )
)

)()