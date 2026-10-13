// @vitest-environment jsdom
import { createElement } from 'react';
import { MemoryRouter } from 'react-router-dom';
import { QueryClient, QueryClientProvider } from '@tanstack/react-query';
import { cleanup, fireEvent, render, screen, waitFor, within } from '@testing-library/react';
import { afterEach, describe, expect, it, vi } from 'vitest';
import { SessionGate } from '@justixauto/kit';
import { ReservationWorkspace } from './reservation-workspace';
import { PurchasesPage } from './pages/purchases';

afterEach(() => { cleanup(); vi.unstubAllGlobals(); });
const response = (body: unknown, status = 200) => new Response(JSON.stringify(body), { status, headers: { 'Content-Type': 'application/json' } });
const party = (type: 'dealer' | 'distributor', verification = 'verified', trade = true) => ({
  data: { user: { id: 'u', displayName: 'U', status: 'active', passwordChangeRequired: false }, roles: [],
    permissions: trade ? ['commerce.read', 'commerce.trade'] : ['commerce.read'],
    context: { revision: '1', companyId: type, branchScope: { mode: 'ALL', branchIds: [] } },
    accessibleCompanies: [{ id: type, name: type, kind: 'seller', userType: type, verification, access: 'full' }],
    setup: { next: 'none' } },
});

const line = { lineId: 'line-1', vehicleId: 'vehicle-1', modelId: 'model-1', modelSpecificationVersion: '7', exteriorColor: 'White', interiorColor: 'Black', quantity: '1' };
const activeLine = { ...line, vehicleId: undefined, supplyLotId: 'lot-1', quantity: '3' };
const activeTerms = { terms: { unitPrice: { amountMinor: '12050', currency: 'USD' }, expiresAt: '2030-07-01T00:00:00Z', advance: { amountMinor: '1000', currency: 'USD' }, advanceDueAt: '2030-06-20T00:00:00Z' }, lines: [activeLine] };
const activeReservation = (actions: string[], extra: Record<string, unknown> = {}) => reservation('active', actions, {
  requested: activeTerms, offered: activeTerms, current: activeTerms,
  currentDigest: 'booking-digest', expiresAt: activeTerms.terms.expiresAt,
  advanceDueAt: activeTerms.terms.advanceDueAt, cancellationState: 'ready', ...extra,
});
const accounting = (extra: Record<string, unknown> = {}) => ({ reservationId: 'reservation-1',
  agreedAdvance: { amountMinor: '1000', currency: 'USD' }, accepted: { amountMinor: '0', currency: 'USD' },
  pending: { amountMinor: '0', currency: 'USD' }, refunded: { amountMinor: '0', currency: 'USD' },
  refundObligation: { amountMinor: '0', currency: 'USD' }, refundStatus: 'none', evidence: [],
  allowedActions: [], revision: '9007199254740993', ...extra });
const association = (id: string, current: boolean, vehicleId: string) => ({ id, claimId: 'claim-1',
  lineId: 'line-1', vehicleId, holderType: 'commerce-reservation', holderId: 'reservation-1', current,
  createdAt: '2030-06-01T00:00:00Z' });
const orderFromReservation = { id: 'order-1', party: 'buyer', buyer: { name: 'Buyer' }, supplier: { name: 'Supplier' },
  source: 'reservation', terms: { lines: [], paymentSchedule: [], route: 'local', deliveryTerms: '', warrantyTerms: '', serviceTerms: '' },
  total: { amountMinor: '36150', currency: 'USD' }, status: 'accepted', statusReason: '', allocations: [], shipments: [],
  addenda: [], allowedActions: [], revision: '1', updatedAt: '2030-06-01T00:00:00Z' };
function workflowFetcher(type: 'dealer' | 'distributor', initial: ReturnType<typeof activeReservation>,
  options: { allocated?: ReturnType<typeof association>[]; paid?: Record<string, unknown>; onWrite?: (url: string, init: RequestInit) => Response | Promise<Response> } = {}) {
  let value = initial;
  let rows = options.allocated ?? [];
  const fetcher = fetcherFor(type, {
    '/supply-lots': () => response({ items: [{ ...lot, availableQuantity: '2', heldQuantity: '1', materializedQuantity: '2' }] }),
    '/accounting': () => response({ data: accounting(options.paid) }),
    '/allocations': (_url, init) => {
      if (init?.method === 'POST') {
        const sent = JSON.parse(String(init.body));
        rows = [...rows, association(`association-${rows.length + 1}`, true, sent.items[0].vehicleId)];
        return response({ data: { reservationId: value.id, allocations: rows }, revision: value.revision });
      }
      return response({ data: { reservationId: value.id, allocations: rows }, revision: value.revision });
    },
    '/inventory/vehicle-units': () => response({ items: [{ id: 'vehicle-2', vin: 'VIN-2', modelId: 'model-1', modelSpecificationVersion: '7',
      exteriorColor: 'White', interiorColor: 'Black', placement: { warehouseId: 'warehouse-1', placedAt: '2030-01-01T00:00:00Z' }, reserved: false }] }),
    '/commerce/orders/order-1/invoices': () => response({ items: [] }),
    '/commerce/orders/order-1': () => response({ data: orderFromReservation }),
    '/reservations': (url, init) => {
      if (init?.method === 'POST' && options.onWrite) return options.onWrite(url, init);
      if (init?.method === 'POST') return response({ data: value, revision: value.revision });
      return url.endsWith('/reservation-1') ? response({ data: value, revision: value.revision }) : response({ items: [value] });
    },
  });
  return { fetcher, setValue: (next: ReturnType<typeof activeReservation>) => { value = next; } };
}

