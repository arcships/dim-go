// interact-search-settings.mjs -- drive the running dim-electron renderer over
// CDP for the search + settings-route captures.
// Usage: node interact-search-settings.mjs <action> [arg]
// Actions:
//   list                     print page urls/titles
//   click-session            click the seeded session row in the sidebar
//   open-search              click [data-testid=sidebar-search] and report facts
//   hover-search             park the pointer over the sidebar search entry (shortcut hint)
//   type-query <text>        type a word into [data-testid=chat-search-input]
//   escape-search            press Escape, report whether the dialog closed and the route
//   open-settings            click the sidebar gear, report route/DOM facts
//   open-appearance          click the 外观 nav entry, report control inventory
//   back                     click 返回, report where it lands and session state
//   route                    print hash route + active session id + timeline row count
import { chromium } from '/Users/eric8810/Code/dim-electron/packages/desktop/node_modules/playwright-core/index.mjs';

const action = process.argv[2];
const arg = process.argv[3];
const CDP_PORT = process.env.CDP_PORT || '9333';
const browser = await chromium.connectOverCDP(`http://127.0.0.1:${CDP_PORT}`);
const context = browser.contexts()[0];
const pages = context.pages();
const page = pages.find((p) => p.url().includes('/packages/renderer/dist/index.html') && !p.url().includes('surface=')) ?? pages[0];
if (!page) {
  console.error('no renderer page found');
  process.exit(1);
}

const route = () =>
  page.evaluate(() => ({
    url: location.href,
    hash: location.hash,
    pathname: location.pathname,
  }));

const activeSession = () =>
  page.evaluate(() => {
    const rows = document.querySelectorAll('[data-testid="sidebar-session-item"]');
    const active = document.querySelector('[data-testid="sidebar-session-item"][data-active="true"], [data-testid="sidebar-session-item"][aria-current="page"]');
    return {
      sessionRowCount: rows.length,
      activeRowText: active ? (active.textContent ?? '').trim().slice(0, 120) : null,
      timelineNodes: document.querySelectorAll('[data-timeline-item], [data-testid="timeline-item"], [data-testid="chat-timeline"] > *').length,
      composerPresent: !!document.querySelector('[data-composer-editor-input="true"]'),
    };
  });

const box = (el) => {
  if (!el) return null;
  const r = el.getBoundingClientRect();
  return { x: Math.round(r.x), y: Math.round(r.y), w: Math.round(r.width), h: Math.round(r.height) };
};

