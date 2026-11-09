// interact-error.mjs -- drive the running dim-electron renderer after its
// daemon has been killed. Usage: node interact-error.mjs <action>
// Actions:
//   type        focus the composer and type a probe sentence
//   click-send  click the composer send button (data-testid=chat-send)
//   probe       print the same DOM probe as probe-error.mjs
import { chromium } from '/Users/eric8810/Code/dim-electron/packages/desktop/node_modules/playwright-core/index.mjs';

const action = process.argv[2];
const browser = await chromium.connectOverCDP('http://127.0.0.1:9333');
const context = browser.contexts()[0];
const pages = context.pages();
const page = pages.find((p) => p.url().includes('/packages/renderer/dist/index.html') && !p.url().includes('surface=')) ?? pages[0];
if (!page) {
  console.error('no renderer page found');
  process.exit(1);
}

if (action === 'type') {
  const editor = page.locator('.ProseMirror').first();
  await editor.click({ timeout: 5000 });
  await page.keyboard.type('After daemon loss: does this send?', { delay: 20 });
  await page.waitForTimeout(800);
  console.log('typed probe sentence into composer');
} else if (action === 'click-sidebar') {
  const row = page.locator('[data-testid="sidebar-session-item"]').first();
  await row.click({ timeout: 5000 });
  await page.mouse.move(600, 400);
  await page.waitForTimeout(2500);
  console.log('clicked sidebar session row after daemon loss');
} else if (action === 'new-session') {
  await page.getByText('新建会话', { exact: true }).first().click({ timeout: 5000 });
  await page.mouse.move(600, 400);
  await page.waitForTimeout(3000);
  console.log('clicked 新建会话 after daemon loss');
} else if (action === 'click-send') {
  const button = page.locator('[data-testid="chat-send"]').first();
  console.log('send button disabled:', await button.isDisabled());
  await button.click({ timeout: 5000, force: true });
  await page.waitForTimeout(2500);
  console.log('clicked send button');
} else {
  console.error('unknown action', action);
  process.exit(2);
}
await browser.close();
