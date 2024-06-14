export var log

var logFunc = function (a, b, c, d, e, f) {
  var n = 6
  if (f === undefined) n--
  if (e === undefined) n--
  if (d === undefined) n--
  if (c === undefined) n--
  if (b === undefined) n--
  if (a === undefined) n--
  switch (n) {
    case 0: console.log(); break
    case 1: console.log(a); break
    case 2: console.log(a, b); break
    case 3: console.log(a, b, c); break
    case 4: console.log(a, b, c, d); break
    case 5: console.log(a, b, c, d, e); break
    case 6: console.log(a, b, c, d, e, f); break
  }
}

export function logEnable(on) {
  log = on ? logFunc : null
}