const dialogFacts = () =>
  page.evaluate(() => {
    const dialog = document.querySelector('[data-testid="chat-search-dialog"]');
    // The dialog's backdrop is [data-slot="dialog-overlay"]; [class*="overlay"]
    // also matches the browser-guest host, which is not a scrim.
    const overlay = document.querySelector('[data-slot="dialog-overlay"]');
    const r = dialog?.getBoundingClientRect();
    const rows = Array.from(document.querySelectorAll('[data-testid="chat-search-result"]'));
    const groups = Array.from(dialog?.querySelectorAll('h3') ?? []);
    const input = document.querySelector('[data-testid="chat-search-input"]');
    return {
      dialogPresent: !!dialog,
      dialogRole: dialog?.getAttribute('role') ?? null,
      dialogAriaModal: dialog?.getAttribute('aria-modal') ?? null,
      dialogDataState: dialog?.getAttribute('data-state') ?? null,
      dialogRect: r ? { x: Math.round(r.x), y: Math.round(r.y), w: Math.round(r.width), h: Math.round(r.height) } : null,
      dialogClasses: dialog?.className ?? null,
      overlayPresent: !!overlay,
      overlayClasses: overlay?.className ?? null,
      overlayBackground: overlay ? getComputedStyle(overlay).backgroundColor : null,
      overlayZIndex: overlay ? getComputedStyle(overlay).zIndex : null,
      overlayRect: overlay ? (() => { const b = overlay.getBoundingClientRect(); return { x: Math.round(b.x), y: Math.round(b.y), w: Math.round(b.width), h: Math.round(b.height) }; })() : null,
      inputPlaceholder: input?.getAttribute('placeholder') ?? null,
      inputValue: input?.value ?? null,
      focusedTestId: document.activeElement?.getAttribute?.('data-testid') ?? null,
      groupLabels: groups.map((h) => (h.textContent ?? '').trim()),
      rowCount: rows.length,
      rows: rows.map((row) => ({
        text: (row.textContent ?? '').replace(/\s+/g, ' ').trim(),
        cls: row.className,
        sessionId: row.getAttribute('data-session-id'),
        rect: (() => { const b = row.getBoundingClientRect(); return { x: Math.round(b.x), y: Math.round(b.y), w: Math.round(b.width), h: Math.round(b.height) }; })(),
      })),
      emptyState: (() => { const e = document.querySelector('[data-testid="chat-search-empty"]'); return e ? (e.textContent ?? '').trim() : null; })(),
      moreButton: (() => { const b = document.querySelector('[data-testid="chat-search-more"]'); return b ? (b.textContent ?? '').trim() : null; })(),
      sidebarTestId: document.querySelector('[data-testid="sidebar"]')?.getAttribute('data-state') ?? null,
      mainChrome: document.querySelector('[data-chrome]')?.getAttribute('data-chrome') ?? null,
      bodyOverflow: getComputedStyle(document.body).overflow,
    };
  });

