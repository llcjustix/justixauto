// @vitest-environment jsdom
import { createElement } from 'react';
import { QueryClient, QueryClientProvider } from '@tanstack/react-query';
import { cleanup, fireEvent, render, screen, waitFor, within } from '@testing-library/react';
import { MemoryRouter } from 'react-router-dom';
import { afterEach, describe, expect, it, vi } from 'vitest';
import { PurchasesPage } from './pages/purchases';

afterEach(() => {
  cleanup();
  vi.unstubAllGlobals();
});

const order = (id: string, party: 'buyer' | 'supplier', supplier: string, status = 'awaiting-supplier') => ({
  id,
  party,
  buyer: { name: party === 'buyer' ? 'Test Company' : 'Buyer Co' },
  supplier: { name: supplier },
  source: 'direct',
  terms: { lines: [] },
  total: { amountMinor: '100000', currency: 'USD' },
  status,
  statusReason: '',
  allocations: [],
  shipments: [],
  addenda: [],
  allowedActions: [],
  revision: '1',
  updatedAt: '2026-09-27T15:34:09Z',
});

const partner = {
  id: 'p-1',
  direction: 'outgoing',
  counterparty: { id: 'c-2', name: 'My Company 1', country: 'Узбекистан' },
  status: 'active',
  statusReason: '',
  allowedActions: [],
};
const warehouse = { id: 'w-1', branchId: null, name: 'Главный склад', capacity: '100', occupied: '10', free: '90', revision: '1' };
const model = {
  id: 'm-1',
  specification: { make: 'Chevrolet', model: 'Cobalt', variant: 'LTZ', year: 2025, version: '1', exteriorColor: 'White', interiorColor: 'Black' },
  revision: '1',
};
const promotion = {
  id: 'offer-1', supplier: { id: 'c-2', name: 'Promotion supplier' }, status: 'published',
  publishedVersion: { id: 'v-1', number: 1, total: { amountMinor: '100', currency: 'USD' }, terms: {
    lines: [{ lineId: 'l-1', modelId: 'm-1', quantity: '1', unitPrice: { amountMinor: '100', currency: 'USD' } }],
    route: 'local', paymentSchedule: [], deliveryTerms: '', warrantyTerms: '', serviceTerms: '',
  } },
};

function renderPage(
  onPost?: (url: string, body: unknown) => void,
  orders?: unknown[],
  offers: unknown[] = [promotion],
  detailOffer: unknown = promotion,
  catalogMode: 'success' | 'loading' | 'error' = 'success',
) {
  vi.stubGlobal(
    'fetch',
    vi.fn(async (input: RequestInfo | URL, init?: RequestInit) => {
      const url = String(input);
      if (init?.method === 'POST') {
        onPost?.(url, JSON.parse(String(init.body)));
        return new Response(JSON.stringify({ data: order('o-9', 'buyer', 'My Company 1'), revision: '1' }), {
          status: 201,
          headers: { 'Content-Type': 'application/json' },
        });
      }
      if (url.includes('/commerce/rfqs')) throw new Error('purchase requests are retired');
      if (url.endsWith('/inventory/vehicle-models/m-1')) return new Response(JSON.stringify({ data: { ...model, versions: [model.specification] } }), { headers: { 'Content-Type': 'application/json' } });
      if (/\/commerce\/offers\/[^?]+/.test(url)) return new Response(JSON.stringify({ data: detailOffer }), { headers: { 'Content-Type': 'application/json' } });
      if (url.includes('/commerce/orders/o-9')) return new Response(JSON.stringify({ data: { ...order('o-9', 'buyer', 'My Company 1'), terms: { lines: [], route: 'local', paymentSchedule: [], deliveryTerms: '', warrantyTerms: '', serviceTerms: '' } } }), { headers: { 'Content-Type': 'application/json' } });
      const items = url.includes('/commerce/orders')
        ? orders ?? [
            order('o-1', 'buyer', 'My Company 1'),
            order('o-2', 'supplier', 'Test Company'),
            order('o-3', 'buyer', 'Closed Supplier', 'completed'),
          ]
        : url.includes('/commerce/partnerships')
          ? [partner]
          : url.includes('/inventory/warehouses')
            ? [warehouse]
          : url.includes('/inventory/vehicle-models')
            ? [model]
            : url.includes('/commerce/offers') ? offers : [];
      if (url.includes('/commerce/offers') && catalogMode === 'loading') return new Promise<Response>(() => {});
      if (url.includes('/commerce/offers') && catalogMode === 'error') return new Response(JSON.stringify({ message: 'catalog unavailable' }), { status: 500, headers: { 'Content-Type': 'application/json' } });
      return new Response(JSON.stringify({ items }), { status: 200, headers: { 'Content-Type': 'application/json' } });
    }),
  );
  const client = new QueryClient({ defaultOptions: { queries: { retry: false } } });
  render(
    createElement(QueryClientProvider, { client }, createElement(MemoryRouter, null, createElement(PurchasesPage))),
  );
}

