import { log } from '../../utils.js'

export default function (config) {
  var hostnames = {}
  var prefixes = []

  config.dnsModifier.domains?.forEach?.(
    ent => {
      var name = ent.name
      var answer = ent.answer
      if (name.startsWith('*')) {
        prefixes.push([name.substring(1), answer])
      } else {
        hostnames[name] = answer
      }
    }
  )

  function findAnswer(name) {
    var a = hostnames[name]
    if (a) return a
    a = prefixes.find(([prefix]) => name.endsWith(prefix))
    if (a) return a[1]
  }

  var $name
  var $question
  var $answer

  return pipeline($=>$
    .replaceData(data => {
      if (data.size > 0) {
        return new Message(data)
      }
    })
    .demux().to($=>$
      .replaceMessage(
        msg => {
          $question = DNS.decode(msg.body)
          $name = $question?.question?.[0]?.name
          if ($name) {
            $answer = findAnswer($name)
          } else {
            $answer = null
          }
          log?.('[DNSModifier] Q =', $question, '; A =', $answer || '?')
          return msg.body
        }
      )
      .pipe(() => {
        return $answer ? 'reply' : 'forward'
      }, {
        'reply': ($=>$
          .replaceData(
            () => new Message(DNS.encode({
              id: $question.id,
              qr: 1,
              rd: 1,
              ra: 1,
              question: $question.question,
              answer: [{
                name: $name,
                type: 'A',
                ttl: 3600,
                ...$answer,
              }]
            }))
          )
        ),
        'forward': ($=>$
          .pipeNext()
          .replaceData(data => new Message(data))
        ),
      })
    )
    .replaceMessage(msg => msg.body)
  )
}
