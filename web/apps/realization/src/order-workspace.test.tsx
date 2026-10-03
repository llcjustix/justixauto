// @vitest-environment jsdom
import { QueryClient, QueryClientProvider } from '@tanstack/react-query';
import { cleanup, fireEvent, render, screen, waitFor } from '@testing-library/react';
import { afterEach, expect, it, vi } from 'vitest';
import { OrderDialog } from './pages/trade';
import { nextOrderAction } from './order-workspace';
import type { Order } from './data';

const money = { amountMinor: '10000', currency: 'USD' };
const terms = { lines: [{ lineId: 'line-1', modelId: 'model-1', quantity: '5', unitPrice: money }], route: 'local', deliveryTerms: '', warrantyTerms: '', serviceTerms: '', paymentSchedule: [] };
// A batch received with known colours: VIN entry then needs no colour choice (receipt-colors.test covers unknown ones).
const receipt = { receiptBatchId: 'batch-1', warehouseId: 'private-warehouse', modelId: 'model-1', orderLineId: 'line-1', shipmentId: 'shipment-1', shippedQuantity: '4', confirmedQuantity: '3', identifiedCount: '1', unidentifiedCount: '2', revision: '2', modelSpecificationVersion: '1', exteriorColor: 'White', interiorColor: 'Black' };
const base: Order = { id: 'order-1', party: 'buyer', buyer: { name: 'Buyer' }, supplier: { name: 'Supplier' }, source: 'direct', terms, total: money, status: 'fulfilling', statusReason: '', revision: '7', updatedAt: '', allocations: [], shipments: [{ id: 'shipment-1', route: 'local', status: 'received' }], addenda: [], allowedActions: [], receiptBatches: [receipt], receivingWarehouseId: 'private-warehouse', hasReceivingWarehouse: true,
  lineProgress: [{ orderLineId: 'line-1', allocated: '1', shipped: '4', identified: '1', unidentified: '2', receiptQuantityAdjusted: true }], history: [{ type: 'order.shipped_delivered', occurredAt: '2026-10-01T01:00:00Z', reason: 'Receipt saved' }] };
const json = (body: unknown, status = 200) => new Response(JSON.stringify(body), { status, headers: { 'Content-Type': 'application/json' } });
afterEach(() => { cleanup(); vi.unstubAllGlobals(); });

function setup(options: { order?: Order; invoiceError?: boolean; invoices?: unknown[]; legacy?: boolean; orderError?: boolean } = {}) {
  let invoiceError = options.invoiceError ?? false;
  let shown = options.order ?? base;
  const posts: { url: string; body: unknown; revision: string | null }[] = [];
  const reads: string[] = [];
  vi.stubGlobal('fetch', vi.fn(async (input: RequestInfo | URL, init: RequestInit = {}) => {
    const url = String(input);
    if (init.method === 'POST') {
      posts.push({ url, body: JSON.parse(String(init.body)), revision: new Headers(init.headers).get('If-Match') });
      if (url.endsWith('/identifications')) shown = { ...shown, receiptBatches: [{ ...receipt, identifiedCount: '2', unidentifiedCount: '1' }], lineProgress: [{ ...shown.lineProgress![0]!, identified: '2', unidentified: '1' }] };
      return json({ data: {} }, 201);
    }
    reads.push(url);
    if (url.includes('/orders/order-1/invoices')) return invoiceError ? json({ message: 'Invoice unavailable' }, 503) : json({ items: options.invoices ?? [] });
    if (url.endsWith('/orders/order-1')) return options.orderError ? json({ message: 'Order unavailable' }, 503) : json({ data: shown });
    if (url.includes('/vehicle-models')) return json({ items: [{ id: 'model-1', specification: { version: '2', make: 'Make', model: 'Model', variant: 'Trim', exteriorColor: 'Current red', interiorColor: 'Current grey', exteriorColors: ['Current red'], interiorColors: ['Current grey'] } }] });
    if (url.endsWith('/warehouses/private-warehouse/inventory')) return json({ data: { unidentifiedBatches: [{ ...receipt, id: 'batch-1', ...shown.receiptBatches?.[0] }] } });
    if (url.includes('/warehouses')) return json({ items: [{ id: 'private-warehouse', name: 'Private destination', free: '50' }] });
    if (url.includes('/shipments/shipment-1')) return json({ data: { id: 'shipment-1', status: options.legacy ? 'shipped' : 'received', route: 'local', revision: '9', vehicles: [{ vehicleId: 'v1', vin: 'LGXC16DF0P0000001', status: options.legacy ? 'shipped' : 'delivered' }], milestones: [] } });
    return json({ items: [] });
  }));
  const client = new QueryClient({ defaultOptions: { queries: { retry: false } } });
  render(<QueryClientProvider client={client}><OrderDialog id="order-1" onClose={vi.fn()} /></QueryClientProvider>);
  return { posts, reads, invoiceRetry: () => { invoiceError = false; } };
}
const tab = async (name: string) => fireEvent.click(await screen.findByRole('tab', { name }));
const oneModal = () => expect(screen.getAllByRole('dialog')).toHaveLength(1);

