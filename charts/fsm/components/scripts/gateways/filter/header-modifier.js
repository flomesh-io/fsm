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
  resolvVar = val => (
    val?.startsWith('$') ? (
      (
        pos = val.indexOf('_'),
        name,
        member,
        content = val,
      ) => (
        (pos > 0) && (
          name = val.substring(1, pos),
          member = val.substring(pos + 1),
          (name === 'http') && (
            content = __http?.headers?.[member] || __http?.[member] || val
          ) || (name === 'consumer') && (
            content = __consumer?.[member] || val
          )
        ),
        content
      )
    )() : val
  ),

  makeModifierHandler = cfg => (
    (
      set = cfg?.set,
      add = cfg?.add,
      remove = cfg?.remove,
    ) => (
      (set || add || remove) && (
        msg => (
          set && set.forEach(
            e => (msg[e.name] = resolvVar(e.value))
          ),
          add && add.forEach(
            e => (
              msg[e.name] ? (
                msg[e.name] = msg[e.name] + ',' + resolvVar(e.value)
              ) : (
                msg[e.name] = resolvVar(e.value)
              )
            )
          ),
          remove && remove.forEach(
            e => delete msg[e]
          )
        )
      )
    )
  )(),

  modifierHandlers = new algo.Cache(makeModifierHandler),

  makeRequestModifierHandler = cfg => (
    (
      handlers = (cfg?.Filters || []).filter(
        e => e?.Type === 'RequestHeaderModifier'
      ).map(
        e => modifierHandlers.get(e)
      ).filter(
        e => e
      )
    ) => (
      handlers.length > 0 ? handlers : null
    )
  )(),

  requestModifierHandlers = new algo.Cache(makeRequestModifierHandler),

  makeResponseModifierHandler = cfg => (
    (
      handlers = (cfg?.Filters || []).filter(
        e => e?.Type === 'ResponseHeaderModifier'
      ).map(
        e => modifierHandlers.get(e)
      ).filter(
        e => e
      )
    ) => (
      handlers.length > 0 ? handlers : null
    )
  )(),

  responseModifierHandlers = new algo.Cache(makeResponseModifierHandler),

) => pipy({
  _requestHandlers: null,
  _responseHandlers: null,
})

.import({
  __service: 'service',
  __http: 'http',
  __consumer: 'consumer',
})

.pipeline()
.onStart(
  () => void (
    _requestHandlers = requestModifierHandlers.get(__service),
    _responseHandlers = responseModifierHandlers.get(__service)
  )
)
.branch(
  () => _requestHandlers, (
    $=>$.handleMessageStart(
      msg => (
        msg?.head?.headers && _requestHandlers.forEach(
          e => e(msg.head.headers)
        )
      )
    )
  ), (
    $=>$
  )
)
.chain()
.branch(
  () => _responseHandlers, (
    $=>$.handleMessageStart(
      msg => (
        msg?.head?.headers && _responseHandlers.forEach(
          e => e(msg.head.headers)
        )
      )
    )
  ), (
    $=>$
  )
)

)()