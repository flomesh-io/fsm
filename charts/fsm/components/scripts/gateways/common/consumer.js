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
      () => (
        console.log('[consumer]:', __consumer)
      )
    )
  )
)
.chain()

))()