export default function () {
  var $ctx
  var $sni
  var $session

  var balancers = new algo.Cache(
    target => new algo.LoadBalancer([target])
  )

  return pipeline($=>$
    .onStart(c => {
      $ctx = c.parent
      $sni = $ctx.originalServerName
      $session = balancers.get($ctx.originalTarget).allocate()
    })
    .muxHTTP(() => $session).to($=>$
      .pipe(() => $sni ? 'tls' : 'tcp', {
        'tcp': ($=>$.connect(() => $session.target)),
        'tls': ($=>$
          .connectTLS({ sni: () => $sni }).to($=>$
            .connect(() => $session.target)
          )
        ),
      })
    )
    .onEnd(() => $session.free())
  )
}