describe('P19c lifecycle and continuation', () => {
  it('shows conserved server lot counts, separates current VINs from history, and allocates by source line and eligible VIN', async () => {
    const run = workflowFetcher('distributor', activeReservation(['allocate']), {
      allocated: [association('old', false, 'vehicle-old'), association('new', true, 'vehicle-1')],
    });
    mount(run.fetcher);
    fireEvent.click(await screen.findByRole('button', { name: 'Открыть' }));
    await screen.findByText(/доступно 2, удержано на будущее 1, материализовано 2/);
    expect(await screen.findByText(/VIN vehicle-1/)).toBeTruthy();
    expect(screen.getByText(/VIN vehicle-old/)).toBeTruthy();
    const select = screen.getByLabelText('Согласованная строка');
    fireEvent.change(select, { target: { value: 'line-1' } });
    const vin = screen.getByLabelText('VIN для резерва') as HTMLSelectElement;
    await within(vin).findByRole('option', { name: /VIN-2/ });
    fireEvent.change(vin, { target: { value: 'vehicle-2' } });
    fireEvent.click(screen.getByRole('button', { name: 'Назначить VIN' }));
    await waitFor(() => expect(call(run.fetcher, '/allocations', 'POST')).toBeTruthy());
    expect(JSON.parse(String(call(run.fetcher, '/allocations', 'POST')!.body))).toEqual({ items: [{ lineId: 'line-1', vehicleId: 'vehicle-2' }] });
    expect((call(run.fetcher, '/allocations', 'POST')!.headers as Headers).get('If-Match')).toBe('"9007199254740993"');
    expect(run.fetcher.mock.calls.some(c => String(c[0]).includes('/inventory/vehicle-units?') && String(c[0]).includes('eligible=true'))).toBe(true);
  });

  it('shows server lifecycle states and sends supplier extension without moving the advance deadline implicitly', async () => {
    const run = workflowFetcher('distributor', activeReservation(['extend'], { cancellationState: 'awaitingEvidenceDecision' }), {
      paid: { pending: { amountMinor: '1000', currency: 'USD' }, evidence: [{ id: 'e1', amount: { amountMinor: '1000', currency: 'USD' }, paidOn: '2030-06-01', externalReference: 'R', attachmentIds: [], status: 'submitted', decisionReason: '', revision: '1', submittedBy: 'dealer-actor', createdAt: '2030-06-01T00:00:00Z' }] },
    });
    mount(run.fetcher); fireEvent.click(await screen.findByRole('button', { name: 'Открыть' }));
    await screen.findByText(/До отмены бухгалтер должен решить/);
    expect(screen.getByText(/Доступные действия по данным сервера: продлить/)).toBeTruthy();
    expect(screen.queryByRole('button', { name: 'Отменить резерв' })).toBeNull();
    fireEvent.change(screen.getByLabelText('Новый срок резерва'), { target: { value: '2030-08-01T00:00' } });
    fireEvent.change(screen.getByLabelText('Причина продления'), { target: { value: 'Поставка позже' } });
    fireEvent.click(screen.getByRole('button', { name: 'Продлить' }));
    await waitFor(() => expect(call(run.fetcher, '/extensions', 'POST')).toBeTruthy());
    expect(JSON.parse(String(call(run.fetcher, '/extensions', 'POST')!.body))).toEqual({ expiresAt: expect.any(String), reason: 'Поставка позже' });
    expect(run.fetcher.mock.calls.some(c => String(c[0]).includes('/cancellations') && c[1]?.method === 'POST')).toBe(false);
  });

  it('distinguishes accepted refund protection and released history from a cancellable rejected-evidence state', async () => {
    const acceptedRun = workflowFetcher('dealer', activeReservation(['requestRefund'], { cancellationState: 'awaitingRefund' }), {
      paid: { accepted: { amountMinor: '1000', currency: 'USD' }, refundObligation: { amountMinor: '1000', currency: 'USD' }, refundStatus: 'requested' },
    });
    mount(acceptedRun.fetcher); fireEvent.click(await screen.findByRole('button', { name: 'Открыть' }));
    await screen.findByText(/полный подтверждённый бухгалтером возврат/);
    expect(screen.queryByRole('button', { name: 'Отменить резерв' })).toBeNull();
    cleanup();
    const rejectedRun = workflowFetcher('dealer', activeReservation(['cancel'], { cancellationState: 'ready' }), {
      paid: { evidence: [{ id: 'e1', amount: { amountMinor: '1000', currency: 'USD' }, paidOn: '2030-06-01', externalReference: 'R', attachmentIds: [], status: 'rejected', decisionReason: 'Не поступило', revision: '2', submittedBy: 'dealer-actor', decidedBy: 'accountant', createdAt: '2030-06-01T00:00:00Z' }] },
    });
    mount(rejectedRun.fetcher); fireEvent.click(await screen.findByRole('button', { name: 'Открыть' }));
    await screen.findByText(/Не поступило/);
    fireEvent.change(screen.getByLabelText('Причина отмены резерва'), { target: { value: 'Больше не нужно' } });
    fireEvent.click(screen.getByRole('button', { name: 'Отменить резерв' }));
    await waitFor(() => expect(call(rejectedRun.fetcher, '/cancellations', 'POST')).toBeTruthy());
    expect(JSON.parse(String(call(rejectedRun.fetcher, '/cancellations', 'POST')!.body))).toEqual({ reason: 'Больше не нужно' });
    cleanup();
    const released = workflowFetcher('dealer', activeReservation([], { status: 'expired', cancellationState: 'expired',
      releasedAt: '2030-07-01T00:00:00Z', releaseReason: 'automatic' }));
    mount(released.fetcher); fireEvent.click(await screen.findByRole('button', { name: 'Открыть' }));
    await screen.findByText(/Причина: automatic/);
    expect(screen.getByText('Согласовано')).toBeTruthy();
  });

  it('proposes exact purchase terms with unchecked partial handover and an explicit empty schedule', async () => {
    const run = workflowFetcher('dealer', activeReservation(['proposeContinuation']));
    mount(run.fetcher); fireEvent.click(await screen.findByRole('button', { name: 'Открыть' }));
    const partial = await screen.findByRole('checkbox', { name: /Стороны явно разрешают передачу/ }) as HTMLInputElement;
    expect(partial.checked).toBe(false);
    fireEvent.change(screen.getByLabelText('Маршрут продолжения'), { target: { value: 'local' } });
    fireEvent.change(screen.getByLabelText('Согласованные условия передачи'), { target: { value: 'После проверки на складе' } });
    fireEvent.click(screen.getByRole('button', { name: 'Отправить предложение' }));
    await waitFor(() => expect(call(run.fetcher, '/continuation-proposals', 'POST')).toBeTruthy());
    const sent = JSON.parse(String(call(run.fetcher, '/continuation-proposals', 'POST')!.body));
    expect(sent).toEqual({ mode: 'purchase', allowPartialPurchaseHandover: false, handoverTerms: 'После проверки на складе',
      terms: { lines: [{ offerLineId: 'line-1', modelId: 'model-1', modelSpecificationVersion: '7', exteriorColor: 'White', interiorColor: 'Black', quantity: '3', unitPrice: { amountMinor: '12050', currency: 'USD' } }],
        route: 'local', deliveryTerms: '', warrantyTerms: '', serviceTerms: '', paymentSchedule: [] } });
  });

  it('sends explicit partial purchase minimum only when supplied, and consignment without the purchase flag', async () => {
    const run = workflowFetcher('dealer', activeReservation(['proposeContinuation']));
    mount(run.fetcher); fireEvent.click(await screen.findByRole('button', { name: 'Открыть' }));
    fireEvent.change(await screen.findByLabelText('Маршрут продолжения'), { target: { value: 'factory' } });
    fireEvent.change(screen.getByLabelText('Согласованные условия передачи'), { target: { value: 'Письменный акт' } });
    fireEvent.click(screen.getByRole('checkbox', { name: /Стороны явно разрешают передачу/ }));
    fireEvent.change(screen.getByLabelText('Согласованный минимум оплаты'), { target: { value: '20.00' } });
    fireEvent.change(screen.getByLabelText('Условия частичной передачи'), { target: { value: 'Только после акта' } });
    fireEvent.click(screen.getByRole('button', { name: 'Отправить предложение' }));
    await waitFor(() => expect(call(run.fetcher, '/continuation-proposals', 'POST')).toBeTruthy());
    const purchase = JSON.parse(String(call(run.fetcher, '/continuation-proposals', 'POST')!.body));
    expect(purchase.allowPartialPurchaseHandover).toBe(true);
    expect(purchase.minimumPaid).toEqual({ amountMinor: '2000', currency: 'USD' });
    expect(purchase.handoverConditions).toBe('Только после акта');
    cleanup();
    const consignment = workflowFetcher('distributor', activeReservation(['proposeContinuation']));
    mount(consignment.fetcher); fireEvent.click(await screen.findByRole('button', { name: 'Открыть' }));
    fireEvent.change(await screen.findByLabelText('Форма продолжения'), { target: { value: 'consignment' } });
    fireEvent.change(screen.getByLabelText('Маршрут продолжения'), { target: { value: 'local' } });
    fireEvent.change(screen.getByLabelText('Согласованные условия передачи'), { target: { value: 'На ответственное хранение' } });
    fireEvent.click(screen.getByRole('button', { name: 'Отправить предложение' }));
    await waitFor(() => expect(call(consignment.fetcher, '/continuation-proposals', 'POST')).toBeTruthy());
    const sent = JSON.parse(String(call(consignment.fetcher, '/continuation-proposals', 'POST')!.body));
    expect(sent.mode).toBe('consignment'); expect(sent.allowPartialPurchaseHandover).toBe(false);
    expect(sent).not.toHaveProperty('minimumPaid');
  });

  it('accepts the exact current digest and converts zero, some or all allocated VINs without a browser prerequisite', async () => {
    for (const count of [0, 1, 3]) {
      const proposal = { mode: count === 3 ? 'consignment' : 'purchase', terms: { lines: [{ lineId: 'order-line-1', offerLineId: 'line-1', modelId: 'model-1', modelSpecificationVersion: '7', exteriorColor: 'White', interiorColor: 'Black', quantity: '3', unitPrice: { amountMinor: '12050', currency: 'USD' } }],
        route: 'local', deliveryTerms: '', paymentSchedule: [], warrantyTerms: '', serviceTerms: '' }, allowPartialPurchaseHandover: count === 1,
        ...(count === 1 ? { minimumPaid: { amountMinor: '2000', currency: 'USD' }, handoverConditions: 'По акту' } : {}), handoverTerms: 'Акт' };
      const run = workflowFetcher('dealer', activeReservation(['acceptContinuation', 'convert'], {
        continuationProposed: proposal, continuationProposedDigest: 'proposal-digest', continuationProposedRevision: '9007199254740992', continuationProposedBy: 'distributor',
        continuationAccepted: proposal, continuationAcceptedDigest: 'proposal-digest', continuationAcceptedRevision: '9007199254740993', continuationAcceptedBy: 'dealer',
      }), { allocated: Array.from({ length: count }, (_, i) => association(`a-${i}`, true, `vehicle-${i}`)) });
      mount(run.fetcher);
      fireEvent.click(await screen.findByRole('button', { name: 'Открыть' }));
      if (count === 1) {
        await screen.findByText(/Согласованный минимум: 20.00 USD/);
        expect(screen.getByText(/передача до полной оплаты: согласована/)).toBeTruthy();
      }
      if (count === 3) await screen.findByText(/Форма: консигнация/);
      fireEvent.click(await screen.findByRole('button', { name: 'Создать согласованный заказ' }));
      await waitFor(() => expect(call(run.fetcher, '/conversions', 'POST')).toBeTruthy());
      expect(JSON.parse(String(call(run.fetcher, '/conversions', 'POST')!.body))).toEqual({ digest: 'proposal-digest', acceptedRevision: '9007199254740993' });
      cleanup();
    }
  });

  it('accepts the other party’s exact proposal digest and proposed revision, then reconciles a changed proposal after stale rejection', async () => {
    const proposal = { mode: 'consignment', terms: { lines: [{ offerLineId: 'line-1', modelId: 'model-1', modelSpecificationVersion: '7', exteriorColor: 'White', interiorColor: 'Black', quantity: '3', unitPrice: { amountMinor: '12050', currency: 'USD' } }],
      route: 'local', deliveryTerms: '', paymentSchedule: [], warrantyTerms: '', serviceTerms: '' }, allowPartialPurchaseHandover: false, handoverTerms: 'Только по акту' };
    let run!: ReturnType<typeof workflowFetcher>;
    run = workflowFetcher('dealer', activeReservation(['acceptContinuation'], { continuationProposed: proposal,
      continuationProposedDigest: 'digest-old', continuationProposedRevision: '9007199254740992', continuationProposedBy: 'distributor' }), {
      onWrite: url => {
        if (url.includes('/continuation-acceptances')) {
          run.setValue(activeReservation(['acceptContinuation'], { continuationProposed: { ...proposal, handoverTerms: 'Новый акт' },
            continuationProposedDigest: 'digest-new', continuationProposedRevision: '9007199254740994', continuationProposedBy: 'distributor', revision: '9007199254740994' }));
          return response({ error: { code: 'continuation_changed', message: 'Proposal changed' } }, 409);
        }
        return response({ data: {} });
      },
    });
    mount(run.fetcher); fireEvent.click(await screen.findByRole('button', { name: 'Открыть' }));
    fireEvent.click(await screen.findByRole('button', { name: 'Принять точное предложение' }));
    await waitFor(() => expect(call(run.fetcher, '/continuation-acceptances', 'POST')).toBeTruthy());
    const sent = call(run.fetcher, '/continuation-acceptances', 'POST')!;
    expect(JSON.parse(String(sent.body))).toEqual({ digest: 'digest-old', proposedRevision: '9007199254740992' });
    expect((sent.headers as Headers).get('If-Match')).toBe('"9007199254740993"');
    await screen.findByText(/Proposal changed/);
    await screen.findByText(/digest-new/);
    expect(screen.getByText(/Новый акт/)).toBeTruthy();
  });

  it('opens a converted order in Purchases once, and reopens an existing link without another conversion', async () => {
    const terms = { mode: 'purchase', terms: { lines: [], paymentSchedule: [], route: 'local', deliveryTerms: '', warrantyTerms: '', serviceTerms: '' },
      allowPartialPurchaseHandover: false, handoverTerms: 'Акт' };
    let release!: (value: Response) => void;
    const pending = new Promise<Response>(resolve => { release = resolve; });
    const run = workflowFetcher('dealer', activeReservation(['convert'], { continuationProposed: terms, continuationAccepted: terms,
      continuationProposedDigest: 'digest', continuationAcceptedDigest: 'digest', continuationAcceptedRevision: '7' }), {
      onWrite: url => url.includes('/conversions') ? pending : response({ data: {} }),
    });
    mount(run.fetcher, true);
    expect(await screen.findByRole('button', { name: 'Создать заказ' })).toBeTruthy();
    expect(screen.getByRole('button', { name: 'Предложения от поставщиков' })).toBeTruthy();
    fireEvent.click(screen.getByRole('button', { name: 'Резервы и поставки' }));
    fireEvent.click(await screen.findByRole('button', { name: 'Открыть' }));
    fireEvent.click(await screen.findByRole('button', { name: 'Создать согласованный заказ' }));
    expect(screen.getByRole('button', { name: 'Создать согласованный заказ' })).toHaveProperty('disabled', true);
    release(response({ data: { orderId: 'order-1' }, revision: '9007199254740994' }, 201));
    await screen.findByRole('dialog', { name: /Заказ order-1/ });
    expect(run.fetcher.mock.calls.filter(c => String(c[0]).includes('/conversions') && c[1]?.method === 'POST')).toHaveLength(1);
    cleanup();
    const linked = workflowFetcher('dealer', activeReservation([], { status: 'converted', cancellationState: 'converted', orderId: 'order-1',
      continuationProposed: terms, continuationAccepted: terms, continuationProposedDigest: 'digest', continuationAcceptedDigest: 'digest' }));
    mount(linked.fetcher, true);
    fireEvent.click(await screen.findByRole('button', { name: 'Резервы и поставки' }));
    fireEvent.click(await screen.findByRole('button', { name: 'Открыть' }));
    fireEvent.click(await screen.findByRole('button', { name: 'Открыть заказ' }));
    await screen.findByRole('dialog', { name: /Заказ order-1/ });
    expect(linked.fetcher.mock.calls.filter(c => String(c[0]).includes('/conversions') && c[1]?.method === 'POST')).toHaveLength(0);
  });
});
const spec = { version: '7', make: 'Maker', model: 'Car', variant: 'X', year: 2026, bodyType: 'sedan',
  exteriorColor: 'White', interiorColor: 'Black', exteriorColors: ['White', 'Blue'], interiorColors: ['Black', 'Tan'], powertrain: 'ICE', drivetrain: 'FWD' };
