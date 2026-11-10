#!/usr/bin/env node
// capture-approval-fixture.mjs -- capture the Electron approval surface in a
// real window, driven by this repository's fake app-server fixture.
//
// The fixture is the only controllable approval source available without a
// real provider/model: the production mock model (dim-app-server --mock-model,
// EchoModel) emits text only and never calls a tool, so it cannot raise a real
// approval.request. This script therefore starts the built app with
// DIMCODE_APP_SERVER_BIN=<electron-main>/tests/fixtures/fake-app-server.mjs and
// FAKE_APP_SERVER_COMMAND_APPROVAL=1, opens fixture session-1, and screenshots
// the real window after the command permission prompt appears. App, preload,
// renderer and window are real; the daemon is the fixture, not dim-app-server.
//
// Usage: node capture-approval-fixture.mjs [out-dir] [work-dir]
import { execFileSync } from 'node:child_process';
import { createHash } from 'node:crypto';
import { existsSync } from 'node:fs';
import { mkdir, mkdtemp, readFile, stat, writeFile } from 'node:fs/promises';
import { dirname, join, resolve } from 'node:path';
import { fileURLToPath } from 'node:url';
import {
  connectToElectronPage,
  launchElectronApp,
  stopElectronApp,
} from '/Users/eric8810/Code/dim-electron/packages/desktop/scripts/e2e/electron-driver.mjs';

const DIR = dirname(fileURLToPath(import.meta.url));
const OUT = resolve(process.argv[2] || join(DIR, '..'));
const WORK = resolve(process.argv[3] || '/tmp/dim-approval-capture');
const WINLIST = process.env.WINLIST || join(WORK, 'winlist');
const DESKTOP = '/Users/eric8810/Code/dim-electron/packages/desktop';
const FAKE_APP_SERVER = join(DESKTOP, 'packages', 'electron-main', 'tests', 'fixtures', 'fake-app-server.mjs');
const REFERENCE_REPO = '/Users/eric8810/Code/dim-electron';

const dismissFirstRun = async (page) => {
  for (let attempt = 0; attempt < 6; attempt += 1) {
    const skip = page.locator('[data-testid="account-login-skip"]');
    if (await skip.isVisible().catch(() => false)) {
      await skip.click();
      await skip.waitFor({ state: 'hidden', timeout: 10000 }).catch(() => null);
      continue;
    }
    const onboarding = page.locator('[data-testid="first-run-onboarding"]');
    if (await onboarding.isVisible().catch(() => false)) {
      await page.evaluate(() => window.localStorage.setItem('dimcode:onboarding:first-run-model-guide:v2', 'done'));
      await page.reload({ waitUntil: 'domcontentloaded' });
      await page.locator('[data-testid="sidebar-session-item"], [data-testid="new-thread-screen"]').first()
        .waitFor({ state: 'visible', timeout: 30000 });
      continue;
    }
    const modeSelection = page.locator('[data-testid="mode-selection-screen"]');
    if (await modeSelection.isVisible().catch(() => false)) {
      await page.locator('[data-testid="mode-selection-confirm"]').click();
      await modeSelection.waitFor({ state: 'hidden', timeout: 10000 });
      continue;
    }
    return;
  }
};

await mkdir(WORK, { recursive: true });
await mkdir(OUT, { recursive: true });
if (!existsSync(WINLIST)) {
  execFileSync('swiftc', ['-O', join(DIR, 'winlist.swift'), '-o', WINLIST], { stdio: 'inherit' });
}

const runDir = await mkdtemp('/tmp/dim-approval-capture-');
const homeDir = join(runDir, 'home');
const dimcodeHome = join(runDir, 'dimcode-home');
const userDataDir = join(runDir, 'user-data');
const projectDir = join(runDir, 'project');
await Promise.all([homeDir, dimcodeHome, userDataDir, projectDir].map((path) => mkdir(path, { recursive: true })));

