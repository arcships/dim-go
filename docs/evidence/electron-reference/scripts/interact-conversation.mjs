// interact-conversation.mjs -- drive the running dim-electron renderer over CDP
// for the conversation-state captures added on 2026-11-09.
// Usage: node interact-conversation.mjs <action>
// Actions:
//   list                       print page urls/titles
//   click-session              click the seeded session row in the sidebar
//   light                      settings -> 外观 -> 浅色 -> 返回
//   dark                       settings -> 外观 -> 深色 -> 返回
//   shot-text                  dump visible text of the main page
//   probe-branch               report whether any sidebar/timeline node mentions a session branch
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

if (action === 'list') {
  for (const p of pages) console.log(`${p.url()} :: ${await p.title()}`);
} else if (action === 'click-session') {
  const row = page.locator('[data-testid="sidebar-session-item"]').first();
  console.log('sidebar session rows:', await page.locator('[data-testid="sidebar-session-item"]').count());
  await row.click({ timeout: 5000 });
  console.log('clicked sidebar session row');
  // park the pointer away from the row so hover-only affordances (pin / delete)
  // and the row's marquee animation do not leak into the screenshot
  await page.mouse.move(600, 400);
  await page.waitForTimeout(1800);
} else if (action === 'light' || action === 'dark') {
  const theme = action === 'light' ? '浅色' : '深色';
  await page.locator('[aria-label="设置"]').first().click({ timeout: 5000 });
  await page.waitForTimeout(2000);
  await page.getByText('外观', { exact: true }).first().click({ timeout: 5000 });
  await page.waitForTimeout(1500);
  await page.getByText(theme, { exact: true }).first().click({ timeout: 5000 });
  await page.waitForTimeout(1500);
  await page.getByText('返回', { exact: true }).first().click({ timeout: 5000 });
  await page.mouse.move(600, 400);
  console.log(`switched to ${theme} and went back to the conversation`);
  await page.waitForTimeout(2000);
} else if (action === 'shot-text') {
  console.log((await page.evaluate(() => document.body.innerText)).slice(0, 4000));
} else if (action === 'probe-branch') {
  const hits = await page.evaluate(() =>
    Array.from(document.querySelectorAll('*'))
      .filter((el) => el.children.length === 0 && /seed: side branch|分支/i.test((el.textContent ?? '').trim()))
      .map((el) => {
        const r = el.getBoundingClientRect();
        return `${el.tagName} ${JSON.stringify((el.textContent ?? '').trim().slice(0, 60))} rect=${Math.round(r.x)},${Math.round(r.y)},${Math.round(r.width)},${Math.round(r.height)}`;
      }),
  );
  console.log('branch-labelled nodes:', hits.length);
  console.log(hits.join('\n'));
} else {
  console.error('unknown action', action);
  process.exit(2);
}
await browser.close();
