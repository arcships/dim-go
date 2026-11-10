// interact-states.mjs -- drive the running dim-electron renderer over CDP for
// the D6/D7 reference-state captures (hover / focus / menu / stalled loading)
// and the search-overlay motion frames.
//
// Usage: node interact-states.mjs <action> [arg...]
// Actions:
//   state                     print theme/locale/sidebar/tooltip facts
//   hover-row <index>         park the pointer on a sidebar session row
//   focus-attach              focus the composer, Tab to the attach (+) button
//   open-menu <index>         right-click a sidebar session row
//   close-menu                press Escape
//   click-row <index>         click a sidebar session row (used while the
//                             daemon is SIGSTOPped)
//   timings                   read the search dialog's animation timings
//   motion <window-id> <out-dir> <name>
//                             capture search open/close frames in slow motion
// Requires the renderer's CDP endpoint (CDP_PORT, default 9333) and, for
// motion, the macOS window id from winlist.
import { execFileSync } from 'node:child_process';
import { createHash } from 'node:crypto';
import { readFileSync, statSync, writeFileSync } from 'node:fs';
import { join } from 'node:path';
import { chromium } from '/Users/eric8810/Code/dim-electron/packages/desktop/node_modules/playwright-core/index.mjs';

const action = process.argv[2];
const CDP_PORT = process.env.CDP_PORT || '9333';
const browser = await chromium.connectOverCDP(`http://127.0.0.1:${CDP_PORT}`);
const context = browser.contexts()[0];
let pages = context.pages();
let page = pages.find((p) => p.url().includes('/packages/renderer/dist/index.html') && !p.url().includes('surface=')) ?? pages[0];
// The CDP endpoint comes up before the renderer page finishes navigating, so
// wait for the actual page instead of failing on an empty target list.
const pageDeadline = Date.now() + 30000;
while (!page && Date.now() < pageDeadline) {
  await new Promise((resolve) => setTimeout(resolve, 250));
  pages = context.pages();
  page = pages.find((p) => p.url().includes('/packages/renderer/dist/index.html') && !p.url().includes('surface=')) ?? pages[0];
}
if (!page) {
  console.error('no renderer page found');
  process.exit(1);
}

const mouseAway = async () => {
  await page.mouse.move(600, 400);
};

const sidebarRows = () => page.locator('[data-testid="sidebar-session-item"]');

const facts = () => page.evaluate(() => {
  const rows = [...document.querySelectorAll('[data-testid="sidebar-session-item"]')];
  const dialog = document.querySelector('[data-testid="chat-search-dialog"]');
  const overlay = document.querySelector('[data-slot="dialog-overlay"]');
  const tooltips = [...document.querySelectorAll('[data-slot="tooltip-content"][data-state]')];
  const active = document.activeElement;
  const box = (el) => {
    if (!el) return null;
    const r = el.getBoundingClientRect();
    return { x: Math.round(r.x), y: Math.round(r.y), w: Math.round(r.width), h: Math.round(r.height) };
  };
  return {
    theme: document.documentElement.className,
    colorScheme: getComputedStyle(document.documentElement).colorScheme,
    htmlLang: document.documentElement.lang,
    uiLang: document.querySelector('[data-testid="sidebar-new-thread"]')?.textContent?.trim() ?? null,
    windowInner: { w: innerWidth, h: innerHeight, dpr: devicePixelRatio },
    route: location.hash,
    sidebarRowCount: rows.length,
    rows: rows.map((row) => ({
      sessionId: row.getAttribute('data-session-id'),
      active: row.getAttribute('data-active'),
      loadingIcon: !!row.querySelector('[data-testid="sidebar-session-loading"]'),
      pinAffordance: !!row.querySelector('[data-testid="sidebar-session-pin"]'),
      background: getComputedStyle(row).backgroundColor,
      rect: box(row),
    })),
    dialog: dialog ? {
      present: true,
      state: dialog.getAttribute('data-state'),
      rect: box(dialog),
      opacity: getComputedStyle(dialog).opacity,
      transform: getComputedStyle(dialog).transform,
      animation: getComputedStyle(dialog).animation,
    } : { present: false },
    overlay: overlay ? {
      present: true,
      opacity: getComputedStyle(overlay).opacity,
      background: getComputedStyle(overlay).backgroundColor,
      animation: getComputedStyle(overlay).animation,
    } : { present: false },
    tooltips: tooltips.map((el) => ({ text: (el.textContent ?? '').trim(), state: el.getAttribute('data-state'), rect: box(el) })),
    timelineSkeleton: !!document.querySelector('[data-testid="timeline-skeleton"]'),
    spinnerCount: document.querySelectorAll('.animate-spin').length,
    roleStatus: [...document.querySelectorAll('[role="status"]')].map((el) => ({
      text: (el.textContent ?? '').trim(),
      testid: el.getAttribute('data-testid'),
      rect: box(el),
    })),
    focused: active ? {
      tag: active.tagName,
      testid: active.getAttribute('data-testid'),
      slot: active.getAttribute('data-slot'),
      ariaLabel: active.getAttribute('aria-label'),
      text: (active.textContent ?? '').trim().slice(0, 80),
      focusVisible: active.matches(':focus-visible'),
      boxShadow: getComputedStyle(active).boxShadow,
      outline: getComputedStyle(active).outline,
      rect: box(active),
    } : null,
  };
});

