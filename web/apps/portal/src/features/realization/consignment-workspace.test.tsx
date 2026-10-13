// @vitest-environment jsdom
import { createElement } from 'react';
import { MemoryRouter } from 'react-router-dom';
import { QueryClient, QueryClientProvider } from '@tanstack/react-query';
import { cleanup, fireEvent, render, screen, waitFor } from '@testing-library/react';
import { afterEach, describe, expect, it, vi } from 'vitest';
import { SessionGate } from '@justixauto/kit';
import type { Consignment, ConsignmentReturn, Invoice, Order, Vehicle } from './data';
import { ConsignmentWorkspace } from './consignment-workspace';
import { OrderDialog } from './pages/trade';
import { VehicleDialog } from './pages/inventory';

afterEach(() => { cleanup(); vi.unstubAllGlobals(); });
const response = (body: unknown, status = 200) => new Response(JSON.stringify(body), { status, headers: { 'Content-Type': 'application/json' } });
const m = (amountMinor: string) => ({ amountMinor, currency: 'USD' });
const order = (mode: 'purchase' | 'consignment' = 'consignment', partial = false): Order => ({
  id: 'o1', party: 'supplier', buyer: { id: 'dealer', name: 'Dealer' }, supplier: { id: 'supplier', name: 'Supplier' },
  source: 'reservation', terms: {
    lines: [{ lineId: 'l1', modelId: 'model-1', quantity: '1', unitPrice: m('10000') }],
    route: 'local', deliveryTerms: '', warrantyTerms: '', serviceTerms: '', paymentSchedule: [{ amount: m('5000'), dueDate: '2030-06-01' }],
    reservation: { reservationId: 'r1', bookingDigest: 'booking', continuationDigest: 'accepted', continuationRevision: 3,
      mode, allowPartialPurchaseHandover: partial, ...(partial ? { minimumPaid: m('2000') } : {}),
      handoverConditions: 'При осмотре', handoverTerms: 'Передать по акту' },
  },
  total: m('10000'), status: 'accepted', statusReason: '', allocations: [{ orderLineId: 'l1', vehicleId: 'v1', vin: 'VIN-1', status: 'allocated', shipmentId: null }],
  shipments: [], addenda: [], allowedActions: ['ship', 'ship-quantity'], revision: '5', updatedAt: '2030-05-01T00:00:00Z',
});
const invoice: Invoice = { id: 'i1', orderId: 'o1', total: m('10000'), status: 'issued', paid: m('3000'), pending: m('0'),
  outstanding: m('7000'), paymentEvidence: [], allowedActions: [], revision: '2', schedule: [] };
const unit = (changes: Record<string, unknown> = {}) => ({ id: 'u1', orderLineId: 'l1', vehicleId: 'v1', vin: 'VIN-1', status: 'held', revision: '7',
  handedOverAt: '2030-05-02T12:00:00Z', ...changes });
const agreement = (changes: Record<string, unknown> = {}): Consignment => ({ id: 'a1', orderId: 'o1', reservationId: 'r1',
  supplierCompanyId: 'supplier', dealerCompanyId: 'dealer', bookingDigest: 'booking', continuationDigest: 'accepted', continuationRevision: '3',
  revision: '1', units: [unit()], ...changes } as Consignment);
const returnRequest = (changes: Record<string, unknown> = {}): ConsignmentReturn => ({ id: 'rr1', unitId: 'u1', agreementId: 'a1', orderId: 'o1', vehicleId: 'v1',
  supplierCompanyId: 'supplier', dealerCompanyId: 'dealer', destinationWarehouseId: 'w1', reason: 'Невостребован', status: 'pending',
  requestedBy: 'supplier-user', requestedAt: '2030-05-03T12:00:00Z', revision: '4', allowedActions: ['confirm', 'reject'], ...changes } as ConsignmentReturn);