try {
if (action === 'list') {
  for (const p of pages) console.log(`${p.url()} :: ${await p.title()}`);
} else if (action === 'click-session') {
  const row = page.locator('[data-testid="sidebar-session-item"]').first();
  console.log('sidebar session rows:', await page.locator('[data-testid="sidebar-session-item"]').count());
  await row.click({ timeout: 5000 });
  await page.mouse.move(600, 400);
  await page.waitForTimeout(1800);
  console.log(JSON.stringify(await route()));
} else if (action === 'open-search' || action === 'hover-search') {
  const entry = page.locator('[data-testid="sidebar-search"]').first();
  if (action === 'hover-search') {
    await entry.hover();
    await page.waitForTimeout(400);
    const hint = await entry.evaluate((el) => {
      const badge = el.querySelector('span.ml-auto');
      return {
        badgeText: badge ? (badge.textContent ?? '').trim() : null,
        badgeDisplay: badge ? getComputedStyle(badge).display : null,
        entryText: (el.textContent ?? '').replace(/\s+/g, ' ').trim(),
        entryRect: (() => { const b = el.getBoundingClientRect(); return { x: Math.round(b.x), y: Math.round(b.y), w: Math.round(b.width), h: Math.round(b.height) }; })(),
      };
    });
    console.log(JSON.stringify({ route: await route(), sidebarSearch: hint }, null, 2));
  } else {
    await entry.click({ timeout: 5000 });
    await page.waitForTimeout(1200);
    console.log(JSON.stringify({ route: await route(), session: await activeSession(), dialog: await dialogFacts() }, null, 2));
  }
} else if (action === 'type-query') {
  if (!arg) { console.error('type-query needs a word'); process.exit(2); }
  const input = page.locator('[data-testid="chat-search-input"]');
  await input.click({ timeout: 5000 });
  await input.fill(arg);
  await page.waitForTimeout(1400);
  console.log(JSON.stringify({ route: await route(), dialog: await dialogFacts() }, null, 2));
} else if (action === 'escape-search') {
  const before = await route();
  await page.keyboard.press('Escape');
  await page.waitForTimeout(900);
  const after = await route();
  const present = await page.locator('[data-testid="chat-search-dialog"]').count();
  const focus = await page.evaluate(() => document.activeElement?.getAttribute?.('data-testid') ?? document.activeElement?.tagName ?? null);
  console.log(JSON.stringify({ before, after, dialogCountAfterEscape: present, focusedAfterEscape: focus }, null, 2));
} else if (action === 'shortcut-open') {
  // Same entry the sidebar badge advertises: mod+g / ⌘G opens the overlay.
  await page.keyboard.press('Meta+g');
  await page.waitForTimeout(900);
  console.log(JSON.stringify({ route: await route(), dialog: await dialogFacts() }, null, 2));
} else if (action === 'backdrop-click') {
  // Click below the dialog, inside the scrim.
  const before = await route();
  await page.mouse.click(400, 640);
  await page.waitForTimeout(800);
  console.log(JSON.stringify({
    before,
    after: await route(),
    dialogCountAfterBackdropClick: await page.locator('[data-testid="chat-search-dialog"]').count(),
    focusedAfterBackdropClick: await page.evaluate(() => document.activeElement?.getAttribute?.('data-testid') ?? null),
  }, null, 2));
} else if (action === 'empty-state') {
  const input = page.locator('[data-testid="chat-search-input"]');
  await input.click({ timeout: 5000 });
  await input.fill('zzzz-no-such-chat');
  await page.waitForTimeout(1400);
  console.log(JSON.stringify({ route: await route(), dialog: await dialogFacts() }, null, 2));
} else if (action === 'open-settings') {
  const before = await route();
  const gear = page.locator('[data-testid="sidebar-account-settings-shortcut"], [aria-label="设置"]').first();
  await gear.click({ timeout: 5000 });
  await page.waitForTimeout(2200);
  const facts = await page.evaluate(() => {
    const navItems = Array.from(document.querySelectorAll('[data-chrome="settings-sidebar"] button, [data-chrome="settings-sidebar"] a'))
      .map((el) => ({ text: (el.textContent ?? '').replace(/\s+/g, ' ').trim(), testid: el.getAttribute('data-testid'), rect: (() => { const b = el.getBoundingClientRect(); return { x: Math.round(b.x), y: Math.round(b.y), w: Math.round(b.width), h: Math.round(b.height) }; })() }))
      .filter((item) => item.text.length > 0);
    const back = document.querySelector('[data-testid="settings-detail-back"]');
    return {
      chromeRoot: document.querySelector('[data-chrome]')?.getAttribute('data-chrome') ?? null,
      chromeNodes: Array.from(document.querySelectorAll('[data-chrome]')).map((el) => el.getAttribute('data-chrome')),
      sidebarReplaced: !document.querySelector('[data-testid="sidebar-session-item"]'),
      appSidebarPresent: !!document.querySelector('[data-testid="sidebar"], [data-sidebar="sidebar"]'),
      titlebarText: (document.querySelector('[data-chrome="toolbar"], [data-chrome="legacy-titlebar-spacer"]')?.textContent ?? '').trim(),
      contentPaneTestId: !!document.querySelector('[data-testid="settings-content-pane"]'),
      backButtonTestId: !!back,
      backButtonRect: back ? (() => { const b = back.getBoundingClientRect(); return { x: Math.round(b.x), y: Math.round(b.y), w: Math.round(b.width), h: Math.round(b.height) }; })() : null,
      navCount: navItems.length,
      navItems: navItems.slice(0, 30),
      heading: (document.querySelector('[data-testid="settings-content-pane"] h2')?.textContent ?? '').trim() || null,
      compactLayout: !!document.querySelector('[data-testid="settings-detail-back"]'),
    };
  });
  console.log(JSON.stringify({ before, after: await route(), facts }, null, 2));
} else if (action === 'open-appearance') {
  const before = await route();
  const nav = page.locator('[data-testid="settings-nav-appearance"]');
  const navCount = await nav.count();
  let clickedBy = 'testid:settings-nav-appearance';
  if (navCount > 0) {
    await nav.first().scrollIntoViewIfNeeded();
    await nav.first().click({ timeout: 5000 });
  } else {
    // Fallback for builds without the nav test id; the visible label is 外观.
    clickedBy = 'text:外观';
    await page.getByText('外观', { exact: true }).first().click({ timeout: 5000 });
  }
  // The section swap is client-side (hash route stays #/settings): wait for the
  // pane heading instead of assuming a fixed delay.
  const deadline = Date.now() + 5000;
  let heading = null;
  while (Date.now() < deadline) {
    heading = await page.evaluate(() =>
      ((document.querySelector('[data-testid="settings-content-pane"] h2')?.textContent) ?? '').trim() || null);
    if (heading === '外观') break;
    await page.waitForTimeout(100);
  }
  await page.waitForTimeout(300);
  const controls = await page.evaluate(() => {
    const pane = document.querySelector('[data-testid="settings-content-pane"]');
    const viewport = pane?.querySelector('[data-reka-scroll-area-viewport]');
    const rectOf = (el) => {
      const b = el.getBoundingClientRect();
      return { x: Math.round(b.x), y: Math.round(b.y), w: Math.round(b.width), h: Math.round(b.height) };
    };
    const controlSelector = 'button, select, input, textarea, [role="combobox"], [role="slider"], [role="switch"], [data-slot="select-trigger"], [data-slot="slider"], [data-slot="switch"], [data-testid^="font-picker-"]';
    const describe = (el) => ({
      tag: el.tagName.toLowerCase(),
      role: el.getAttribute('role'),
      slot: el.getAttribute('data-slot'),
      testid: el.getAttribute('data-testid'),
      type: el.getAttribute('type'),
      text: (el.textContent ?? '').replace(/\s+/g, ' ').trim().slice(0, 60),
      ariaChecked: el.getAttribute('aria-checked'),
      ariaPressed: el.getAttribute('aria-pressed'),
      ariaExpanded: el.getAttribute('aria-expanded'),
      ariaValueNow: el.getAttribute('aria-valuenow'),
      ariaValueMin: el.getAttribute('aria-valuemin'),
      ariaValueMax: el.getAttribute('aria-valuemax'),
      value: typeof el.value === 'string' ? el.value.slice(0, 60) : null,
      disabled: el.disabled === true || el.getAttribute('aria-disabled') === 'true' || el.hasAttribute('data-disabled'),
      rect: rectOf(el),
    });
    const sections = Array.from(pane?.querySelectorAll('section') ?? []).map((section) => {
      const controls = [];
      const seen = new Set();
      for (const el of section.querySelectorAll(controlSelector)) {
        // Skip a thumb/input nested inside an already-recorded composite control.
        if (el.parentElement?.closest(controlSelector)) continue;
        if (seen.has(el)) continue;
        seen.add(el);
        controls.push(describe(el));
      }
      return {
        label: (section.querySelector('label')?.textContent ?? '').replace(/\s+/g, ' ').trim(),
        description: Array.from(section.querySelectorAll('p'))
          .map((p) => (p.textContent ?? '').replace(/\s+/g, ' ').trim())
          .filter((text) => text.length > 0)
          .slice(0, 4),
        controlCount: controls.length,
        controls: controls.slice(0, 12),
      };
    });
    return {
      heading: (pane?.querySelector('h2')?.textContent ?? '').trim() || null,
      scroll: viewport
        ? { scrollTop: viewport.scrollTop, scrollHeight: viewport.scrollHeight, clientHeight: viewport.clientHeight, scrollable: viewport.scrollHeight > viewport.clientHeight + 1 }
        : null,
      sectionCount: sections.length,
      sections,
      dividers: pane?.querySelectorAll('div.h-px.bg-border').length ?? 0,
      selectedTheme: (() => {
        const buttons = Array.from(pane?.querySelectorAll('section:first-of-type button') ?? []);
        // The active card carries a 2×2 dot (div.bg-primary.rounded-full); the
        // static class list also contains hover:border-primary, so match the
        // dot first and only then the exact utility class.
        const active = buttons.find((b) => b.querySelector('div.bg-primary.rounded-full'))
          ?? buttons.find((b) => b.classList.contains('border-primary'));
        return active ? { label: (active.textContent ?? '').trim(), disabled: active.disabled } : null;
      })(),
      resolvedTheme: document.documentElement.classList.contains('dark') ? 'dark' : 'light',
      prefersDark: window.matchMedia('(prefers-color-scheme: dark)').matches,
      slider: (() => { const s = pane?.querySelector('[role="slider"]'); return s ? { ariaValueNow: s.getAttribute('aria-valuenow'), ariaValueMin: s.getAttribute('aria-valuemin'), ariaValueMax: s.getAttribute('aria-valuemax') } : null; })(),
      pageText: (pane?.innerText ?? '').slice(0, 1600),
    };
  });
  console.log(JSON.stringify({ before, route: await route(), clickedBy, heading, controls }, null, 2));
} else if (action === 'scroll-appearance') {
  // Scroll the settings ScrollArea to the bottom and report where it landed.
  const scroll = await page.evaluate(() => {
    const pane = document.querySelector('[data-testid="settings-content-pane"]');
    const viewport = pane?.querySelector('[data-reka-scroll-area-viewport]') ?? pane;
    if (!viewport) return null;
    viewport.scrollTop = viewport.scrollHeight;
    return { scrollTop: viewport.scrollTop, scrollHeight: viewport.scrollHeight, clientHeight: viewport.clientHeight };
  });
  await page.waitForTimeout(300);
  const facts = await page.evaluate(() => {
    const pane = document.querySelector('[data-testid="settings-content-pane"]');
    return {
      heading: (pane?.querySelector('h2')?.textContent ?? '').trim() || null,
      sectionLabels: Array.from(pane?.querySelectorAll('section label') ?? [])
        .map((label) => (label.textContent ?? '').replace(/\s+/g, ' ').trim())
        .filter(Boolean),
      visibleTail: Array.from(pane?.querySelectorAll('section') ?? []).slice(-4).map((section) => ({
        label: (section.querySelector('label')?.textContent ?? '').replace(/\s+/g, ' ').trim(),
        rect: (() => { const b = section.getBoundingClientRect(); return { y: Math.round(b.y), h: Math.round(b.height) }; })(),
        visible: (() => { const b = section.getBoundingClientRect(); return b.top < window.innerHeight && b.bottom > 0; })(),
      })),
    };
  });
  console.log(JSON.stringify({ route: await route(), scroll, facts }, null, 2));
} else if (action === 'back') {
  const before = await route();
  const sessionsBefore = await activeSession();
  const navBack = page.locator('[data-testid="settings-back"]');
  const detailBack = page.locator('[data-testid="settings-detail-back"]');
  let backControl = 'text:返回';
  if (await navBack.count() > 0) {
    backControl = 'settings-back';
    await navBack.first().click({ timeout: 5000 });
  } else if (await detailBack.count() > 0) {
    backControl = 'settings-detail-back';
    await detailBack.first().click({ timeout: 5000 });
  } else {
    await page.getByText('返回', { exact: true }).first().click({ timeout: 5000 });
  }
  await page.waitForTimeout(2500);
  await page.mouse.move(600, 400);
  console.log(JSON.stringify({ before, after: await route(), sessionsBefore, sessionsAfter: await activeSession(), backControl }, null, 2));
} else if (action === 'route') {
  console.log(JSON.stringify({ route: await route(), session: await activeSession() }, null, 2));
} else {
  console.error('unknown action', action);
  process.exit(2);
}
} catch (error) {
  console.error(`[interact] ${action} failed: ${error?.message ?? error}`);
  if (error?.stack) console.error(error.stack.split('\n').slice(0, 8).join('\n'));
  process.exitCode = 1;
}
await browser.close();
