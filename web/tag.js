async function setTag(tag, value) {
  const response = await fetch('/.tagSet', {
    method: 'POST',
    cache: 'no-cache',
    headers: {
      'Content-Type': 'text/plain'
    },
    body: tag + " = " + value
  });
  return response;
}

function n2b(n, s) {
  const r = [];
  for (let i = 0; i < s; i++) {
    if (i * 8 >= 32) {
      r.push(Number(Math.floor(n / Math.pow(2, i * 8))));
    } else {
      r.push((n >>> (i * 8)) & 0xFF);
    }
  }
  return r;
}

function to_float32(x) {
  let buf = new ArrayBuffer(4);
  let num = new Float32Array(buf);
  num[0] = parseFloat(x);
  x = new Uint8Array(buf);
  return [x[0], x[1], x[2], x[3]];
}

function to_float64(x) {
  let buf = new ArrayBuffer(8);
  let num = new Float64Array(buf);
  num[0] = parseFloat(x);
  x = new Uint8Array(buf);
  return [x[0], x[1], x[2], x[3], x[4], x[5], x[6], x[7]];
}

function r2b(n, s) {
  return s == 4 ? to_float32(n) : to_float64(n);
}

function clicBOOL(ev) {
  const tc = ev.target.textContent === "1" ? "0" : "1";
  ev.target.textContent = tc;
  setTag(ev.target.attributes[2].textContent, tc);
}

function clicINT(ev) {
  let tc = ev.target.textContent;
  tc = prompt("Podaj liczbę", tc);
  if (tc !== null) {
    const size = parseInt(ev.target.attributes[3].textContent, 10);
    ev.target.textContent = tc;
    tc = n2b(tc, size);
    setTag(ev.target.attributes[2].textContent, tc);
  }
}

function clicREAL(ev) {
  let tc = ev.target.textContent;
  tc = prompt("Podaj liczbę", tc);
  if (tc !== null) {
    const size = parseInt(ev.target.attributes[3].textContent, 10);
    ev.target.textContent = tc;
    tc = r2b(tc, size);
    setTag(ev.target.attributes[2].textContent, tc);
  }
}