const vehicle = (changes: Record<string, unknown> = {}): Vehicle => ({ id: 'v1', vin: 'VIN-1', modelId: 'model-1', modelSpecificationVersion: '1',
  placement: { warehouseId: 'w1', placedAt: '2030-05-02T12:00:00Z' }, reserved: false,
  ownerCompanyId: 'supplier', custodianCompanyId: 'dealer', revision: '2', ...changes } as Vehicle);
const session = (company: 'supplier' | 'dealer', verified = true, trade = true) => ({ data: {
  user: { id: `${company}-user`, displayName: 'User', status: 'active', passwordChangeRequired: false }, roles: [],
  permissions: trade ? ['commerce.read', 'commerce.trade'] : ['commerce.read'],
  context: { revision: '1', companyId: company, branchScope: { mode: 'ALL', branchIds: [] } },
  accessibleCompanies: [{ id: company, name: company, kind: 'seller', userType: verified ? (company === 'supplier' ? 'distributor' : 'dealer') : null,
    verification: verified ? 'verified' : 'unverified', access: 'full' }], setup: { next: 'none' },
} });

function api(options: {
  company?: 'supplier' | 'dealer'; verified?: boolean; trade?: boolean; mode?: 'purchase' | 'consignment'; partial?: boolean;
  agreement?: Consignment; history?: ConsignmentReturn[]; pages?: (offset: number) => Response;
  vehicle?: Vehicle; vehicleDenied?: boolean; onWrite?: (url: string, init: RequestInit) => Response;
} = {}) {
  const company = options.company ?? 'supplier';
  const state = { agreement: options.agreement ?? agreement(), history: options.history ?? [], failedRead: false, postCount: 0 };
  const fetcher = vi.fn(async (input: RequestInfo | URL, init?: RequestInit) => {
    const url = String(input);
    const method = init?.method ?? 'GET';
    if (url.endsWith('/identity/session')) return response(session(company, options.verified ?? true, options.trade ?? true));
    if (method === 'POST') {
      state.postCount++;
      if (options.onWrite) return options.onWrite(url, init!);
      if (url.includes('/consignments/vehicles/v1/return-requests')) {
        state.agreement = agreement({ units: [unit({ revision: '8', pendingReturnId: 'rr1' })] });
        state.history = [returnRequest({ allowedActions: company === 'dealer' ? ['confirm', 'reject'] : [] })];
        return response({ data: state.history[0], revision: '1' }, 201);
      }
      if (url.endsWith('/confirm')) {
        state.agreement = agreement({ units: [unit({ status: 'returned', revision: '8', returnedAt: '2030-05-04T12:00:00Z' })] });
        state.history = [returnRequest({ status: 'confirmed', revision: '5', allowedActions: [], decidedAt: '2030-05-04T12:00:00Z', movedAt: '2030-05-04T12:00:00Z' })];
        return response({ data: state.history[0], revision: '5' });
      }
      if (url.endsWith('/reject')) {
        state.agreement = agreement({ units: [unit({ revision: '8' })] });
        state.history = [returnRequest({ status: 'rejected', revision: '5', allowedActions: [], decidedAt: '2030-05-04T12:00:00Z', decisionReason: 'На продаже' })];
        return response({ data: state.history[0], revision: '5' });
      }
    }
    if (url.includes('/commerce/consignments/return-requests?')) {
      const offset = Number(new URL(url, 'http://example.test').searchParams.get('offset'));
      return options.pages?.(offset) ?? response({ items: state.history, nextCursor: null });
    }
    if (url.includes('/commerce/consignments/orders/o1')) return response({ data: state.agreement, revision: state.agreement.revision });
    if (url.includes('/commerce/orders/o1/invoices')) return response({ items: [invoice] });
    if (url.includes('/commerce/orders/o1')) return state.failedRead ? response({ error: { code: 'unavailable', message: 'Unavailable' } }, 503) : response({ data: order(options.mode, options.partial), revision: '5' });
    if (url.includes('/inventory/warehouses')) return response({ items: [{ id: 'w1', name: 'Склад поставщика', free: '2', capacity: '3', occupied: '1' }] });
    if (url.includes('/inventory/vehicle-units/v1')) return options.vehicleDenied ? response({ error: { code: 'forbidden', message: 'Forbidden' } }, 403) : response({ data: { vehicle: options.vehicle ?? vehicle(), specification: { make: 'M', model: 'X', variant: 'V', year: 2030, bodyType: 'Sedan', powertrain: 'EV', drivetrain: 'AWD' }, history: [] } });
    if (url.includes('/inventory/vehicle-models')) return response({ items: [] });
    if (url.includes('/inventory/vehicle-units')) return response({ items: [] });
    return response({ items: [] });
  });
  return { state, fetcher };
}

