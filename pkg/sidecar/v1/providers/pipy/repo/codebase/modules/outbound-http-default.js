pipy()

.import({
  __route: 'outbound-http-routing',
})

.pipeline()
.replaceData()
.branch(
  () => !__route, (
    $=>$.replaceMessage(
      new Message({
          status: 403
        },
        'Access denied'
      )
    )
  ),

  (
    $=>$.replaceMessage(
      new Message({
          status: 404
        }, 'Not found'
      )
    )
  )
)