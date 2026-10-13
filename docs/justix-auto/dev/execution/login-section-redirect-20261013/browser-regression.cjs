const assert = require('node:assert/strict');
const { chromium, expect } = require('/Users/bakhromachilov/startups/justixauto/web/node_modules/@playwright/test');
const baseURL = 'http://127.0.0.1:5191';
const seller = (id, name) => ({ id, name, kind: 'seller', access: 'active', userType: 'dealer', verification: 'verified', verificationReason: '' });
const a = seller('seller-a', 'Fixture Dealer A');
const b = seller('seller-b', 'Fixture Dealer B');
const bank = { ...a, id: 'bank-a', name: 'Fixture Bank', kind: 'bank', userType: null };
const memberships = { admin: [], dealer: [a], multi: [a, b], banker: [bank], blocked: [] };
const results = [];
let browser;

async function fixture(initialUser = 'admin', initialCompany = null, holdContext = false) {
  const context = await browser.newContext({ viewport: { width: 1440, height: 1000 }, locale: 'ru-RU', serviceWorkers: 'block' });
  const state = { user: initialUser, company: initialCompany, contextWrites: [], pending: null, errors: [], privilegedAfterLogin: [], deniedVisible: false };
  await context.addInitScript(() => {
    localStorage.setItem('justixauto:last-section:v1:dealer:seller-a', 'admin');
    new MutationObserver(() => {
      if (Array.from(document.querySelectorAll('h1,h2,h3')).some((node) => node.textContent === 'Нет доступа')) window.__sawNoAccess = true;
    }).observe(document, { childList: true, subtree: true });
  });
  await context.route('**/api/v1/**', async (route) => {
    const request = route.request();
    const path = new URL(request.url()).pathname.slice('/api/v1'.length);
    const respond = (body, status = 200) => route.fulfill({ status, contentType: 'application/json', body: JSON.stringify(body) });
    if (path === '/identity/session/logout') {
      state.user = null; state.company = null;
      return respond({ data: null });
    }
    if (path === '/identity/session/login') {
      const { login } = request.postDataJSON();
      assert.ok(memberships[login], 'Only synthetic fixture users are accepted');
      state.user = login; state.company = null;
      return respond({ data: null });
    }
    if (path === '/identity/session/context') {
      const { companyId } = request.postDataJSON();
      assert.ok(memberships[state.user].some((company) => company.id === companyId));
      state.contextWrites.push({ user: state.user, companyId, ifMatch: request.headers()['if-match'] });
      if (holdContext) await new Promise((resolve) => { state.pending = resolve; });
      state.company = companyId;
      return respond({ data: null });
    }
    if (path === '/identity/session') {
      if (!state.user) return respond({ error: { code: 'unauthenticated', message: 'Sign in' } }, 401);
      const permissions = state.user === 'admin' ? ['platform.directory.read'] : state.company ?
        (state.user === 'banker' ? ['financing.read'] : ['retail.read', 'commerce.read', 'inventory.read']) : [];
      return respond({ data: {
        user: { id: state.user, displayName: state.user, status: 'active', passwordChangeRequired: false },
        roles: [], permissions,
        context: { revision: state.company || 'initial', companyId: state.company, branchScope: { mode: 'ALL', branchIds: [] } },
        accessibleCompanies: memberships[state.user], setup: { next: 'none' },
      } });
    }
    if (state.user !== 'admin' && path.startsWith('/identity/admin/')) state.privilegedAfterLogin.push(path);
    assert.equal(request.method(), 'GET', `Unexpected mutation ${path}`);
    return respond({ items: [], nextCursor: null, asOf: '', data: {}, revision: '1' });
  });
  const page = await context.newPage();
  page.on('pageerror', (error) => state.errors.push(error.message));
  return { context, page, state };
}

async function loginAfterAdmin(f, user) {
  await f.page.goto(`${baseURL}/admin/companies`);
  await expect(f.page.getByRole('heading', { name: 'Компании', exact: true })).toBeVisible();
  await f.page.getByRole('button', { name: 'Пользователь admin', exact: true }).click();
  await f.page.getByRole('button', { name: 'Выйти', exact: true }).click();
  await expect(f.page.getByRole('heading', { name: 'Вход', exact: true })).toBeVisible();
  assert.equal(new URL(f.page.url()).pathname, '/admin/companies');
  await f.page.getByLabel('Логин', { exact: true }).fill(user);
  await f.page.getByLabel('Пароль', { exact: true }).fill('fixture-password');
  await f.page.getByRole('button', { name: 'Войти', exact: true }).click();
}

