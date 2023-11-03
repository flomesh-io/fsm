((
  { isDebugEnabled } = pipy.solve('config.js'),
) => (

pipy()

.export('consumer', {
  __consumer: null,
})

.pipeline()
.branch(
  isDebugEnabled, (
    $=>$.handleStreamStart(
      () => __consumer && (
        console.log('[consumer]:', __consumer)
      )
    )
  )
)
.chain()

))()