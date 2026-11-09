// interact.mjs -- drive the running dim-electron renderer over CDP.
// Usage: node interact.mjs <action> [arg]
// Actions:
//   list                          print page urls/titles
//   dismiss-onboarding            click the "跳过" button if the welcome modal is open
//   new-session                   click "新建会话"
//   shot-text <file>              dump visible text of the main page
import { chromium } from '/Users/eric8810/Code/dim-electron/packages/desktop/node_modules/playwright-core/index.mjs';

const action = process.argv[2];
const arg = process.argv[3];

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
} else if (action === 'dismiss-onboarding') {
  const button = page.getByText('跳过', { exact: true });
  const count = await button.count();
  console.log('skip buttons:', count);
  if (count > 0) {
    await button.first().click({ timeout: 5000 });
    console.log('clicked 跳过');
  }
  await page.waitForTimeout(1200);
} else if (action === 'new-session') {
  const entry = page.getByText('新建会话', { exact: true });
  console.log('new-session entries:', await entry.count());
  await entry.first().click({ timeout: 5000 });
  console.log('clicked 新建会话');
  await page.waitForTimeout(1500);
} else if (action === 'labels') {
  const out = await page.evaluate(() => Array.from(document.querySelectorAll('button,[role="button"],a'))
    .map((el) => `${el.tagName}\t${el.getAttribute('aria-label') ?? ''}\t${el.getAttribute('title') ?? ''}\t${(el.textContent ?? '').trim().slice(0, 40)}`));
  console.log(out.join('\n'));
} else if (action === 'click-label') {
  const target = page.locator(`[aria-label="${arg}"]`).first();
  console.log('matching aria-labels:', await page.locator(`[aria-label="${arg}"]`).count());
  await target.click({ timeout: 5000 });
  console.log('clicked aria-label', arg);
  await page.waitForTimeout(1500);
} else if (action === 'open-settings') {
  await page.locator('[aria-label="设置"]').first().click({ timeout: 5000 });
  console.log('clicked 设置');
  await page.waitForTimeout(2500);
} else if (action === 'open-skills') {
  await page.getByText('技能/插件', { exact: true }).first().click({ timeout: 5000 });
  console.log('clicked 技能/插件');
  await page.waitForTimeout(2500);
} else if (action.startsWith('text:')) {
  const label = action.slice('text:'.length);
  const loc = page.getByText(label, { exact: true });
  console.log(`"${label}" matches:`, await loc.count());
  await loc.first().click({ timeout: 5000 });
  console.log('clicked', label);
  await page.waitForTimeout(2000);
} else if (action === 'escape') {
  await page.keyboard.press('Escape');
  console.log('pressed Escape');
  await page.waitForTimeout(1500);
} else if (action === 'shot-text') {
  const text = await page.evaluate(() => document.body.innerText);
  console.log(text.slice(0, 4000));
} else {
  console.error('unknown action', action);
  process.exit(2);
}
await browser.close();
