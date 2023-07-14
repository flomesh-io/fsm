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
  getPrefix = uri => (
    (
      path = uri?.split?.('?')[0] || '',
      elts = path.split('/'),
    ) => (
      (elts[0] === '' && elts.length > 1) ? ('/' + (elts[1] || '')) : elts[0]
    )
  )(),

  makeHeadHandler = cfg => (
    (cfg?.path?.type === 'ReplacePrefixMatch') ? (
      head => (
        (
          prefix = getPrefix(head?.path),
          suffix = (head?.path || '').substring(prefix.length),
        ) => (
          cfg?.path?.replacePrefixMatch === '/' ? (
            head.path = suffix
          ) : (
            head.path = cfg?.path?.replacePrefixMatch + suffix
          ),
          cfg?.hostname && head.headers && (
            head.headers.host = cfg.hostname
          )
        )
      )()
    ) : (
      (cfg?.path?.type === 'ReplaceFullPath') ? (
        head => (
          (
            prefix = (head?.path || '').split('?')[0],
            suffix = (head?.path || '').substring(prefix.length),
          ) => (
            head.path  = cfg?.path?.replaceFullPath + suffix,
            cfg?.hostname && head.headers && (
              head.headers.host = cfg.hostname
            )
          )
        )()
      ) : null
    )
  ),

  headHandlers = new algo.Cache(makeHeadHandler),

  makeRewriteHandler = cfg => (
    (
      handlers = (cfg?.Filters || []).filter(
        e => e?.Type === 'HTTPURLRewriteFilter'
      ).map(
        e => headHandlers.get(e)
      ).filter(
        e => e
      )
    ) => (
      handlers.length > 0 ? handlers : null
    )
  )(),

  rewriteHandlersCache = new algo.Cache(makeRewriteHandler),

) => pipy({
  _rewriteHandlers: null,
})

.import({
  __service: 'service',
})

.pipeline()
.onStart(
  () => void (
    _rewriteHandlers = rewriteHandlersCache.get(__service)
  )
)
.branch(
  () => _rewriteHandlers, (
    $=>$.handleMessageStart(
      msg => (
        msg?.head?.headers && _rewriteHandlers.forEach(
          e => e(msg.head)
        )
      )
    )
  ), (
    $=>$
  )
)
.chain()

)()