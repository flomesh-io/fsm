pipy()

.import({
  __port: 'listener',
})

.pipeline()
.link('ratelimit')
.chain()
.link('ratelimit')

.pipeline('ratelimit')
.branch(
  () => __port?.bpsLimit > 0, (
    $=>$.throttleDataRate(
      () => (
        new algo.Quota(
          __port.bpsLimit,
          {
            produce: __port.bpsLimit,
            per: '1s',
          }
        )
      )
    )
  ), (
    $ => $
  )
)