const closeDialog = async () => {
  for (let attempt = 0; attempt < 3; attempt += 1) {
    if (await page.locator('[data-testid="chat-search-dialog"]').count() === 0) return;
    await page.keyboard.press('Escape');
    await page.waitForTimeout(500);
  }
};

const animationFacts = () => page.evaluate(() => document.getAnimations().map((animation) => {
  const timing = animation.effect?.getComputedTiming?.() ?? {};
  const target = animation.effect?.target;
  return {
    name: animation.animationName ?? null,
    kind: animation.constructor.name,
    duration: timing.duration,
    delay: timing.delay,
    fill: timing.fill,
    playState: animation.playState,
    currentTime: animation.currentTime,
    target: target
      ? `${target.tagName}${target.getAttribute('data-slot') ? `[data-slot=${target.getAttribute('data-slot')}]` : ''}${target.getAttribute('data-testid') ? `[data-testid=${target.getAttribute('data-testid')}]` : ''}`
      : null,
    keyframes: animation.effect?.getKeyframes?.().map((keyframe) => ({
      offset: keyframe.offset,
      opacity: keyframe.opacity,
      transform: keyframe.transform,
      easing: keyframe.easing,
    })) ?? [],
  };
}));

const dialogAnimation = () => page.evaluate(() => {
  const dialog = document.querySelector('[data-testid="chat-search-dialog"]');
  const overlay = document.querySelector('[data-slot="dialog-overlay"]');
  const animationOf = (el) => {
    const list = el ? el.getAnimations?.() ?? [] : [];
    // Prefer the named CSSAnimation (enter/exit) over a transition that may
    // also be attached to the same element.
    const animation = list.find((candidate) => candidate.animationName) ?? list[0];
    if (animation === undefined) return null;
    const timing = animation.effect.getComputedTiming();
    return {
      name: animation.animationName ?? null,
      currentTime: animation.currentTime,
      duration: timing.duration,
      easing: timing.easing,
      keyframeEasings: animation.effect.getKeyframes().map((keyframe) => keyframe.easing),
    };
  };
  return {
    dialogPresent: !!dialog,
    dialogState: dialog?.getAttribute('data-state') ?? null,
    overlayPresent: !!overlay,
    dialog: animationOf(dialog),
    overlay: animationOf(overlay),
    computed: dialog ? {
      opacity: getComputedStyle(dialog).opacity,
      transform: getComputedStyle(dialog).transform,
    } : null,
    overlayComputed: overlay ? {
      opacity: getComputedStyle(overlay).opacity,
    } : null,
  };
});

const screenshot = (windowId, path) => {
  const started = Date.now();
  execFileSync('screencapture', ['-x', '-o', '-l', String(windowId), path], { stdio: 'ignore' });
  if (!statSync(path).isFile() || statSync(path).size === 0) {
    throw new Error(`screencapture produced no file for ${path}`);
  }
  return {
    path,
    startedAt: new Date(started).toISOString(),
    bytes: statSync(path).size,
    md5: createHash('md5').update(readFileSync(path)).digest('hex'),
  };
};

