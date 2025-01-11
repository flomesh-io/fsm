export default function (config) {
  var abortMessage = new Message(
    {
      status: config.requestTermination.response?.status || 503,
      headers: config.requestTermination.response?.headers,
    },
    config.requestTermination.response?.body || 'Service unavailable'
  )

  return pipeline($=>$
    .replaceData()
    .replaceMessage(abortMessage)
  )
}
