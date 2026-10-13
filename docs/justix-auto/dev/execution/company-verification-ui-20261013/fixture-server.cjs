// Isolated final browser fixture: serves the actual Vite UI, never proxies API calls.
// Run with the repository-pinned Node, then use CUA at the printed local origins.
const http = require('node:http');
const assert = require('node:assert/strict');
const roles = { 5192: 'company-admin', 5193: 'platform-admin', 5194: 'company-reader', 5195: 'platform-reader' };
const company = {
  id: 'verification-fixture', kind: 'seller', name: 'Проверка Авто', legalName: 'Проверка Авто',
  legalForm: 'llc', country: { label: 'Узбекистан' }, region: null, registration: '',
  taxId: 'TEST-123456', bankRequisites: 'Тестовый банк · счёт TEST-001',
  email: 'company@example.test', phone: '+000 000 000', address: 'Тестовая улица, 1',
  access: 'active', accessReason: '', userType: 'dealer', verification: 'unverified',
  verificationReason: '', verificationAt: null, revision: '1',
};
const events = [];
const failures = [];
const servers = [];
let companyReadFailure = false;
const viewCompany = role => role === 'company-reader'
  ? { ...company, verification: 'unverified', verificationReason: '', userType: null }
  : role === 'platform-reader' ? { ...company, verification: 'submitted', verificationReason: '', userType: null } : company;
const answer = (res, body, status = 200) => {
  res.writeHead(status, { 'Content-Type': 'application/json', 'Cache-Control': 'no-store', 'X-CSRF-Token': 'fixture-only' });
  res.end(JSON.stringify(body));
};
const envelope = () => ({ data: company, revision: company.revision });
const permissions = role => role.startsWith('platform')
  ? ['platform.directory.read', ...(role === 'platform-admin' ? ['platform.companies.access', 'platform.companies.edit'] : [])]
  : ['company.read', 'inventory.read', 'retail.read', ...(role === 'company-admin' ? ['company.edit'] : [])];

async function handle(req, res, role) {
  const url = new URL(req.url, 'http://127.0.0.1');
  if (url.pathname === '/__fixture/state' && req.method === 'GET') return answer(res, { company, events, failures });
  if (url.pathname === '/__fixture/fail-company-read' && req.method === 'POST') {
    companyReadFailure = true;
    return answer(res, { armed: true });
  }
  if (url.pathname === '/__fixture/restore-company-read' && req.method === 'POST') {
    companyReadFailure = false;
    return answer(res, { restored: true });
  }
  if (!url.pathname.startsWith('/api/')) {
    const proxy = http.request({ hostname: '127.0.0.1', port: 5191, method: req.method, path: req.url,
      headers: { ...req.headers, host: '127.0.0.1:5191' } }, upstream => {
      res.writeHead(upstream.statusCode, upstream.headers); upstream.pipe(res);
    });
    proxy.on('error', error => { res.writeHead(502); res.end(error.message); });
    req.pipe(proxy);
    return;
  }
  const path = url.pathname.slice('/api/v1'.length);
  if (req.method === 'GET') {
    if (path === '/identity/session') {
      const platform = role.startsWith('platform');
      events.push({ role, action: 'session', verification: company.verification });
      return answer(res, { data: {
        user: { id: role, displayName: role, status: 'active', passwordChangeRequired: false },
        roles: [], permissions: permissions(role),
        context: { revision: 'fixture-context', companyId: platform ? null : company.id,
          branchScope: { mode: 'ALL', branchIds: [] } },
        accessibleCompanies: platform ? [] : [viewCompany(role)], setup: { next: 'none' },
      } });
    }
    if (path === `/identity/companies/${company.id}`) {
      if (companyReadFailure) {
        return answer(res, { error: { code: 'fixture_unavailable', message: 'Не удалось загрузить компанию (тест)' } }, 503);
      }
      return answer(res, { data: viewCompany(role), revision: company.revision });
    }
    if (path === '/identity/admin/companies') return answer(res, { items: [viewCompany(role)], nextCursor: null });
    return answer(res, { items: [], nextCursor: null, data: {}, revision: '1' });
  }
  let body = '';
  for await (const chunk of req) body += chunk;
  const input = body ? JSON.parse(body) : {};
  assert.equal(req.headers['if-match'], `"${company.revision}"`, 'Current company revision must be supplied');
  let action;
  if (path === `/identity/companies/${company.id}/submit-verification` && req.method === 'POST') {
    assert.equal(role, 'company-admin');
    assert.ok(['unverified', 'returned'].includes(company.verification));
    company.verification = 'submitted'; company.verificationReason = ''; company.verificationAt = null;
    action = 'submit';
  } else if (path === `/identity/admin/companies/${company.id}/return-verification` && req.method === 'POST') {
    assert.equal(role, 'platform-admin'); assert.equal(company.verification, 'submitted');
    assert.ok(input.reason?.trim(), 'Return requires a reason');
    company.verification = 'returned'; company.verificationReason = input.reason;
    company.verificationAt = new Date().toISOString(); action = 'return';
  } else if (path === `/identity/admin/companies/${company.id}/verify` && req.method === 'POST') {
    assert.equal(role, 'platform-admin'); assert.equal(company.verification, 'submitted');
    company.verification = 'verified'; company.verificationReason = '';
    company.verificationAt = new Date().toISOString(); action = 'verify';
  } else if (path === `/identity/companies/${company.id}` && req.method === 'PATCH') {
    assert.ok(['company-admin', 'platform-admin'].includes(role));
    assert.equal(input.taxId, company.taxId, 'Unrelated edit must preserve tax ID');
    assert.equal(input.bankRequisites, company.bankRequisites, 'Unrelated edit must preserve bank details');
    for (const key of ['name', 'legalName', 'legalForm', 'country', 'region', 'registration', 'taxId', 'bankRequisites', 'email', 'phone', 'address']) {
      if (Object.hasOwn(input, key)) company[key] = input[key];
    }
    action = 'edit';
  } else {
    throw new Error(`Unexpected fixture mutation: ${req.method} ${path}`);
  }
  events.push({ role, action, ifMatch: req.headers['if-match'], verification: company.verification });
  company.revision = String(Number(company.revision) + 1);
  answer(res, envelope());
}

for (const [port, role] of Object.entries(roles)) {
  const server = http.createServer((req, res) => handle(req, res, role).catch(error => {
    failures.push(error.message);
    answer(res, { error: { code: 'fixture_assertion', message: error.message } }, 409);
  }));
  server.listen(Number(port), '127.0.0.1', () => console.log(`${role}: http://127.0.0.1:${port}`));
  servers.push(server);
}
const stop = () => { for (const server of servers) server.close(); };
process.on('SIGINT', stop);
process.on('SIGTERM', stop);