try {
  if (action === 'state') {
    await mouseAway();
    console.log(JSON.stringify(await facts(), null, 2));
  } else if (action === 'hover-row') {
    const index = Number(process.argv[3] ?? '0');
    const row = sidebarRows().nth(index);
    const box = await row.boundingBox();
    if (!box) throw new Error(`sidebar row ${index} has no box`);
    await page.mouse.move(box.x + box.width / 2, box.y + box.height / 2);
    await page.waitForTimeout(600);
    console.log(JSON.stringify({ hoveredRow: index, pointer: { x: Math.round(box.x + box.width / 2), y: Math.round(box.y + box.height / 2) }, ...(await facts()) }, null, 2));
  } else if (action === 'focus-attach') {
    await closeDialog();
    await mouseAway();
    await page.locator('[data-composer-editor-input="true"]').first().click();
    await page.waitForTimeout(250);
    await page.keyboard.press('Tab');
    await page.waitForTimeout(900);
    console.log(JSON.stringify({ tabCount: 1, note: 'click editor, one Tab lands on the composer attach (+) button', ...(await facts()) }, null, 2));
  } else if (action === 'open-menu') {
    const index = Number(process.argv[3] ?? '0');
    const waitMs = Number(process.argv[4] ?? '1200');
    await mouseAway();
    await page.keyboard.press('Escape');
    await page.waitForTimeout(300);
    const row = sidebarRows().nth(index);
    await row.click({ button: 'right' });
    await page.waitForTimeout(waitMs);
    const menu = await page.evaluate(() => {
      const root = [...document.querySelectorAll('[role="menu"]')].find((el) => el.querySelector('[data-testid^="thread-context-"]'));
      if (!root) return null;
      const r = root.getBoundingClientRect();
      return {
        rect: { x: Math.round(r.x), y: Math.round(r.y), w: Math.round(r.width), h: Math.round(r.height) },
        items: [...root.querySelectorAll('[role="menuitem"]')].map((item) => ({
          testid: item.getAttribute('data-testid'),
          text: (item.textContent ?? '').trim(),
          disabled: item.getAttribute('data-disabled'),
          renameReadiness: item.getAttribute('data-rename-readiness'),
          title: item.getAttribute('title'),
        })),
      };
    });
    console.log(JSON.stringify({ menuRow: index, menu, ...(await facts()) }, null, 2));
  } else if (action === 'close-menu') {
    await page.keyboard.press('Escape');
    await page.waitForTimeout(400);
    console.log(JSON.stringify({ menuCount: await page.locator('[role="menu"]').count() }, null, 2));
  } else if (action === 'click-row') {
    const index = Number(process.argv[3] ?? '0');
    const waitMs = Number(process.argv[4] ?? '1200');
    await sidebarRows().nth(index).click();
    await page.waitForTimeout(waitMs);
    console.log(JSON.stringify({ clickedRow: index, waitMs, ...(await facts()) }, null, 2));
  } else if (action === 'timings') {
    await closeDialog();
    await page.evaluate(() => document.querySelector('[data-testid="sidebar-search"]').click());
    await page.waitForTimeout(600);
    const open = await dialogAnimation();
    await page.keyboard.press('Escape');
    await page.waitForTimeout(80);
    const close = await dialogAnimation();
    console.log(JSON.stringify({ open, close }, null, 2));
  } else if (action === 'wait-skeleton') {
    const appPid = process.argv[3];
    const outPng = process.argv[4];
    const factsOut = process.argv[5];
    if (!appPid || !outPng) {
      console.error('wait-skeleton needs <app-pid> <out-png> [facts-json]');
      process.exit(2);
    }
    const winlist = process.env.WINLIST || '/tmp/dim-state-capture/winlist';
    const deadline = Date.now() + 30000;
    let seen = false;
    while (Date.now() < deadline) {
      seen = await page.evaluate(() => !!document.querySelector('[data-testid="timeline-skeleton"]'));
      if (seen) break;
      await page.waitForTimeout(100);
    }
    if (!seen) {
      console.error('timeline skeleton did not appear');
      process.exit(3);
    }
    const seenAt = new Date().toISOString();
    const skeletonFacts = await page.evaluate(() => {
      const skeleton = document.querySelector('[data-testid="timeline-skeleton"]');
      const box = skeleton?.getBoundingClientRect();
      return {
        theme: document.documentElement.className,
        htmlLang: document.documentElement.lang,
        bodyText: document.body.innerText.replace(/\s+/g, ' ').slice(0, 400),
        activeRows: [...document.querySelectorAll('[data-testid="sidebar-session-item"][data-active="true"]')].map((row) => row.getAttribute('data-session-id')),
        skeleton: skeleton ? {
          rect: { x: Math.round(box.x), y: Math.round(box.y), w: Math.round(box.width), h: Math.round(box.height) },
          brandMarks: skeleton.querySelectorAll('svg').length,
          text: (skeleton.textContent ?? '').trim(),
        } : null,
        spinnerCount: document.querySelectorAll('.animate-spin').length,
        roleStatus: [...document.querySelectorAll('[role="status"]')].map((el) => (el.textContent ?? '').trim()),
        composerPresent: !!document.querySelector('[data-composer-editor-input="true"]'),
      };
    });
    const line = execFileSync(winlist, [String(appPid)], { encoding: 'utf8' })
      .split('\n')
      .find((row) => row.split('\t')[1] === '0' && /DimAgent/.test(row));
    if (!line) throw new Error('no DimAgent window line from winlist');
    const wid = line.split('\t')[0];
    const shot = screenshot(wid, outPng);
    const skeletonGoneDeadline = Date.now() + 30000;
    let gone = false;
    while (Date.now() < skeletonGoneDeadline) {
      gone = await page.evaluate(() => !document.querySelector('[data-testid="timeline-skeleton"]'));
      if (gone) break;
      await page.waitForTimeout(200);
    }
    const recovered = await page.evaluate(() => ({
      skeleton: !!document.querySelector('[data-testid="timeline-skeleton"]'),
      messageCount: document.querySelectorAll('[data-timeline-item], [data-testid="timeline-item"]').length,
      bodyText: document.body.innerText.replace(/\s+/g, ' ').slice(0, 300),
    }));
    const report = {
      windowId: wid,
      windowLine: line,
      seenAt,
      screenshot: shot,
      skeletonFacts,
      skeletonGone: gone,
      recovered,
    };
    if (factsOut) writeFileSync(factsOut, `${JSON.stringify(report, null, 2)}\n`);
    console.log(JSON.stringify(report, null, 2));
  } else if (action === 'motion') {
    const windowId = process.argv[3];
    const outDir = process.argv[4];
    const name = process.argv[5] ?? 'motion-search';
    if (!windowId || !outDir) {
      console.error('motion needs <window-id> <out-dir> [name]');
      process.exit(2);
    }
    const rate = 0.02;
    const targets = [0, 75, 150, 250];
    const cdp = await context.newCDPSession(page);
    await cdp.send('Animation.enable');
    await mouseAway();
    await closeDialog();
    await page.waitForTimeout(600);

    const frames = [];
    const openTiming = {};
    const closeTiming = {};

    await cdp.send('Animation.setPlaybackRate', { playbackRate: rate });
    const openStarted = Date.now();
    await page.evaluate(() => document.querySelector('[data-testid="sidebar-search"]').click());
    await page.waitForTimeout(400);
    Object.assign(openTiming, await dialogAnimation(), { initialAnimations: await animationFacts() });

    for (const target of targets) {
      const due = openStarted + target / rate;
      await page.waitForTimeout(Math.max(0, due - Date.now()));
      const before = await dialogAnimation();
      const file = join(outDir, `${name}-${String(target).padStart(3, '0')}.png`);
      const shot = screenshot(windowId, file);
      const after = await dialogAnimation();
      frames.push({
        phase: 'open',
        targetMs: target,
        actualAnimationMs: {
          before: before.dialog?.currentTime ?? null,
          after: after.dialog?.currentTime ?? null,
        },
        dialogPresent: after.dialogPresent,
        dialogState: after.dialogState,
        computed: after.computed,
        overlayComputed: after.overlayComputed,
        file,
        bytes: shot.bytes,
        md5: shot.md5,
        capturedAt: shot.startedAt,
      });
    }

    // Close: Escape starts the exit animation; capture the same four offsets.
    const closeStarted = Date.now();
    await page.keyboard.press('Escape');
    await page.waitForTimeout(200);
    Object.assign(closeTiming, await dialogAnimation());

    for (const target of targets) {
      const due = closeStarted + target / rate;
      await page.waitForTimeout(Math.max(0, due - Date.now()));
      const before = await dialogAnimation();
      const file = join(outDir, `${name}-close-${String(target).padStart(3, '0')}.png`);
      const shot = screenshot(windowId, file);
      const after = await dialogAnimation();
      frames.push({
        phase: 'close',
        targetMs: target,
        actualAnimationMs: {
          before: before.dialog?.currentTime ?? null,
          after: after.dialog?.currentTime ?? null,
        },
        dialogPresent: after.dialogPresent,
        dialogState: after.dialogState,
        computed: after.computed,
        overlayComputed: after.overlayComputed,
        file,
        bytes: shot.bytes,
        md5: shot.md5,
        capturedAt: shot.startedAt,
      });
    }

    await cdp.send('Animation.setPlaybackRate', { playbackRate: 1 });
    await page.keyboard.press('Escape');
    await page.waitForTimeout(500);

    const report = {
      name,
      method: `CDP Animation.setPlaybackRate(${rate}) slow-motion; screencapture -x -o -l <window> at wall-clock offsets target/${rate}ms; actual animation times read from each element's Animation.currentTime before and after the screenshot`,
      playbackRate: rate,
      openTiming,
      closeTiming,
      frames,
      dialogAfterClose: (await facts()).dialog,
      capturedAt: new Date().toISOString(),
    };
    writeFileSync(join(outDir, `${name}-frames.json`), `${JSON.stringify(report, null, 2)}\n`);
    console.log(JSON.stringify({
      report: join(outDir, `${name}-frames.json`),
      openTiming,
      closeTiming,
      frames: frames.map((frame) => ({
        phase: frame.phase,
        targetMs: frame.targetMs,
        actualAnimationMs: frame.actualAnimationMs,
        dialogPresent: frame.dialogPresent,
        file: frame.file,
      })),
      dialogAfterClose: report.dialogAfterClose,
    }, null, 2));
  } else {
    console.error('unknown action', action);
    process.exit(2);
  }
} finally {
  await browser.close();
}