const catalog = { id: 'model-1', specification: spec, versions: [spec], revision: '3' };
const lot = { id: 'lot-1', supplierCompanyId: 'distributor', modelId: 'model-1', modelSpecificationVersion: '7',
  exteriorColor: 'White', interiorColor: 'Black', totalQuantity: '5', availableQuantity: '5',
  heldQuantity: '0', materializedQuantity: '0', expectedArrival: '2030-06-01T10:00:00Z', revision: '9007199254740993' };
const stock = { vehicleId: 'vehicle-1', vin: 'VIN-1', supplierCompanyId: 'distributor', supplierCompanyName: 'Supplier',
  modelId: 'model-1', modelSpecificationVersion: '7', exteriorColor: 'White', interiorColor: 'Black' };
const partners = [{ id: 'p', status: 'active', counterparty: { id: 'distributor', name: 'Supplier' } }];
const requested = { terms: {}, lines: [line] };
const offered = { terms: { unitPrice: { amountMinor: '9007199254740993', currency: 'USD' }, expiresAt: '2030-07-01T00:00:00Z' }, lines: [line] };
function reservation(status: string, actions: string[], extra: Record<string, unknown> = {}) {
  return { id: 'reservation-1', status, buyer: { id: 'dealer', name: 'Buyer' }, supplier: { id: 'distributor', name: 'Supplier' },
    requested, requestedDigest: 'requested-digest', offeredDigest: '', currentDigest: '', revision: '9007199254740993',
    allowedActions: actions, ...extra };
}
type Mock = ReturnType<typeof vi.fn>;
function fetcherFor(type: 'dealer' | 'distributor', overrides: Partial<Record<string, (url: string, init?: RequestInit) => Response | Promise<Response>>> = {}, verification = 'verified', trade = true) {
  return vi.fn(async (input: RequestInfo | URL, init?: RequestInit) => {
    const url = String(input);
    if (url.endsWith('/identity/session')) return response(party(type, verification, trade));
    for (const [part, handler] of Object.entries(overrides)) if (url.includes(part)) return handler!(url, init);
    if (url.includes('/partnerships')) return response({ items: partners });
    if (url.includes('/vehicle-models/model-1')) return response({ data: catalog, revision: '3' });
    if (url.includes('/vehicle-models')) return response({ items: [catalog] });
    if (url.includes('/supply-lots')) return response({ items: type === 'dealer' ? [lot] : [lot] });
    if (url.includes('/reservation-stock')) return response({ items: [stock], nextCursor: null });
    if (url.includes('/reservations')) return response({ items: [] });
    return response({ items: [] });
  });
}
function mount(fetcher: Mock, host = false) {
  vi.stubGlobal('fetch', fetcher);
  const client = new QueryClient({ defaultOptions: { queries: { retry: false } } });
  const onClose = vi.fn();
  render(createElement(QueryClientProvider, { client }, createElement(MemoryRouter, {},
    createElement(SessionGate, { title: 'T' }, host ? createElement(PurchasesPage) : createElement(ReservationWorkspace, { onClose })))));
  return { client, onClose };
}
function call(fetcher: Mock, path: string, method: string) {
  return fetcher.mock.calls.find(c => String(c[0]).includes(path) && c[1]?.method === method)?.[1] as RequestInit | undefined;
}
async function chooseSupplier() {
  fireEvent.click(await screen.findByRole('button', { name: 'Запросить резерв' }));
  const field = screen.getByLabelText('Поставщик') as HTMLSelectElement;
  await within(field).findByRole('option', { name: 'Supplier' });
  fireEvent.change(field, { target: { value: 'distributor' } });
}