async function pick(label: string, name: string | RegExp) {
  screen.getAllByRole('combobox').forEach(input => fireEvent.blur(input));
  fireEvent.focus(screen.getByRole('combobox', { name: label, exact: true }));
  fireEvent.click(await screen.findByRole('option', { name, exact: true }));
}

describe('purchases', () => {
  it('lists only our orders and offers no quotation requests', async () => {
    renderPage();
    expect(await screen.findByText('My Company 1')).toBeTruthy();
    expect(screen.queryByText('Buyer Co')).toBeNull();
    expect(screen.queryByRole('button', { name: 'Создать запрос' })).toBeNull();
    expect(screen.getByText(/Заказ o-1/)).toBeTruthy();
    expect(screen.getAllByText(/отгружено 0 из 0/).length).toBeGreaterThan(0);
  });

  it('does not count our own proposed addendum as a decision and labels buyer VIN and supplier shipping actions', async () => {
    renderPage(undefined, [
      { ...order('own', 'buyer', 'Own proposal', 'accepted'), addenda: [{ status: 'proposed', proposedBy: 'buyer' }], allowedActions: ['propose-addendum'] },
      { ...order('vin', 'buyer', 'VIN supplier', 'completed'), receiptBatches: [{ unidentifiedCount: '2' }] },
      { ...order('ship', 'supplier', 'Our company', 'accepted'), allowedActions: ['ship-quantity'] },
    ]);
    await screen.findByText('Own proposal');
    const ownRow = screen.getByText('Own proposal').closest('tr')!;
    expect(within(ownRow).getByRole('button', { name: 'Открыть' })).toBeTruthy();
    expect(screen.getByRole('button', { name: 'Ввести VIN' })).toBeTruthy();
    fireEvent.click(screen.getByRole('tab', { name: /^Полученные заказы/ }));
    expect(screen.getByRole('button', { name: 'Продолжить отгрузку' })).toBeTruthy();
  });

  it('shows received orders in their own tab, with open orders counted on both tabs', async () => {
    renderPage();
    await screen.findByText('My Company 1');
    // Two orders sent, one of them completed; one open order received.
    expect(screen.getByRole('tab', { name: /^Мои заказы\s*1$/ })).toBeTruthy();
    fireEvent.click(screen.getByRole('tab', { name: /^Полученные заказы\s*1$/ }));
    expect(await screen.findByText('Buyer Co')).toBeTruthy();
    expect(screen.queryByText('My Company 1')).toBeNull();
  });

  it('has no finished tab: closed orders are listed and found by the status filter', async () => {
    renderPage();
    expect(await screen.findByText('Closed Supplier')).toBeTruthy();
    expect(screen.queryByRole('tab', { name: /Завершённые/ })).toBeNull();
    fireEvent.focus(screen.getByDisplayValue('Все статусы'));
    fireEvent.click(screen.getByRole('option', { name: 'Выполнен', exact: true }));
    await waitFor(() => expect(screen.queryByText('My Company 1')).toBeNull());
    expect(screen.getByText('Closed Supplier')).toBeTruthy();
  });

  it('creates a direct order only after its three-step review', async () => {
    let posted: { url: string; body: unknown } | undefined;
    renderPage((url, body) => (posted = { url, body }));
    await screen.findByText('My Company 1');

    fireEvent.click(screen.getByRole('button', { name: 'Создать заказ' }));
    const dialog = await screen.findByRole('dialog', { name: 'Новый заказ поставщику' });
    await pick('Поставщик', 'My Company 1');
    await pick('Склад получения', /Главный склад/);
    fireEvent.click(within(dialog).getByRole('button', { name: 'Далее' }));
    await pick('Модель 1', /Chevrolet Cobalt/);
    await waitFor(() => expect((screen.getByLabelText('Цвет кузова 1') as HTMLInputElement).disabled).toBe(false));
    fireEvent.change(within(dialog).getByLabelText('Количество'), { target: { value: '50' } });
    fireEvent.change(within(dialog).getByLabelText('Цена'), { target: { value: '25000' } });
    fireEvent.click(within(dialog).getByRole('button', { name: 'Далее' }));
    expect(within(dialog).getByText('1 250 000.00 USD')).toBeTruthy();
    fireEvent.click(within(dialog).getByRole('button', { name: 'Отправить заказ' }));

    await waitFor(() => expect(posted).toBeDefined());
    expect(posted!.url).toMatch(/\/commerce\/orders$/);
    expect(posted!.body).toMatchObject({
      supplierCompanyId: 'c-2',
      warehouseId: 'w-1',
      terms: { lines: [{ modelId: 'm-1', quantity: '50', unitPrice: { amountMinor: '2500000', currency: 'USD' } }] },
    });
    expect(await screen.findByText('Заказ создан и отправлен поставщику на подтверждение.')).toBeTruthy();
    await waitFor(() => expect(screen.queryByRole('dialog', { name: 'Новый заказ поставщику' })).toBeNull());
    expect(screen.getAllByRole('dialog')).toHaveLength(1);
  });

  it('keeps a draft and identifies the missing first-step field', async () => {
    let posted = false;
    renderPage(() => (posted = true));
    await screen.findByText('My Company 1');
    fireEvent.click(screen.getByRole('button', { name: 'Создать заказ' }));
    const dialog = await screen.findByRole('dialog', { name: 'Новый заказ поставщику' });
    fireEvent.click(within(dialog).getByRole('button', { name: 'Далее' }));
    expect(await within(dialog).findByText('Выберите склад получения')).toBeTruthy();
    expect(posted).toBe(false);
  });

  it('replaces catalog and offer with one creation modal and opens the resulting promotion order', async () => {
    const posted = vi.fn();
    renderPage(posted);
    await screen.findByText('My Company 1');
    fireEvent.click(screen.getByRole('button', { name: 'Предложения от поставщиков' }));
    const catalog = await screen.findByRole('dialog', { name: 'Предложения от поставщиков' });
    fireEvent.click(await within(catalog).findByRole('button', { name: 'Открыть' }));
    await screen.findByRole('dialog', { name: 'Предложение Promotion supplier' });
    expect(screen.getAllByRole('dialog')).toHaveLength(1);
    fireEvent.click(screen.getByRole('button', { name: 'Заказать' }));
    const form = await screen.findByRole('dialog', { name: 'Заказ по акции' });
    expect(screen.getAllByRole('dialog')).toHaveLength(1);
    await pick('Склад получения', /Главный склад/);
    fireEvent.click(within(form).getByRole('button', { name: 'Далее' }));
    fireEvent.change(within(form).getByLabelText('Количество 1'), { target: { value: '2' } });
    await waitFor(() => expect((screen.getByLabelText('Цвет кузова 1') as HTMLInputElement).disabled).toBe(false));
    fireEvent.click(within(form).getByRole('button', { name: 'Далее' }));
    fireEvent.click(within(form).getByRole('button', { name: 'Отправить заказ' }));
    expect(await screen.findByText('Заказ создан и отправлен поставщику на подтверждение.')).toBeTruthy();
    await waitFor(() => expect(screen.queryByRole('dialog', { name: 'Заказ по акции' })).toBeNull());
    expect(screen.getAllByRole('dialog')).toHaveLength(1);
    expect(posted).toHaveBeenCalledWith(expect.stringContaining('/commerce/orders'), { offerVersionId: 'v-1', warehouseId: 'w-1', lines: [{ offerLineId: 'l-1', quantity: '2', modelSpecificationVersion: '1', exteriorColor: 'White', interiorColor: 'Black' }] });
  });
});