function mount(fetcher: ReturnType<typeof vi.fn>, view: 'workspace' | 'order' | 'vehicle' = 'workspace', mode: 'purchase' | 'consignment' = 'consignment', partial = false) {
  vi.stubGlobal('fetch', fetcher);
  const client = new QueryClient({ defaultOptions: { queries: { retry: false } } });
  const child = view === 'order' ? createElement(OrderDialog, { id: 'o1', onClose: vi.fn() })
    : view === 'vehicle' ? createElement(VehicleDialog, { id: 'v1', onClose: vi.fn() })
      : createElement(ConsignmentWorkspace, { order: order(mode, partial), invoices: [invoice] });
  render(createElement(QueryClientProvider, { client }, createElement(MemoryRouter, {}, createElement(SessionGate, { title: 'T' }, child))));
  return client;
}
function sent(fetcher: ReturnType<typeof vi.fn>, path: string) {
  return fetcher.mock.calls.find(c => String(c[0]).includes(path) && c[1]?.method === 'POST')?.[1] as RequestInit | undefined;
}

describe('P23 consignment and ownership workspace', () => {
  it('shows accepted partial purchase handover and the invoice Paid without adding advance again', async () => {
    const run = api({ mode: 'purchase', partial: true });
    mount(run.fetcher, 'workspace', 'purchase', true);
    expect(await screen.findByText(/Частично оплаченный выкуп разрешён соглашением; минимум оплаты/)).toBeTruthy();
    expect(screen.getByText(/Передать по акту/)).toBeTruthy();
    expect(screen.getByText(/оплачено 30/)).toBeTruthy();
    expect(screen.getByText(/к оплате 70/)).toBeTruthy();
    expect(run.fetcher.mock.calls.some(c => String(c[0]).includes('/commerce/consignments/orders/'))).toBe(false);
  });

  it('uses current nullable inventory facts rather than agreement parties, including customer delivery', async () => {
    const run = api({ vehicle: vehicle({ ownerCompanyId: null, custodianCompanyId: null, placement: null }) });
    mount(run.fetcher, 'vehicle');
    expect(await screen.findByText('Текущий собственник')).toBeTruthy();
    expect(screen.getAllByText('Нет компании')).toHaveLength(2);
  });

  it('lets the supplier request an unsold handed VIN with its unit revision and chosen warehouse without moving it', async () => {
    const run = api(); mount(run.fetcher);
    fireEvent.click(await screen.findByRole('button', { name: 'Запросить возврат VIN VIN-1' }));
    fireEvent.change(screen.getByLabelText('Склад возврата'), { target: { value: 'w1' } });
    fireEvent.change(screen.getByLabelText('Причина возврата'), { target: { value: 'Не продан' } });
    fireEvent.click(screen.getByRole('button', { name: 'Отправить запрос' }));
    await waitFor(() => expect(sent(run.fetcher, '/consignments/vehicles/v1/return-requests')).toBeTruthy());
    const request = sent(run.fetcher, '/consignments/vehicles/v1/return-requests')!;
    expect((request.headers as Headers).get('If-Match')).toBe('"7"');
    expect(JSON.parse(String(request.body))).toEqual({ destinationWarehouseId: 'w1', reason: 'Не продан' });
    expect(await screen.findByText(/Автомобиль остаётся у дилера до подтверждения/)).toBeTruthy();
    expect(run.fetcher.mock.calls.some(c => String(c[0]).includes('/receipt-decisions') || String(c[0]).includes('/inventory/custody'))).toBe(false);
  });

  it('lets an unverified dealer service a pending old obligation using the request revision and actual movement time', async () => {
    const run = api({ company: 'dealer', verified: false, history: [returnRequest()], agreement: agreement({ units: [unit({ pendingReturnId: 'rr1' })] }), vehicleDenied: true });
    mount(run.fetcher);
    expect(await screen.findByText(/Ожидает решения дилера/)).toBeTruthy();
    expect(await screen.findByText(/Текущие собственник и хранитель недоступны/)).toBeTruthy();
    fireEvent.click(screen.getByRole('button', { name: 'Подтвердить возврат VIN VIN-1' }));
    fireEvent.change(screen.getByLabelText('Фактическое время движения'), { target: { value: '2030-05-04T12:00' } });
    fireEvent.click(screen.getByRole('button', { name: 'Подтвердить движение' }));
    await waitFor(() => expect(sent(run.fetcher, '/return-requests/rr1/confirm')).toBeTruthy());
    const decision = sent(run.fetcher, '/return-requests/rr1/confirm')!;
    expect((decision.headers as Headers).get('If-Match')).toBe('"4"');
    expect(JSON.parse(String(decision.body)).movedAt).toBe(new Date('2030-05-04T12:00').toISOString());
    expect((await screen.findAllByText(/Возврат подтверждён/)).length).toBeGreaterThan(0);
    expect(screen.getByText(/к оплате 70/)).toBeTruthy();
  });

  it('records a reasoned rejection and then removes pending association for a later request', async () => {
    const run = api({ company: 'dealer', history: [returnRequest()], agreement: agreement({ units: [unit({ pendingReturnId: 'rr1' })] }) });
    mount(run.fetcher);
    fireEvent.click(await screen.findByRole('button', { name: 'Отклонить возврат VIN VIN-1' }));
    fireEvent.change(screen.getByLabelText('Причина отказа'), { target: { value: 'На продаже' } });
    fireEvent.click(screen.getByRole('button', { name: 'Отклонить запрос' }));
    await waitFor(() => expect(sent(run.fetcher, '/return-requests/rr1/reject')).toBeTruthy());
    expect((sent(run.fetcher, '/return-requests/rr1/reject')!.headers as Headers).get('If-Match')).toBe('"4"');
    expect(JSON.parse(String(sent(run.fetcher, '/return-requests/rr1/reject')!.body))).toEqual({ reason: 'На продаже' });
    expect((await screen.findAllByText(/Возврат отклонён/)).length).toBeGreaterThan(0);
    expect(screen.getByText(/причина: На продаже/)).toBeTruthy();
  });

  it('hides return controls for sold or retail-held VINs and without commerce trade', async () => {
    const run = api({ agreement: agreement({ units: [unit({ status: 'sold', currentRetailDealId: 'retail-1' }), unit({ id: 'u2', vehicleId: 'v2', vin: 'VIN-2', currentRetailDealId: 'retail-2' })] }) });
    mount(run.fetcher);
    expect(await screen.findByText(/Проданный автомобиль нельзя вернуть/)).toBeTruthy();
    expect(screen.queryByRole('button', { name: /Запросить возврат VIN/ })).toBeNull();
    cleanup();
    const noTrade = api({ trade: false }); mount(noTrade.fetcher);
    await screen.findByText(/На хранении у дилера/);
    expect(screen.queryByRole('button', { name: /Запросить возврат VIN/ })).toBeNull();
  });

  it('loads every history page after return without dealer inventory access and rejects partial history on a later page failure', async () => {
    const first = Array.from({ length: 100 }, (_, index) => returnRequest({ id: `old-${index}`, status: 'rejected', allowedActions: [], reason: `Reason ${index}` }));
    const latest = returnRequest({ id: 'terminal', status: 'confirmed', allowedActions: [], reason: 'Последний запрос', decidedAt: '2030-05-04T12:00:00Z', movedAt: '2030-05-04T12:00:00Z' });
    const run = api({ company: 'dealer', vehicleDenied: true, agreement: agreement({ units: [unit({ status: 'returned', returnedAt: '2030-05-04T12:00:00Z' })] }),
      pages: offset => offset === 0 ? response({ items: first, nextCursor: '100' }) : response({ items: [latest], nextCursor: null }) });
    mount(run.fetcher);
    expect(await screen.findByText(/Последний запрос/)).toBeTruthy();
    expect(screen.getByText(/Reason 0/)).toBeTruthy();
    expect(run.fetcher.mock.calls.some(c => String(c[0]).includes('offset=100'))).toBe(true);
    cleanup();
    const broken = api({ company: 'dealer', pages: offset => offset === 0 ? response({ items: first, nextCursor: '100' }) : response({ error: { code: 'unavailable', message: 'Unavailable' } }, 503) });
    mount(broken.fetcher);
    expect(await screen.findByText(/Не удалось загрузить полную историю возвратов/)).toBeTruthy();
    expect(screen.queryByText(/Reason 0/)).toBeNull();
  });

  it('shows a server conflict and locks a committed write until authoritative refresh succeeds without replay', async () => {
    const conflict = api({ onWrite: () => response({ error: { code: 'capacity_exceeded', message: 'Склад занят' } }, 409) });
    mount(conflict.fetcher);
    fireEvent.click(await screen.findByRole('button', { name: 'Запросить возврат VIN VIN-1' }));
    fireEvent.change(screen.getByLabelText('Склад возврата'), { target: { value: 'w1' } });
    fireEvent.change(screen.getByLabelText('Причина возврата'), { target: { value: 'Не продан' } });
    fireEvent.click(screen.getByRole('button', { name: 'Отправить запрос' }));
    expect(await screen.findByText(/На складе не хватает мест/)).toBeTruthy();
    cleanup();
    const run = api(); mount(run.fetcher);
    fireEvent.click(await screen.findByRole('button', { name: 'Запросить возврат VIN VIN-1' }));
    fireEvent.change(screen.getByLabelText('Склад возврата'), { target: { value: 'w1' } });
    fireEvent.change(screen.getByLabelText('Причина возврата'), { target: { value: 'Не продан' } });
    run.state.failedRead = true;
    fireEvent.click(screen.getByRole('button', { name: 'Отправить запрос' }));
    expect(await screen.findByText(/Действие выполнено, но данные не обновились/)).toBeTruthy();
    expect(run.state.postCount).toBe(1);
    run.state.failedRead = false;
    fireEvent.click(screen.getByRole('button', { name: 'Повторить загрузку' }));
    await waitFor(() => expect(screen.queryByText(/Действия остановлены до обновления/)).toBeNull());
    expect(run.state.postCount).toBe(1);
  });

  it('shows agreed consignment mode before the existing physical action and hides quantity shipment', async () => {
    const run = api(); mount(run.fetcher, 'order');
    expect(await screen.findByText('Консигнация')).toBeTruthy();
    expect(screen.queryByRole('button', { name: 'Отгрузить по количеству' })).toBeNull();
    fireEvent.click(screen.getByRole('button', { name: 'Отгрузить назначенные VIN' }));
    expect(screen.getByText(/Передать по акту/)).toBeTruthy();
    expect(screen.queryByRole('button', { name: 'Отгрузить по количеству' })).toBeNull();
  });
});