it('shows truthful receipt adjustment and partial VIN completion with fresh order progress in one modal', async () => {
  const run = setup();
  expect(await screen.findByText(/Исторически отгруженное количество сохранено/)).toBeTruthy();
  fireEvent.click(screen.getByRole('button', { name: 'Ввести VIN' }));
  fireEvent.change(await screen.findByLabelText('VIN 1'), { target: { value: 'lgxc16df0p0000002' } });
  oneModal();
  fireEvent.click(screen.getByRole('button', { name: 'Сохранить VIN' }));
  expect(await screen.findByText(/Осталось без VIN: 1/)).toBeTruthy();
  expect(await screen.findByText(/с VIN 2, без VIN 1/)).toBeTruthy();
  expect(run.posts[0]?.body).toEqual({ atomic: true, items: [{ vin: 'LGXC16DF0P0000002', modelId: 'model-1' }] });
  oneModal();
});

it('hides receipt navigation and destination details from suppliers even if supplied accidentally', async () => {
  setup({ order: { ...base, party: 'supplier', allowedActions: ['ship-quantity'] } });
  await screen.findByRole('tab', { name: 'Оплаты' });
  expect(screen.queryByText(/Private destination|private-warehouse|batch-1/)).toBeNull();
  expect(screen.queryByRole('button', { name: 'Ввести VIN' })).toBeNull();
  expect(screen.getByText(/Ваш следующий шаг: подготовьте/)).toBeTruthy();
  await tab('История');
  expect(screen.getByText(/Receipt saved/)).toBeTruthy();
});

it('distinguishes invoice errors from empty state and restores the payment tab after a payment form', async () => {
  const invoice = { id: 'invoice-1', total: money, paid: money, outstanding: money, status: 'issued', allowedActions: ['submit-payment'], revision: '3', paymentEvidence: [] };
  const run = setup({ invoiceError: true, invoices: [invoice] });
  await tab('Оплаты');
  await screen.findByRole('button', { name: 'Повторить загрузку счетов' });
  expect(screen.queryByText('Счета пока не выставлены.')).toBeNull();
  run.invoiceRetry();
  fireEvent.click(screen.getByRole('button', { name: 'Повторить загрузку счетов' }));
  fireEvent.click(await screen.findByRole('button', { name: 'Сообщить об оплате' }));
  oneModal();
  fireEvent.change(screen.getByLabelText(/Сумма/), { target: { value: '10.25' } });
  fireEvent.change(screen.getByLabelText(/Дата оплаты/), { target: { value: '2026-10-01' } });
  fireEvent.change(screen.getByLabelText(/Номер платёжки/), { target: { value: 'PAY-1' } });
  fireEvent.click(screen.getByRole('button', { name: 'Сообщить об оплате' }));
  await screen.findByText('Сообщить об оплате: выполнено.');
  expect(run.posts[0]?.body).toEqual({ claimedAmount: { amountMinor: '1025', currency: 'USD' }, paidOn: '2026-10-01', externalReference: 'PAY-1', attachmentBindingIds: [] });
  expect(screen.getByRole('tab', { name: 'Оплаты' }).getAttribute('aria-selected')).toBe('true');
  oneModal();
});