describe('supplier offer catalog', () => {
  it('creates an order directly from an eligible row through review with the captured offer payload', async () => {
    const posted = vi.fn();
    renderPage(posted);
    await screen.findByText('My Company 1');

    fireEvent.click(screen.getByRole('button', { name: 'Предложения от поставщиков' }));
    const catalog = await screen.findByRole('dialog', { name: 'Предложения от поставщиков' });
    const row = await within(catalog).findByText('Promotion supplier');
    fireEvent.click(within(row.closest('tr')!).getByRole('button', { name: 'Создать заказ' }));
    const form = await screen.findByRole('dialog', { name: 'Заказ по акции' });
    expect(screen.getAllByRole('dialog')).toHaveLength(1);
    expect(posted).not.toHaveBeenCalled();
    await pick('Склад получения', /Главный склад/);
    fireEvent.click(within(form).getByRole('button', { name: 'Далее' }));
    fireEvent.change(within(form).getByLabelText('Количество 1'), { target: { value: '2' } });
    await waitFor(() => expect((screen.getByLabelText('Цвет кузова 1') as HTMLInputElement).disabled).toBe(false));
    fireEvent.click(within(form).getByRole('button', { name: 'Далее' }));
    fireEvent.click(within(form).getByRole('button', { name: 'Отправить заказ' }));
    await waitFor(() => expect(posted).toHaveBeenCalledTimes(1));
    expect(posted).toHaveBeenCalledWith(expect.stringContaining('/commerce/orders'), { offerVersionId: 'v-1', warehouseId: 'w-1', lines: [{ offerLineId: 'l-1', quantity: '2', modelSpecificationVersion: '1', exteriorColor: 'White', interiorColor: 'Black' }] });
  });

  it.each([
    ['draft', 'draft', promotion.publishedVersion],
    ['withdrawn', 'withdrawn', promotion.publishedVersion],
    ['missing version', 'published', null],
  ])('does not offer direct creation for %s offers', async (_name, status, publishedVersion) => {
    renderPage(undefined, undefined, [{ ...promotion, status, publishedVersion }]);
    await screen.findByText('My Company 1');
    fireEvent.click(screen.getByRole('button', { name: 'Предложения от поставщиков' }));
    const catalog = await screen.findByRole('dialog', { name: 'Предложения от поставщиков' });
    const row = await within(catalog).findByText('Promotion supplier');
    expect(within(row.closest('tr')!).queryByRole('button', { name: 'Создать заказ' })).toBeNull();
  });

  it('keeps an invalid detail response open instead of creating an order', async () => {
    renderPage(undefined, undefined, [promotion], { ...promotion, status: 'draft' });
    await screen.findByText('My Company 1');
    fireEvent.click(screen.getByRole('button', { name: 'Предложения от поставщиков' }));
    const catalog = await screen.findByRole('dialog', { name: 'Предложения от поставщиков' });
    fireEvent.click(await within(catalog).findByRole('button', { name: 'Открыть' }));
    const detail = await screen.findByRole('dialog', { name: 'Предложение Promotion supplier' });
    fireEvent.click(within(detail).getByRole('button', { name: 'Заказать' }));
    expect(screen.getByRole('dialog', { name: 'Предложение Promotion supplier' })).toBeTruthy();
    expect(screen.queryByRole('dialog', { name: 'Заказ по акции' })).toBeNull();
  });

  it('shows catalog loading, empty and error states and closes from its footer', async () => {
    renderPage(undefined, undefined, [promotion], promotion, 'loading');
    await screen.findByText('My Company 1');
    fireEvent.click(screen.getByRole('button', { name: 'Предложения от поставщиков' }));
    expect(await screen.findByText('Загрузка…')).toBeTruthy();
    cleanup();

    renderPage(undefined, undefined, []);
    await screen.findByText('My Company 1');
    fireEvent.click(screen.getByRole('button', { name: 'Предложения от поставщиков' }));
    expect(await screen.findByText(/Сейчас у партнёров нет предложений/)).toBeTruthy();
    const emptyCatalog = screen.getByRole('dialog', { name: 'Предложения от поставщиков' });
    fireEvent.click(within(emptyCatalog.querySelector('.modal-footer')!).getByRole('button', { name: 'Закрыть' }));
    expect(screen.queryByRole('dialog', { name: 'Предложения от поставщиков' })).toBeNull();
    cleanup();

    renderPage(undefined, undefined, [promotion], promotion, 'error');
    await screen.findByText('My Company 1');
    fireEvent.click(screen.getByRole('button', { name: 'Предложения от поставщиков' }));
    expect(await screen.findByRole('alert')).toBeTruthy();
  });
});
