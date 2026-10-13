// Final browser fixture. Actual Vite source; every API request stays in memory.
const http = require('node:http');
const assert = require('node:assert/strict');
const company = {
  id: 'settings-fixture', kind: 'seller', name: 'Ориент Авто', legalName: 'Ориент Авто',
  legalForm: 'llc', country: null, region: null, registration: '',
  taxId: 'TEST-123456', bankRequisites: 'Тестовый банк\nРасчётный счёт TEST-001',
  email: 'company@example.test', phone: '+000 000 000', address: 'Тестовая улица, 1',
  access: 'active', accessReason: '', userType: 'dealer', verification: 'verified',
  verificationReason: '', verificationAt: '2026-10-13T10:00:00Z', revision: '1',
};
const spec = { version: '1', make: 'Chevrolet', model: 'Tracker', variant: 'Premier', year: 2026,
  bodyType: 'SUV', exteriorColor: 'Белый', interiorColor: 'Чёрный', exteriorColors: ['Белый', 'Серый'],
  interiorColors: ['Чёрный'], powertrain: 'Бензин', drivetrain: 'Передний' };
const model = { id: 'model-tracker', specification: spec, versions: [spec], revision: '1' };
const supplier = { id: 'supplier-fixture', name: 'Авто Поставка' };
const lot = { id: 'lot-fixture', supplierCompanyId: supplier.id, modelId: model.id,
  modelSpecificationVersion: '1', exteriorColor: 'Белый', interiorColor: 'Чёрный',
  totalQuantity: '12', availableQuantity: '12', heldQuantity: '0', materializedQuantity: '0',
  expectedArrival: '2027-03-01T10:00:00Z', revision: '1' };
const stock = ['XWB00000000000001', 'XWB00000000000002'].map((vin, i) => ({ vehicleId: `vehicle-${i + 1}`,
  vin, supplierCompanyId: supplier.id, supplierCompanyName: supplier.name, modelId: model.id,
  modelSpecificationVersion: '1', exteriorColor: 'Белый', interiorColor: 'Чёрный' }));
const branch = { id: 'branch-fixture', name: 'Главный филиал', address: 'Тестовая улица, 1', revision: '1' };
const warehouse = { id: 'warehouse-fixture', branchId: branch.id, name: 'Основной склад', city: 'Ташкент',
  address: 'Тестовая улица, 2', country: { label: 'Узбекистан' }, capacity: '50', occupied: '5', free: '45', revision: '1' };
const reservations = [];
const events = [];
const failures = [];
let emptyPartners = false;
let failCompany = false;
const roles = { 5192: 'admin', 5193: 'reader', 5194: 'distributor' };
const companyView = role => ({ ...company, ...(role === 'distributor' ? { ...supplier, userType: 'distributor' } : {}) });
const permissions = role => ['company.read', 'inventory.read', 'retail.read', 'commerce.read',
  ...(role === 'reader' ? [] : ['company.edit', 'company.users.manage', 'inventory.manage', 'commerce.trade'])];