it('keeps addendum decisions and terms proposal reachable without stacking and preserves changes tab', async () => {
  const addendum = { id: 'a1', number: 1, terms, total: money, reason: 'New price', proposedBy: 'supplier', status: 'proposed', decisionReason: '' };
  const run = setup({ order: { ...base, addenda: [addendum], allowedActions: ['accept-addendum', 'reject-addendum', 'propose-addendum'] } });
  await tab('Изменения');
  fireEvent.click(screen.getByRole('button', { name: 'Принять изменение №1' }));
  oneModal();
  fireEvent.click(screen.getByRole('button', { name: 'Принять изменение №1' }));
  await screen.findByText('Принять изменение №1: выполнено.');
  expect(run.posts[0]).toEqual({ url: '/api/v1/commerce/orders/order-1/addenda/a1/accept', body: {}, revision: '"7"' });
  fireEvent.click(screen.getByRole('button', { name: 'Предложить изменение' }));
  oneModal();
  fireEvent.click(screen.getByRole('button', { name: 'Назад' }));
  expect(screen.getByRole('tab', { name: 'Изменения' }).getAttribute('aria-selected')).toBe('true');
});

it('keeps legacy receipt and milestones in one modal and returns to the originating shipment', async () => {
  const run = setup({ legacy: true, order: { ...base, receivingWarehouseId: null, hasReceivingWarehouse: false, receiptBatches: [], shipments: [{ id: 'shipment-1', route: 'local', status: 'shipped' }] } });
  fireEvent.click(await screen.findByRole('button', { name: 'Отгрузка shipment' }));
  fireEvent.click(await screen.findByRole('button', { name: 'Принять на склад' }));
  oneModal();
  fireEvent.click(screen.getByLabelText('LGXC16DF0P0000001'));
  // The warehouse is an autocomplete: pick the option, a typed id alone commits nothing.
  fireEvent.focus(screen.getByRole('combobox', { name: /Склад/ }));
  fireEvent.click(await screen.findByRole('option', { name: /Private destination/ }));
  fireEvent.click(screen.getByRole('button', { name: 'Принять на склад' }));
  await screen.findByText('Принять на склад: выполнено.');
  expect(run.posts[0]).toEqual({ url: '/api/v1/commerce/shipments/shipment-1/receipt-decisions', body: { decision: 'accept', vehicleIds: ['v1'], warehouseId: 'private-warehouse' }, revision: '"9"' });
  fireEvent.click(await screen.findByRole('button', { name: 'Отметить этап' }));
  oneModal();
  fireEvent.click(screen.getByRole('button', { name: 'Назад' }));
  expect(screen.getByRole('button', { name: 'Назад к заказу' })).toBeTruthy();
  oneModal();
});

it('shows completed transfer without a second receipt step and contextual empty tabs', async () => {
  setup({ order: { ...base, receiptBatches: [], history: [] } });
  await tab('Оплаты');
  expect(await screen.findByText('Счета пока не выставлены.')).toBeTruthy();
  await tab('Изменения');
  expect(screen.getByText('Изменений условий пока нет.')).toBeTruthy();
  await tab('История');
  expect(screen.getByText('Событий пока нет.')).toBeTruthy();
  await tab('Автомобили и исполнение');
  fireEvent.click(screen.getByRole('button', { name: 'Отгрузка shipment' }));
  expect(await screen.findByText(/повторная приёмка не требуется/)).toBeTruthy();
  expect(screen.queryByRole('button', { name: 'Принять на склад' })).toBeNull();
  expect(screen.queryByRole('button', { name: 'Отказать в приёмке' })).toBeNull();
});

it('offers retry for a failed order instead of an endless loader', async () => {
  setup({ orderError: true });
  expect(await screen.findByRole('button', { name: 'Повторить загрузку заказа' })).toBeTruthy();
  expect(screen.queryByText('Загрузка заказа…')).toBeNull();
});

