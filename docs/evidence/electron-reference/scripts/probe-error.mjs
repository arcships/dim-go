// probe-error.mjs -- read-only DOM probe for the daemon-loss capture.
// Usage: node probe-error.mjs
// Prints one JSON object: visible body text, sidebar rows, composer/send state,
// any "session unavailable" copy, and the titlebar text. No page mutation.
import { chromium } from '/Users/eric8810/Code/dim-electron/packages/desktop/node_modules/playwright-core/index.mjs';

const browser = await chromium.connectOverCDP('http://127.0.0.1:9333');
const context = browser.contexts()[0];
const pages = context.pages();
const page = pages.find((p) => p.url().includes('/packages/renderer/dist/index.html') && !p.url().includes('surface=')) ?? pages[0];
if (!page) {
  console.error('no renderer page found');
  process.exit(1);
}

const probe = await page.evaluate(() => {
  const q = (selector) => document.querySelector(selector);
  const all = (selector) => Array.from(document.querySelectorAll(selector));
  const text = (el) => (el?.textContent ?? '').replace(/\s+/g, ' ').trim();
  const visible = (el) => {
    if (!el) return false;
    const rect = el.getBoundingClientRect();
    const style = getComputedStyle(el);
    return rect.width > 0 && rect.height > 0 && style.visibility !== 'hidden' && style.display !== 'none';
  };
  const editor = q('.ProseMirror');
  const sendButton = q('[data-testid="send-button"], button[aria-label*="发送"], button[aria-label*="Send"]');
  const unavailable = q('[data-testid="session-send-unavailable"]');
  const sessionRows = all('[data-testid="sidebar-session-item"]');
  const errorish = all('*')
    .filter((el) => el.children.length === 0)
    .map((el) => text(el))
    .filter((t) => /错误|失败|断开|连接|不可用|重试|error|failed|disconnect|retry/i.test(t))
    .filter((t, index, list) => t.length > 0 && list.indexOf(t) === index)
    .slice(0, 40);
  return {
    url: location.href,
    title: document.title,
    body_text: (document.body.innerText ?? '').slice(0, 6000),
    sidebar: {
      session_row_count: sessionRows.length,
      session_row_texts: sessionRows.map((el) => text(el)).slice(0, 10),
      sidebar_present: visible(q('aside, [data-testid="sidebar"], nav')),
    },
    titlebar_text: text(q('header')) || text(q('[data-testid="chat-titlebar"]')),
    composer: {
      editor_present: !!editor,
      editor_visible: visible(editor),
      editor_contenteditable: editor?.getAttribute('contenteditable') ?? null,
      editor_aria_disabled: editor?.getAttribute('aria-disabled') ?? null,
      send_button_present: !!sendButton,
      send_button_disabled: sendButton ? (sendButton.disabled ?? null) : null,
      send_button_aria_label: sendButton?.getAttribute('aria-label') ?? null,
      unavailable_present: !!unavailable,
      unavailable_text: unavailable ? text(unavailable) : null,
      unavailable_retry_button: unavailable ? text(unavailable.querySelector('button')) : null,
      placeholder: editor?.getAttribute('data-placeholder') ?? null,
    },
    status_bar_text: text(q('[data-testid="chat-status-bar"]')) || null,
    errorish_texts: errorish,
  };
});

console.log(JSON.stringify(probe, null, 2));
await browser.close();
