#!/usr/bin/env node
// app-server-delay-proxy.mjs -- evidence-only latency proxy for the reference
// capture. It spawns the real dim-app-server on `<unix>.real` and listens on
// the `<unix>` path the Electron main process connects to, forwarding every
// NDJSON line unchanged except for the responses to the methods named in
// DIM_PROXY_DELAY_METHODS (default session.history,session.attach), which it
// delays by DIM_PROXY_DELAY_MS.
//
// It exists because a SIGSTOPped daemon never lets the renderer switch the
// active session, so the restore-time loading skeleton is not observable by
// pausing the process. The app, preload, renderer and daemon are unmodified;
// only response latency is injected. Used by capture-loading-state.sh.
import { spawn } from 'node:child_process';
import { unlinkSync } from 'node:fs';
import { connect, createServer } from 'node:net';

const args = process.argv.slice(2);
const unixIndex = args.indexOf('--unix');
if (unixIndex < 0 || args[unixIndex + 1] === undefined) {
  console.error('app-server-delay-proxy: --unix is required');
  process.exit(2);
}
const expected = args[unixIndex + 1];
const real = `${expected}.real`;
const realArgs = [...args];
realArgs[unixIndex + 1] = real;
const delayMs = Number(process.env.DIM_PROXY_DELAY_MS || '8000');
const methods = new Set((process.env.DIM_PROXY_DELAY_METHODS || 'session.history,session.attach').split(','));
const delayOnce = process.env.DIM_PROXY_DELAY_ONCE === '1';
let delayedCount = 0;
const serverBin = process.env.DIM_PROXY_REAL_SERVER;
if (serverBin === undefined || serverBin === '') {
  console.error('app-server-delay-proxy: DIM_PROXY_REAL_SERVER must name the real dim-app-server binary');
  process.exit(2);
}

try { unlinkSync(expected); } catch {}
try { unlinkSync(real); } catch {}
const child = spawn(serverBin, realArgs, { stdio: ['ignore', 'inherit', 'inherit'] });
child.on('error', (error) => {
  console.error(`app-server-delay-proxy: cannot spawn ${serverBin}: ${error.message}`);
  process.exit(1);
});
child.on('exit', (code) => process.exit(code ?? 0));
const stop = () => {
  child.kill('SIGTERM');
  setTimeout(() => process.exit(0), 300);
};
process.on('SIGTERM', stop);
process.on('SIGINT', stop);

const server = createServer((client) => {
  const upstream = connect(real);
  let clientBuffer = Buffer.alloc(0);
  let serverBuffer = Buffer.alloc(0);
  const delayedIds = new Set();
  client.on('data', (chunk) => {
    clientBuffer = Buffer.concat([clientBuffer, chunk]);
    for (let index = clientBuffer.indexOf(10); index >= 0; index = clientBuffer.indexOf(10)) {
      const line = clientBuffer.subarray(0, index + 1);
      clientBuffer = clientBuffer.subarray(index + 1);
      try {
        const message = JSON.parse(line.toString('utf8'));
        if (message && message.id !== undefined && methods.has(message.method) && (!delayOnce || delayedCount === 0)) {
          delayedIds.add(message.id);
          delayedCount += 1;
        }
      } catch {}
      upstream.write(line);
    }
  });
  upstream.on('data', (chunk) => {
    serverBuffer = Buffer.concat([serverBuffer, chunk]);
    for (let index = serverBuffer.indexOf(10); index >= 0; index = serverBuffer.indexOf(10)) {
      const line = Buffer.from(serverBuffer.subarray(0, index + 1));
      serverBuffer = serverBuffer.subarray(index + 1);
      let delayed = false;
      try {
        const message = JSON.parse(line.toString('utf8'));
        if (message && message.id !== undefined && delayedIds.has(message.id)) {
          delayedIds.delete(message.id);
          delayed = true;
        }
      } catch {}
      if (delayed) {
        setTimeout(() => { if (!client.destroyed) client.write(line); }, delayMs);
      } else {
        client.write(line);
      }
    }
  });
  const close = () => { client.destroy(); upstream.destroy(); };
  client.on('error', close);
  upstream.on('error', close);
  client.on('close', close);
  upstream.on('close', close);
});
server.on('error', (error) => {
  console.error(`app-server-delay-proxy: listen ${expected}: ${error.message}`);
  child.kill('SIGTERM');
  process.exit(1);
});
server.listen(expected);