describe('P19a supply and agreement', () => {
  it('discovers partner VIN through the actual list envelope and sends a no-hold request with unknown terms omitted', async () => {
    const fetcher = fetcherFor('dealer', { '/reservations': (_url, init) => init?.method === 'POST'
      ? response({ data: reservation('requested', []), revision: '1' }, 201)
      : response({ items: [] }) });
    mount(fetcher);
    await chooseSupplier();
    const vin = screen.getByLabelText('VIN') as HTMLSelectElement;
    await within(vin).findByRole('option', { name: /VIN-1/ });
    fireEvent.change(vin, { target: { value: 'vehicle-1' } });
    fireEvent.click(screen.getByRole('button', { name: 'Отправить запрос' }));
    await waitFor(() => expect(call(fetcher, '/reservations', 'POST')).toBeTruthy());
    expect(JSON.parse(String(call(fetcher, '/reservations', 'POST')!.body))).toEqual({
      supplierCompanyId: 'distributor', lines: [{ vehicleId: 'vehicle-1', quantity: '1' }], desiredTerms: {},
    });
    expect(fetcher.mock.calls.filter(c => c[1]?.method === 'POST')).toHaveLength(1);
    expect(fetcher.mock.calls.some(c => String(c[0]).includes('reservation-stock?') && c[1]?.method === 'GET')).toBe(true);
  });

  it('requests future units from an actual partner lot without fabricating a VIN or price', async () => {
    const fetcher = fetcherFor('dealer', { '/reservations': (_url, init) => init?.method === 'POST'
      ? response({ data: reservation('requested', []), revision: '1' }, 201) : response({ items: [] }) });
    mount(fetcher); await chooseSupplier();
    const lots = screen.getByLabelText('Будущая поставка') as HTMLSelectElement;
    await within(lots).findByRole('option', { name: /доступно 5/ });
    fireEvent.change(lots, { target: { value: 'lot-1' } });
    fireEvent.change(screen.getByLabelText('Количество будущей поставки'), { target: { value: '3' } });
    fireEvent.click(screen.getByRole('button', { name: 'Отправить запрос' }));
    await waitFor(() => expect(call(fetcher, '/reservations', 'POST')).toBeTruthy());
    expect(JSON.parse(String(call(fetcher, '/reservations', 'POST')!.body))).toEqual({
      supplierCompanyId: 'distributor', lines: [{ supplyLotId: 'lot-1', quantity: '3' }], desiredTerms: {},
    });
  });

  it('adds two distinct real VINs to one request without a capacity hold', async () => {
    const other = { ...stock, vehicleId: 'vehicle-2', vin: 'VIN-2' };
    const fetcher = fetcherFor('dealer', {
      '/reservation-stock': () => response({ items: [stock, other], nextCursor: null }),
      '/reservations': (_url, init) => init?.method === 'POST'
        ? response({ data: reservation('requested', []), revision: '1' }, 201) : response({ items: [] }),
    });
    mount(fetcher); await chooseSupplier();
    const vin = screen.getByLabelText('VIN') as HTMLSelectElement;
    await within(vin).findByRole('option', { name: /VIN-1/ });
    fireEvent.change(vin, { target: { value: 'vehicle-1' } });
    fireEvent.click(screen.getByRole('button', { name: 'Добавить VIN' }));
    fireEvent.change(vin, { target: { value: 'vehicle-2' } });
    fireEvent.click(screen.getByRole('button', { name: 'Отправить запрос' }));
    await waitFor(() => expect(call(fetcher, '/reservations', 'POST')).toBeTruthy());
    expect(JSON.parse(String(call(fetcher, '/reservations', 'POST')!.body)).lines).toEqual([
      { vehicleId: 'vehicle-1', quantity: '1' }, { vehicleId: 'vehicle-2', quantity: '1' },
    ]);
    expect(fetcher.mock.calls.filter(c => c[1]?.method === 'POST')).toHaveLength(1);
  });

  it('creates and edits future supply with catalog history, palettes, exact revision and refreshed capacity', async () => {
    let saved = { ...lot }; const fetcher = fetcherFor('distributor', {
      '/supply-lots': (_url, init) => {
        if (init?.method === 'POST') { saved = { ...lot, totalQuantity: '8' }; return response({ data: saved, revision: '1' }, 201); }
        if (init?.method === 'PUT') { saved = { ...saved, totalQuantity: '9', availableQuantity: '9', revision: '9007199254740994' }; return response({ data: saved, revision: saved.revision }); }
        return response({ items: [saved] });
      },
    });
    mount(fetcher);
    fireEvent.click(await screen.findByRole('button', { name: 'Добавить будущую поставку' }));
    const model = screen.getByLabelText('Модель') as HTMLSelectElement;
    await within(model).findByRole('option', { name: /Maker Car X/ });
    fireEvent.change(model, { target: { value: 'model-1' } });
    const version = screen.getByLabelText('Версия спецификации') as HTMLSelectElement;
    await within(version).findByRole('option', { name: /7/ });
    fireEvent.change(version, { target: { value: '7' } });
    fireEvent.change(screen.getByLabelText('Цвет кузова'), { target: { value: 'White' } });
    fireEvent.change(screen.getByLabelText('Цвет салона'), { target: { value: 'Black' } });
    fireEvent.change(screen.getByLabelText('Общее количество'), { target: { value: '8' } });
    fireEvent.change(screen.getByLabelText('Ожидаемое прибытие'), { target: { value: '2030-06-01T10:00' } });
    fireEvent.click(screen.getByRole('button', { name: 'Сохранить поставку' }));
    await waitFor(() => expect(call(fetcher, '/supply-lots', 'POST')).toBeTruthy());
    expect(JSON.parse(String(call(fetcher, '/supply-lots', 'POST')!.body))).toEqual({
      modelId: 'model-1', modelSpecificationVersion: '7', exteriorColor: 'White', interiorColor: 'Black',
      totalQuantity: '8', expectedArrival: expect.any(String),
    });
    const edit = await screen.findByRole('button', { name: 'Изменить' });
    fireEvent.click(edit);
    fireEvent.change(screen.getByLabelText('Общее количество'), { target: { value: '9' } });
    fireEvent.click(screen.getByRole('button', { name: 'Сохранить поставку' }));
    await waitFor(() => expect(call(fetcher, '/supply-lots/lot-1', 'PUT')).toBeTruthy());
    expect((call(fetcher, '/supply-lots/lot-1', 'PUT')!.headers as Headers).get('If-Match')).toBe('"9007199254740993"');
    await screen.findByText('9 из 9');
  });

  it('supplier confirms exact requested terms and quantities using the current revision', async () => {
    const exact = { terms: { unitPrice: { amountMinor: '12050', currency: 'USD' }, expiresAt: '2030-07-01T00:00:00Z' }, lines: [line] };
    let value = reservation('requested', ['confirm'], { requested: exact, requestedDigest: 'same' });
    const fetcher = fetcherFor('distributor', { '/reservations': (url, init) => {
      if (init?.method === 'POST') { value = reservation('active', [], { requested: exact, offered: exact, current: exact, requestedDigest: 'same', offeredDigest: 'same', currentDigest: 'same' }); return response({ data: value, revision: '9007199254740994' }); }
      return url.endsWith('/reservation-1') ? response({ data: value, revision: value.revision }) : response({ items: [value] });
    } });
    mount(fetcher);
    fireEvent.click(await screen.findByRole('button', { name: 'Открыть' }));
    await screen.findByRole('button', { name: 'Отправить подтверждение' });
    fireEvent.click(screen.getByRole('button', { name: 'Отправить подтверждение' }));
    await waitFor(() => expect(call(fetcher, '/supplier-confirmations', 'POST')).toBeTruthy());
    const sent = call(fetcher, '/supplier-confirmations', 'POST')!;
    expect((sent.headers as Headers).get('If-Match')).toBe('"9007199254740993"');
    expect(JSON.parse(String(sent.body))).toEqual({ terms: { ...exact.terms, expiresAt: '2030-07-01T00:00:00.000Z' }, lines: [{ lineId: 'line-1', quantity: '1' }] });
    await screen.findByText(/статус: active/);
    expect(screen.getByText('Согласовано')).toBeTruthy();
  });

  it('supplier quantity change produces an offer without activating or hiding the requested snapshot', async () => {
    const futureLine = { ...line, vehicleId: undefined, supplyLotId: 'lot-1', quantity: '2' };
    const original = { terms: {}, lines: [futureLine] };
    const changed = { terms: { unitPrice: { amountMinor: '10001', currency: 'USD' }, expiresAt: '2030-07-01T00:00:00Z' },
      lines: [{ ...futureLine, quantity: '3' }] };
    let value = reservation('requested', ['confirm'], { requested: original });
    const fetcher = fetcherFor('distributor', { '/reservations': (url, init) => {
      if (init?.method === 'POST') {
        value = reservation('offered', [], { requested: original, offered: changed, offeredDigest: 'changed-digest' });
        return response({ data: value, revision: '9007199254740994' });
      }
      return url.endsWith('/reservation-1') ? response({ data: value, revision: value.revision }) : response({ items: [value] });
    } });
    mount(fetcher);
    fireEvent.click(await screen.findByRole('button', { name: 'Открыть' }));
    await screen.findByRole('button', { name: 'Отправить подтверждение' });
    fireEvent.change(screen.getByLabelText('Цена за единицу'), { target: { value: '100.01' } });
    fireEvent.change(screen.getByLabelText('Срок действия'), { target: { value: '2030-07-01T00:00' } });
    fireEvent.change(screen.getByLabelText('Количество line-1'), { target: { value: '3' } });
    fireEvent.click(screen.getByRole('button', { name: 'Отправить подтверждение' }));
    await waitFor(() => expect(call(fetcher, '/supplier-confirmations', 'POST')).toBeTruthy());
    expect(JSON.parse(String(call(fetcher, '/supplier-confirmations', 'POST')!.body)).lines).toEqual([{ lineId: 'line-1', quantity: '3' }]);
    await screen.findByText(/статус: offered/);
    expect(screen.getByText('Запрошено')).toBeTruthy();
    expect(screen.getByText('Предложено поставщиком')).toBeTruthy();
    expect(screen.queryByText('Согласовано')).toBeNull();
  });

  it('shows changed offered and immutable requested snapshots, then accepts only the exact offered digest', async () => {
    let value = reservation('offered', ['accept'], { offered, offeredDigest: 'offer-digest' });
    const fetcher = fetcherFor('dealer', { '/reservations': (url, init) => {
      if (init?.method === 'POST') { value = reservation('active', [], { offered, current: offered, offeredDigest: 'offer-digest', currentDigest: 'offer-digest' }); return response({ data: value, revision: '9007199254740994' }); }
      return url.endsWith('/reservation-1') ? response({ data: value, revision: value.revision }) : response({ items: [value] });
    } });
    mount(fetcher);
    fireEvent.click(await screen.findByRole('button', { name: 'Открыть' }));
    await screen.findByText('Предложено поставщиком');
    expect(screen.getByText('Запрошено')).toBeTruthy();
    expect(screen.queryByLabelText('Цена за единицу')).toBeNull();
    fireEvent.click(screen.getByRole('button', { name: 'Принять предложенные условия' }));
    await waitFor(() => expect(call(fetcher, '/acceptances', 'POST')).toBeTruthy());
    const sent = call(fetcher, '/acceptances', 'POST')!;
    expect(JSON.parse(String(sent.body))).toEqual({ digest: 'offer-digest' });
    expect((sent.headers as Headers).get('If-Match')).toBe('"9007199254740993"');
    await screen.findByText('Согласовано');
    expect(screen.queryByRole('button', { name: 'Принять предложенные условия' })).toBeNull();
  });

  it('blocks wrong party, missing permission and unverified mutations while preserving readable history', async () => {
    const value = reservation('requested', ['confirm']);
    const fetcher = fetcherFor('dealer', { '/reservations': url => url.endsWith('/reservation-1')
      ? response({ data: value, revision: value.revision }) : response({ items: [value] }) }, 'submitted', false);
    mount(fetcher);
    expect(await screen.findByRole('button', { name: 'Запросить резерв' })).toHaveProperty('disabled', true);
    fireEvent.click(await screen.findByRole('button', { name: 'Открыть' }));
    await screen.findByText('Запрошено');
    expect(screen.queryByRole('button', { name: 'Отправить подтверждение' })).toBeNull();
    expect(fetcher.mock.calls.filter(c => c[1]?.method === 'POST')).toHaveLength(0);
  });

  it('shows stock read errors, permits retry and keeps request disabled until a real source is selected', async () => {
    let attempts = 0;
    const fetcher = fetcherFor('dealer', { '/reservation-stock': () => {
      attempts++;
      return attempts === 1 ? response({ error: { code: 'temporary_failure', message: 'Stock unavailable' } }, 503)
        : response({ items: [stock], nextCursor: null });
    } });
    mount(fetcher); await chooseSupplier();
    await screen.findByText(/Stock unavailable/);
    expect(screen.getByLabelText('VIN')).toHaveProperty('disabled', true);
    fireEvent.click(screen.getByRole('button', { name: 'Повторить' }));
    const vin = screen.getByLabelText('VIN') as HTMLSelectElement;
    await within(vin).findByRole('option', { name: /VIN-1/ });
    expect(attempts).toBe(2);
    expect(fetcher.mock.calls.filter(c => c[1]?.method === 'POST')).toHaveLength(0);
  });

  it('keeps selected reservation and refreshes after a stale confirmation; duplicate clicks send once', async () => {
    let getCount = 0; let release!: (value: Response) => void;
    const pending = new Promise<Response>(resolve => { release = resolve; });
    const value = reservation('requested', ['confirm']);
    const fetcher = fetcherFor('distributor', { '/reservations': (url, init) => {
      if (init?.method === 'POST') return pending;
      if (url.endsWith('/reservation-1')) { getCount++; return response({ data: value, revision: value.revision }); }
      return response({ items: [value] });
    } });
    mount(fetcher);
    fireEvent.click(await screen.findByRole('button', { name: 'Открыть' }));
    await screen.findByRole('button', { name: 'Отправить подтверждение' });
    fireEvent.change(screen.getByLabelText('Цена за единицу'), { target: { value: '120.50' } });
    fireEvent.change(screen.getByLabelText('Срок действия'), { target: { value: '2030-07-01T00:00' } });
    fireEvent.click(screen.getByRole('button', { name: 'Отправить подтверждение' }));
    fireEvent.click(screen.getByRole('button', { name: 'Сохранение…' }));
    expect(fetcher.mock.calls.filter(c => String(c[0]).includes('/supplier-confirmations') && c[1]?.method === 'POST')).toHaveLength(1);
    release(response({ error: { code: 'stale', message: 'stale revision' } }, 409));
    await screen.findByText(/stale revision/);
    await waitFor(() => expect(getCount).toBeGreaterThan(1));
    expect(screen.getByText('Запрошено')).toBeTruthy();
  });

  it('keeps Purchases order tabs and creation while opening and closing the reservation workspace by keyboard', async () => {
    const fetcher = fetcherFor('dealer');
    const { onClose } = mount(fetcher, true);
    expect(await screen.findByRole('button', { name: 'Создать заказ' })).toBeTruthy();
    expect(screen.getByText('Мои заказы')).toBeTruthy();
    expect(screen.getByText('Полученные заказы')).toBeTruthy();
    fireEvent.click(screen.getByRole('button', { name: 'Резервы и поставки' }));
    await screen.findByRole('dialog', { name: 'Резервы и поставки' });
    fireEvent.keyDown(document, { key: 'Escape' });
    await waitFor(() => expect(screen.queryByRole('dialog', { name: 'Резервы и поставки' })).toBeNull());
    expect(screen.getByRole('button', { name: 'Создать заказ' })).toBeTruthy();
    expect(onClose).not.toHaveBeenCalled();
  });
});
