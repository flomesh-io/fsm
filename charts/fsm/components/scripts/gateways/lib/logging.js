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

(
  (
    { isDebugEnabled } = pipy.solve('config.js'),

    {
      namespace,
      kind,
      name,
      pod,
      toInt63,
    } = pipy.solve('lib/utils.js'),

    address = os.env.REMOTE_LOGGING_ADDRESS,
    tracingLimitedID = os.env.REMOTE_LOGGING_SAMPLED_FRACTION && (os.env.REMOTE_LOGGING_SAMPLED_FRACTION * Math.pow(2, 63)),
    logLogging = address && new logging.JSONLogger('access-logger').toHTTP('http://' + address +
      (os.env.REMOTE_LOGGING_ENDPOINT || '/?query=insert%20into%20log(message)%20format%20JSONAsString'), {
      batch: {
        timeout: 1,
        interval: 1,
        prefix: '[',
        postfix: ']',
        separator: ','
      },
      headers: {
        'Content-Type': 'application/json',
        'Authorization': os.env.REMOTE_LOGGING_AUTHORIZATION || ''
      }
    }).log,

    initTracingHeaders = (headers) => (
      (
        uuid = algo.uuid(),
        id = uuid.substring(0, 18).replaceAll('-', ''),
      ) => (
        headers['x-forwarded-proto'] = 'http',
        headers['x-b3-spanid'] && (
          (headers['x-b3-parentspanid'] = headers['x-b3-spanid']) && (headers['x-b3-spanid'] = id)
        ),
        !headers['x-b3-traceid'] && (
          (headers['x-b3-traceid'] = id) && (headers['x-b3-spanid'] = id)
        ),
        !headers['x-request-id'] && (
          headers['x-request-id'] = uuid
        ),
        headers['fgw-stats-namespace'] = namespace,
        headers['fgw-stats-kind'] = kind,
        headers['fgw-stats-name'] = name,
        headers['fgw-stats-pod'] = pod
      )
    )(),
  ) => (
    {
      loggingEnabled: Boolean(logLogging),

      makeLoggingData: (msg, remoteAddr, remotePort, localAddr, localPort) => (
        (
          sampled = false,
        ) => (
          msg?.head?.headers && (
            !msg.head.headers['x-b3-traceid'] && (
              initTracingHeaders(msg.head.headers)
            ),
            sampled = (!tracingLimitedID || toInt63(msg.head.headers['x-b3-traceid']) < tracingLimitedID)
          ),
          sampled ? (
            {
              reqTime: Date.now(),
              meshName: os.env.MESH_NAME || '',
              remoteAddr,
              remotePort,
              localAddr,
              localPort,
              node: {
                ip: os.env.POD_IP || '127.0.0.1',
                name: os.env.HOSTNAME || 'localhost',
              },
              pod: {
                ns: os.env.POD_NAMESPACE || 'default',
                ip: os.env.POD_IP || '127.0.0.1',
                name: os.env.POD_NAME || os.env.HOSTNAME || 'localhost',
              },
              trace: {
                id: msg.head.headers?.['x-b3-traceid'] || '',
                span: msg.head.headers?.['x-b3-spanid'] || '',
                parent: msg.head.headers?.['x-b3-parentspanid'] || '',
                sampled: '1',
              },
              req: Object.assign({ reqSize: msg.body.size, body: msg.body.toString('base64') }, msg.head)
            }
          ) : null
        )
      )(),

      saveLoggingData: (loggingData, msg, service, target) => (
        loggingData.service = {
          name: service || 'anonymous', target: target
        },
        loggingData.res = Object.assign({}, msg.head),
        loggingData.res['resSize'] = msg.body.size,
        loggingData.res['body'] = msg.body.toString('base64'),
        loggingData['resTime'] = Date.now(),
        loggingData['endTime'] = Date.now(),
        loggingData['type'] = 'fgw',
        logLogging(loggingData),
        isDebugEnabled && console.log('[logging] json:', loggingData)
      ),
    }
  )

)()