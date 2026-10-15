// Isolated UI check: actual Vite assets, all API data and mutations stay in memory.
const http = require('node:http');
const assert = require('node:assert/strict');
const company = { id: 'receipt-company', kind: 'seller', name: 'Ориент Авто', access: 'active',
  userType: 'dealer', verification: 'verified', verificationReason: '' };
const warehouse = { id: 'receipt-warehouse', name: 'Основной склад', branchId: null,
  country: { label: 'Узбекистан' }, city: 'Ташкент', address: 'Тестовая улица, 1',
  capacity: '100', occupied: '0', free: '100', revision: '1' };
const spec = { version: '1', make: 'Chevrolet', model: 'Tracker', variant: 'Premier', year: 2026,
  bodyType: 'SUV', exteriorColors: ['Белый', 'Серый'], interiorColors: ['Чёрный', 'Бежевый'],
  exteriorColor: '', interiorColor: '', powertrain: 'Бензин', drivetrain: 'Передний' };
const model = { id: 'receipt-model', specification: spec, versions: [spec], revision: '1' };
const events = [], failures = [];
let failDetail = false;
const answer = (res, data, status = 200) => {
  res.writeHead(status, { 'Content-Type': 'application/json', 'Cache-Control': 'no-store', 'X-CSRF-Token': 'fixture-only' });
  res.end(JSON.stringify(data));
};
async function input(req) { let raw = ''; for await (const part of req) raw += part; return raw ? JSON.parse(raw) : {}; }
async function handle(req, res) {
  const path = new URL(req.url, 'http://127.0.0.1').pathname;
  if (path === '/__fixture/state') return answer(res, { events, failures });
  if (path === '/__fixture/configure' && req.method === 'POST') {
    const body = await input(req); if ('failDetail' in body) failDetail = body.failDetail;
    return answer(res, { configured: true });
  }
  if (!path.startsWith('/api/')) {
    const upstream = http.request({ hostname: '127.0.0.1', port: 5191, path: req.url, method: req.method,
      headers: { ...req.headers, host: '127.0.0.1:5191' } }, response => {
      res.writeHead(response.statusCode, response.headers); response.pipe(res);
    });
    upstream.on('error', error => { res.writeHead(502); res.end(error.message); });
    req.pipe(upstream); return;
  }
  const api = path.slice('/api/v1'.length);
  if (req.method === 'GET') {
    if (api === '/identity/session') return answer(res, { data: {
      user: { id: 'receipt-admin', displayName: 'Администратор · тест', status: 'active', passwordChangeRequired: false },
      roles: [], permissions: ['company.read', 'inventory.read', 'inventory.manage', 'retail.read', 'commerce.read'],
      context: { companyId: company.id, revision: 'fixture-context', branchScope: { mode: 'ALL', branchIds: [] } },
      accessibleCompanies: [company], setup: { next: 'none' },
    } });
    if (api === '/inventory/warehouses') return answer(res, { items: [warehouse], nextCursor: null });
    if (api === `/inventory/warehouses/${warehouse.id}/inventory`) {
      events.push({ action: 'inventory-read' });
      return answer(res, { data: { warehouse, vehicles: [], unidentifiedBatches: [] }, revision: warehouse.revision });
    }
    if (api === '/inventory/vehicle-models') return answer(res, { items: [model], nextCursor: null });
    if (api === `/inventory/vehicle-models/${model.id}`) return failDetail
      ? answer(res, { error: { code: 'fixture_unavailable', message: 'Спецификация временно недоступна' } }, 503)
      : answer(res, { data: model, revision: '1' });
    return answer(res, { items: [], nextCursor: null, data: {} });
  }
  const body = await input(req);
  assert.equal(req.method, 'POST');
  assert.equal(api, `/inventory/warehouses/${warehouse.id}/receipt-batches`);
  assert.equal(req.headers['if-match'], '"1"');
  assert.equal(body.modelId, model.id); assert.equal(body.modelSpecificationVersion, '1');
  assert.ok(spec.exteriorColors.includes(body.exteriorColor)); assert.ok(spec.interiorColors.includes(body.interiorColor));
  assert.ok(Number.isFinite(Date.parse(body.receivedAt)));
  assert.ok(['identified', 'unidentified'].includes(body.stock.mode));
  if (body.stock.mode === 'identified') {
    assert.ok(body.stock.vins.length); assert.equal(new Set(body.stock.vins).size, body.stock.vins.length);
    body.stock.vins.forEach(vin => assert.match(vin, /^[A-HJ-NPR-Z0-9]{17}$/));
  } else assert.match(body.stock.quantity, /^[1-9]\d*$/);
  events.push({ action: 'receipt', body, ifMatch: req.headers['if-match'] });
  answer(res, { data: {}, revision: '1' }, 201);
}
const server = http.createServer((req, res) => handle(req, res).catch(error => {
  failures.push(error.message); answer(res, { error: { code: 'fixture_assertion', message: error.message } }, 409);
}));
server.listen(5192, '127.0.0.1', () => console.log('Manual receipt fixture: http://127.0.0.1:5192/warehouses'));
const stop = () => { server.closeAllConnections(); server.close(() => process.exit(0)); setTimeout(() => process.exit(0), 200).unref(); };
process.on('SIGINT', stop); process.on('SIGTERM', stop);