let session;
try {
  session = await launchElectronApp({
    runDir,
    env: {
      ...process.env,
      HOME: homeDir,
      DIMCODE_HOME: dimcodeHome,
      DIMCODE_USER_DATA_PATH: userDataDir,
      DIMCODE_APP_SERVER_BIN: FAKE_APP_SERVER,
      DIMCODE_APP_SERVER_ENTRY: 'standalone',
      FAKE_APP_SERVER_SESSION_CWD: projectDir,
      FAKE_APP_SERVER_QUIET: '1',
      FAKE_APP_SERVER_COMMAND_APPROVAL: '1',
    },
  });
  const { page } = await connectToElectronPage(session, 30000);
  await dismissFirstRun(page);

  // Same dark + zh-CN state as the other reference captures.
  await page.locator('[aria-label="设置"]').first().click({ timeout: 10000 });
  await page.waitForTimeout(1500);
  await page.getByText('外观', { exact: true }).first().click({ timeout: 10000 });
  await page.waitForTimeout(1200);
  await page.getByText('深色', { exact: true }).first().click({ timeout: 10000 });
  await page.waitForTimeout(1200);
  await page.getByText('返回', { exact: true }).first().click({ timeout: 10000 });
  await page.waitForTimeout(1500);

  const sessionItem = page.locator('[data-testid="sidebar-session-item"][data-session-id="session-1"]').first();
  await sessionItem.waitFor({ state: 'visible', timeout: 30000 });
  await sessionItem.click();
  const prompt = page.locator('[data-testid="tool-permission-prompt"]').filter({ visible: true }).last();
  await prompt.waitFor({ state: 'visible', timeout: 30000 });
  await page.waitForTimeout(1200);
  await page.bringToFront().catch(() => null);

  const appPid = session.app.process().pid;
  const windowLine = execFileSync(WINLIST, [String(appPid)], { encoding: 'utf8' })
    .split('\n')
    .find((row) => row.split('\t')[1] === '0' && /DimAgent/.test(row));
  if (!windowLine) throw new Error('no DimAgent window line from winlist');
  const windowId = windowLine.split('\t')[0];
  const png = join(OUT, 'window-approval.png');
  execFileSync('screencapture', ['-x', '-o', '-l', windowId, png], { stdio: 'inherit' });
  if (!(await stat(png)).size) throw new Error('screencapture produced no file');

  const facts = await page.evaluate(() => {
    const prompt = document.querySelector('[data-testid="tool-permission-prompt"]');
    const options = [...(prompt?.querySelectorAll('[data-testid^="tool-permission-option-"]') ?? [])];
    return {
      theme: document.documentElement.className,
      htmlLang: document.documentElement.lang,
      route: location.hash,
      windowInner: { w: innerWidth, h: innerHeight, dpr: devicePixelRatio },
      bodyText: document.body.innerText.replace(/\s+/g, ' ').slice(0, 1200),
      promptText: prompt?.innerText?.replace(/\s+/g, ' ').trim() ?? null,
      promptRect: (() => {
        const box = prompt?.getBoundingClientRect();
        return box ? { x: Math.round(box.x), y: Math.round(box.y), w: Math.round(box.width), h: Math.round(box.height) } : null;
      })(),
      options: options.map((option) => ({
        testid: option.getAttribute('data-testid'),
        text: (option.textContent ?? '').replace(/\s+/g, ' ').trim(),
      })),
      sidebarRows: [...document.querySelectorAll('[data-testid="sidebar-session-item"]')].map((row) => ({
        sessionId: row.getAttribute('data-session-id'),
        active: row.getAttribute('data-active'),
        text: (row.textContent ?? '').replace(/\s+/g, ' ').trim().slice(0, 120),
      })),
    };
  });

  const pixels = execFileSync('sips', ['-g', 'pixelWidth', '-g', 'pixelHeight', png], { encoding: 'utf8' })
    .split('\n')
    .filter((line) => line.includes('pixel'))
    .map((line) => line.trim().split(':')[1].trim())
    .join(' ');
  const bytes = (await stat(png)).size;
  const md5 = createHash('md5').update(await readFile(png)).digest('hex');
  const capturedAt = new Date().toISOString();

  const report = {
    windowId,
    windowLine,
    appPid,
    runDir,
    homeDir,
    userDataDir,
    fakeAppServer: FAKE_APP_SERVER,
    fixtureMode: 'FAKE_APP_SERVER_COMMAND_APPROVAL=1',
    capturedAt,
    png: { path: png, pixels, bytes, md5 },
    facts,
  };
  await writeFile(join(OUT, 'window-approval.json'), `${JSON.stringify(report, null, 2)}\n`);
  const txt = [
    'state\twindow-approval',
    'action\tlaunch the built app against electron-main/tests/fixtures/fake-app-server.mjs with FAKE_APP_SERVER_COMMAND_APPROVAL=1, switch to 深色, click fixture session-1, wait for the command permission prompt, then screencapture the real window',
    `captured_at\t${capturedAt}`,
    `app_pid\t${appPid}`,
    `window_id\t${windowId}`,
    `window_row\t${windowLine.replace(/\t/g, ' | ')}`,
    `window_points\t${(() => { const b = windowLine.split('\t')[2]?.split(','); return b ? `${b[2]}x${b[3]}` : 'unknown'; })()}`,
    `png\t${png}`,
    `png_pixels\t${pixels}`,
    `png_bytes\t${bytes}`,
    `png_md5\t${md5}`,
    `dimcode_home\t${dimcodeHome}`,
    `user_data\t${userDataDir}`,
    `app_server_bin\t${FAKE_APP_SERVER}`,
    'app_server_kind\tfixture (electron-main/tests/fixtures/fake-app-server.mjs), not dim-app-server',
    'fixture_mode\tFAKE_APP_SERVER_COMMAND_APPROVAL=1',
    'approval_call\texec { command: "cargo test -p demo" }; reason: "run test"',
    'approval_options\treject_once=拒绝, allow_once=仅这次, allow_all=始终允许此命令',
    'natural_approval_gap\tthe production mock model (dim-app-server --mock-model / EchoModel) emits text only and calls no tool, so it cannot raise a real approval.request; no real provider/model configuration is available in this workspace. This image is the approval surface rendered by the real window against the fixture, not evidence of a real daemon approval round trip.',
    'locale\tvisible UI strings are zh-CN; document.documentElement.lang is "en" (existing renderer behavior, unchanged)',
    `git_head\t${execFileSync('git', ['-C', join(OUT, '..', '..', '..'), 'rev-parse', 'HEAD'], { encoding: 'utf8' }).trim()}`,
    `reference_head\t${execFileSync('git', ['-C', REFERENCE_REPO, 'rev-parse', 'HEAD'], { encoding: 'utf8' }).trim()}`,
    `dom\t${JSON.stringify(report).replace(/\s+/g, ' ')}`,
  ].join('\n');
  await writeFile(join(OUT, 'window-approval.txt'), `${txt}\n`);
  console.log(JSON.stringify({ png, json: join(OUT, 'window-approval.json'), txt: join(OUT, 'window-approval.txt'), ...report }, null, 2));
} finally {
  if (session) await stopElectronApp(session);
}