it('attributes next actions to the correct party, including own pending changes and buyer VINs', () => {
  const pending = { id: 'a', number: 1, terms, total: money, status: 'proposed', proposedBy: 'buyer', reason: '', decisionReason: '' };
  const own = { ...base, receiptBatches: [], addenda: [pending], allowedActions: ['propose-addendum'] };
  expect(nextOrderAction(own).needsMe).toBe(false);
  expect(nextOrderAction(own).message).toContain('Ждём поставщика');
  expect(nextOrderAction({ ...own, party: 'supplier', allowedActions: ['accept-addendum'] }).label).toBe('Рассмотреть изменение');
  expect(nextOrderAction(base).label).toBe('Ввести VIN');
  expect(nextOrderAction({ ...base, party: 'supplier', allowedActions: ['ship-quantity'] }).needsMe).toBe(true);
});

it('colors workspace distinguishes saved combinations and actual batches independently of current catalogue', async () => {
  const lines = [
    { ...terms.lines[0]!, modelSpecificationVersion: '1', exteriorColor: 'White', interiorColor: 'Black' },
    { ...terms.lines[0]!, lineId: 'line-2', modelSpecificationVersion: '1', exteriorColor: 'Blue', interiorColor: 'Tan' },
  ];
  const run = setup({ order: { ...base, terms: { ...terms, lines }, allowedActions: ['allocate'], receiptBatches: [
    { ...receipt, modelSpecificationVersion: '3', exteriorColor: 'Actual silver', interiorColor: '' },
  ] } });
  await screen.findByRole('columnheader', { name: 'Модель и заказанные цвета' });
  await waitFor(() => expect(screen.getAllByText(/Make Model Trim · Кузов: White · Салон: Black · Версия: 1/).length).toBeGreaterThan(0));
  expect(screen.getAllByText(/Make Model Trim · Кузов: Blue · Салон: Tan · Версия: 1/).length).toBeGreaterThan(0);
  expect(screen.getByText('Фактические цвета партии · Кузов: Actual silver · Салон: Не указан · Версия: 3')).toBeTruthy();
  expect(screen.queryByText(/Current red|Current grey/)).toBeNull();
  const buttons = screen.getAllByRole('button', { name: /Назначить VIN —/ });
  expect(buttons.some(button => button.textContent?.includes('White'))).toBe(true);
  expect(buttons.some(button => button.textContent?.includes('Blue'))).toBe(true);
  for (const name of ['Оплаты', 'Изменения', 'История', 'Автомобили и исполнение']) await tab(name);
  oneModal();
  expect(run.posts).toHaveLength(0);
});

it('colors unrelated terms edits preserve line identity provenance and known or unknown history', async () => {
  const known = { ...terms.lines[0]!, offerLineId: 'source-1', modelSpecificationVersion: '1', exteriorColor: 'White', interiorColor: 'Black' };
  const unknown = { ...terms.lines[0]!, lineId: 'line-2' };
  const run = setup({ order: { ...base, terms: { ...terms, lines: [known, unknown] }, allowedActions: ['propose-addendum'] } });
  await tab('Изменения');
  fireEvent.click(screen.getByRole('button', { name: 'Предложить изменение' }));
  await waitFor(() => expect((screen.getByLabelText('Модель 1') as HTMLInputElement).value).toBe('Make Model Trim'));
  expect(screen.getByText('Кузов: White · Салон: Black · Версия: 1')).toBeTruthy();
  expect(screen.getByText('Кузов: Не указан · Салон: Не указан · Версия: Не указан')).toBeTruthy();
  fireEvent.change(screen.getByLabelText(/Причина изменения/), { target: { value: 'Warranty only' } });
  fireEvent.change(screen.getByLabelText('Гарантия'), { target: { value: 'Updated warranty' } });
  fireEvent.click(screen.getByRole('button', { name: 'Сохранить' }));
  await screen.findByText('Изменение предложено. Ждём решения контрагента.');
  expect(run.posts[0]?.body).toEqual({ terms: { ...terms, lines: [known, unknown], warrantyTerms: 'Updated warranty' }, reason: 'Warranty only' });
  expect(screen.getByRole('tab', { name: 'Изменения' }).getAttribute('aria-selected')).toBe('true');
  oneModal();
});