function answer(res, body, status = 200) {
  res.writeHead(status, { 'Content-Type': 'application/json', 'Cache-Control': 'no-store', 'X-CSRF-Token': 'fixture-only' });
  res.end(JSON.stringify(body));
}
const list = (res, items) => answer(res, { items, nextCursor: null });
async function body(req) {
  let value = ''; for await (const chunk of req) value += chunk;
  return value ? JSON.parse(value) : {};
}
async function handle(req, res, role) {
  const url = new URL(req.url, 'http://127.0.0.1');
  if (url.pathname === '/__fixture/state') return answer(res, { company, events, failures });
  if (url.pathname === '/__fixture/configure' && req.method === 'POST') {
    const input = await body(req);
    if (input.company) Object.assign(company, input.company, { revision: String(Number(company.revision) + 1) });
    if ('emptyPartners' in input) emptyPartners = input.emptyPartners;
    if ('failCompany' in input) failCompany = input.failCompany;
    return answer(res, { configured: true });
  }
  if (!url.pathname.startsWith('/api/')) {
    const proxy = http.request({ hostname: '127.0.0.1', port: 5191, method: req.method, path: req.url,
      headers: { ...req.headers, host: '127.0.0.1:5191' } }, upstream => {
      res.writeHead(upstream.statusCode, upstream.headers); upstream.pipe(res);
    });
    proxy.on('error', error => { res.writeHead(502); res.end(error.message); });
    req.pipe(proxy); return;
  }
  const path = url.pathname.slice('/api/v1'.length);
  const view = companyView(role);
  if (req.method === 'GET') {
    if (path === '/identity/session') return answer(res, { data: {
      user: { id: `fixture-${role}`, displayName: 'Александр · тестовый профиль', status: 'active', passwordChangeRequired: false },
      roles: [], permissions: permissions(role),
      context: { revision: 'fixture-context', companyId: view.id, branchScope: { mode: 'ALL', branchIds: [] } },
      accessibleCompanies: [view], setup: { next: 'none' },
    } });
    if (path === `/identity/companies/${view.id}`) return failCompany
      ? answer(res, { error: { code: 'fixture_unavailable', message: 'Компания временно недоступна (тест)' } }, 503)
      : answer(res, { data: view, revision: view.revision });
    if (path.endsWith('/branches')) return list(res, [branch]);
    if (path === '/inventory/warehouses') return list(res, [warehouse]);
    if (path === '/inventory/vehicle-models') return list(res, [model]);
    if (path === `/inventory/vehicle-models/${model.id}`) return answer(res, { data: model, revision: model.revision });
    if (path === '/commerce/partnerships') return list(res, emptyPartners ? [] : [{ id: 'partner-fixture',
      direction: 'out', counterparty: { ...supplier, country: 'Узбекистан' }, status: 'active', statusReason: '', allowedActions: [], revision: '1' }]);
    if (path === '/commerce/supply-lots') return list(res, [lot]);
    if (path === '/commerce/reservation-stock') return list(res, url.searchParams.get('supplierCompanyId') === supplier.id ? stock : []);
    if (path === '/commerce/reservations') return list(res, reservations);
    const reservation = reservations.find(item => path === `/commerce/reservations/${item.id}`);
    if (reservation) return answer(res, { data: reservation, revision: reservation.revision });
    return answer(res, { items: [], nextCursor: null, data: {}, revision: '1' });
  }
  const input = await body(req);
  assert.notEqual(role, 'reader', 'Read-only fixture cannot mutate');
  if (path === `/identity/companies/${company.id}/submit-verification` && req.method === 'POST') {
    assert.equal(req.headers['if-match'], `"${company.revision}"`);
    assert.ok(['unverified', 'returned'].includes(company.verification));
    company.verification = 'submitted'; company.verificationReason = '';
    company.revision = String(Number(company.revision) + 1);
  } else if (path === `/identity/companies/${company.id}` && req.method === 'PATCH') {
    assert.equal(req.headers['if-match'], `"${company.revision}"`);
    assert.equal(input.taxId, company.taxId); assert.equal(input.bankRequisites, company.bankRequisites);
    Object.assign(company, input, { revision: String(Number(company.revision) + 1) });
  } else if (path === '/commerce/reservations' && req.method === 'POST') {
    assert.equal(input.supplierCompanyId, supplier.id); assert.ok(input.lines.length > 0);
    const lines = input.lines.map((line, i) => {
      const source = line.vehicleId ? stock.find(v => v.vehicleId === line.vehicleId) : lot;
      assert.ok(source); assert.match(line.quantity, /^[1-9]\d*$/);
      return { ...line, lineId: `line-${i}`, modelId: source.modelId, modelSpecificationVersion: source.modelSpecificationVersion,
        exteriorColor: source.exteriorColor, interiorColor: source.interiorColor };
    });
    const reservation = { id: `reservation-${reservations.length + 1}`, status: 'requested', buyer: { id: company.id, name: company.name },
      supplier, requested: { terms: input.desiredTerms, lines }, requestedDigest: 'fixture-request-digest', revision: '1', allowedActions: [] };
    reservations.push(reservation);
    events.push({ role, path, input });
    return answer(res, { data: reservation, revision: '1' });
  } else if (path === '/commerce/supply-lots' && req.method === 'POST') {
    assert.equal(role, 'distributor');
    events.push({ role, path, input });
    return answer(res, { data: { ...lot, ...input, id: 'lot-created' }, revision: '1' });
  } else throw new Error(`Unexpected fixture mutation: ${req.method} ${path}`);
  events.push({ role, path, input });
  answer(res, { data: company, revision: company.revision });
}
const servers = Object.entries(roles).map(([port, role]) => {
  const server = http.createServer((req, res) => handle(req, res, role).catch(error => {
    failures.push(error.message);
    answer(res, { error: { code: 'fixture_assertion', message: error.message } }, 409);
  }));
  server.listen(Number(port), '127.0.0.1', () => console.log(`${role}: http://127.0.0.1:${port}`));
  return server;
});
const stop = () => servers.forEach(server => server.close());
process.on('SIGINT', stop); process.on('SIGTERM', stop);
