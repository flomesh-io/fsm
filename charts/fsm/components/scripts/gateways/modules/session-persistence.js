var $sessionKey

export default function (sessionPersistence) {
  var sessionCache = new algo.Cache
  var sessionReqKeyGetter
  var sessionResKeyGetter

  switch (sessionPersistence.type || 'Cookie') {
    case 'Cookie':
      sessionReqKeyGetter = makeReqCookieSessionKeyGetter()
      sessionResKeyGetter = makeResCookieSessionKeyGetter()
      break
    case 'Header':
      sessionReqKeyGetter = sessionResKeyGetter = makeHeaderSessionKeyGetter()
      break
  }

  function restore(head) {
    $sessionKey = sessionReqKeyGetter(head)
    return $sessionKey && sessionCache.get($sessionKey)
  }

  function preserve(head, session) {
    var k = sessionResKeyGetter(head)
    if (k) $sessionKey = k
    sessionCache.set($sessionKey, session)
  }

  function makeReqCookieSessionKeyGetter() {
    var cookiePrefix = sessionPersistence.sessionName + '='
    return (head) => {
      var cookies = head.headers.cookie
      if (cookies) {
        var cookie = cookies.split(';').find(c => c.trim().startsWith(cookiePrefix))
        if (cookie) return cookie.substring(cookiePrefix.length).trim()
      }
    }
  }

  function makeResCookieSessionKeyGetter() {
    var cookiePrefix = sessionPersistence.sessionName + '='
    return (head) => {
      var v
      var values = head.headers['set-cookie']
      if (values instanceof Array) {
        v = values.find(v => v.startsWith(cookiePrefix))
      } else {
        v = values || ''
      }
      if (v) {
        var i = v.indexOf(';')
        if (i >= 0) v = v.substring(0, i)
        return v.substring(cookiePrefix.length)
      }
    }
  }

  function makeHeaderSessionKeyGetter() {
    var headerName = sessionPresistance.sessionName
    return (head) => head.headers[headerName]
  }

  return { restore, preserve }
}
