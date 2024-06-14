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
  () => __port?.BpsLimit > 0, (
    $=>$.throttleDataRate(
      () => (
        new algo.Quota(
          __port.BpsLimit,
          {
            produce: __port.BpsLimit,
            per: '1s',
          }
        )
      )
    )
  ), (
    $ => $
  )
)