async function finish(f, name, allowDenied = false) {
  assert.deepEqual(f.state.errors, [], 'No uncaught page errors');
  assert.deepEqual(f.state.privilegedAfterLogin, [], 'No Admin reads as the next user');
  if (!allowDenied) assert.equal(await f.page.evaluate(() => !!window.__sawNoAccess), false, 'No transient denial');
  results.push({ scenario: name, url: f.page.url(), contextWrites: f.state.contextWrites, passed: true });
  await f.context.close();
}

(async () => {
  browser = await chromium.launch({ headless: true, executablePath: '/Users/bakhromachilov/Library/Caches/ms-playwright/chromium_headless_shell-1223/chrome-headless-shell-mac-arm64/chrome-headless-shell' });
  try {
    let f = await fixture('admin', null, true);
    await loginAfterAdmin(f, 'dealer');
    await expect.poll(() => Boolean(f.state.pending)).toBe(true);
    await expect(f.page.getByRole('heading', { name: 'Дашборд', exact: true })).toHaveCount(0);
    f.state.pending();
    await expect(f.page).toHaveURL(`${baseURL}/dashboard`);
    await expect(f.page.getByRole('heading', { name: 'Дашборд', exact: true })).toBeVisible();
    assert.equal(f.state.contextWrites.length, 1);
    assert.equal(f.state.contextWrites[0].ifMatch, '"initial"');
    await f.page.screenshot({ path: '/private/tmp/justixauto-login-redirect-dealer.png', fullPage: true });
    await finish(f, 'Admin logout -> dealer login; confirmed company before rendering; revoked remembered Admin ignored');

    f = await fixture();
    await loginAfterAdmin(f, 'multi');
    await expect(f.page.getByRole('heading', { name: 'Выберите компанию', exact: true })).toBeVisible();
    assert.equal(f.state.contextWrites.length, 0);
    await f.page.getByRole('button', { name: 'Fixture Dealer B', exact: true }).click();
    await expect(f.page).toHaveURL(`${baseURL}/dashboard`);
    await expect(f.page.getByRole('heading', { name: 'Дашборд', exact: true })).toBeVisible();
    assert.equal(f.state.contextWrites[0].companyId, 'seller-b');
    await finish(f, 'Multiple memberships require an explicit company choice');

    f = await fixture('dealer', 'seller-a');
    await f.page.goto(`${baseURL}/finance/applications?old=1#previous-user`);
    await expect(f.page).toHaveURL(`${baseURL}/dashboard`);
    await expect(f.page.getByRole('heading', { name: 'Дашборд', exact: true })).toBeVisible();
    assert.equal(f.state.contextWrites.length, 0);
    await finish(f, 'Stale provider URL without matching membership returns to accessible section');

    f = await fixture('banker', 'bank-a');
    await f.page.goto(`${baseURL}/finance/applications?q=keep#part`);
    await expect(f.page.getByRole('heading', { name: 'Заявки', exact: true })).toBeVisible();
    await expect(f.page).toHaveURL(`${baseURL}/finance/applications?q=keep#part`);
    assert.equal(f.state.contextWrites.length, 0);
    await finish(f, 'Permitted deep link retains its pathname, query and hash');

    f = await fixture('blocked');
    await f.page.goto(`${baseURL}/admin/companies`);
    await expect(f.page).toHaveURL(`${baseURL}/`);
    await expect(f.page.getByRole('heading', { name: 'Нет доступа', exact: true })).toBeVisible();
    assert.equal(f.state.contextWrites.length, 0);
    await finish(f, 'An account with no grants or companies still receives no access', true);
    console.log(JSON.stringify({ browser: 'Chromium', fixture: 'Actual local frontend with intercepted API only; no live backend mutations', results }, null, 2));
  } finally { await browser.close(); }
})().catch((error) => { console.error(error); process.exitCode = 1; });
