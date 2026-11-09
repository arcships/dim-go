// watch-console.mjs -- attach to the running dim-electron renderer over CDP and
// print every console message / page error / log entry for N seconds.
// Usage: node watch-console.mjs <seconds>
// It only listens; it never evaluates or mutates the page.
import { chromium } from '/Users/eric8810/Code/dim-electron/packages/desktop/node_modules/playwright-core/index.mjs';

const seconds = Number(process.argv[2] ?? 15);
const browser = await chromium.connectOverCDP('http://127.0.0.1:9333');
const context = browser.contexts()[0];
const pages = context.pages();
const page = pages.find((p) => p.url().includes('/packages/renderer/dist/index.html') && !p.url().includes('surface=')) ?? pages[0];
if (!page) {
  console.error('no renderer page found');
  process.exit(1);
}

const stamp = () => new Date().toISOString();
page.on('console', (msg) => {
  console.log(`[${stamp()}] console.${msg.type()}: ${msg.text()}`);
});
page.on('pageerror', (error) => {
  console.log(`[${stamp()}] pageerror: ${error.message}`);
});
page.on('requestfailed', (request) => {
  console.log(`[${stamp()}] requestfailed: ${request.url()} :: ${request.failure()?.errorText ?? ''}`);
});

console.log(`[${stamp()}] watching renderer console for ${seconds}s`);
await new Promise((resolve) => setTimeout(resolve, seconds * 1000));
console.log(`[${stamp()}] watch finished`);
await browser.close();
