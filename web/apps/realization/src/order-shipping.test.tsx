// @vitest-environment jsdom
import { createElement } from 'react';
import { QueryClient, QueryClientProvider } from '@tanstack/react-query';
import { cleanup, fireEvent, render, screen, waitFor, within } from '@testing-library/react';
import { afterEach, describe, expect, it, vi } from 'vitest';
import { OrderDialog } from './pages/trade';

const order = {
  id: 'o-1',
  party: 'supplier' as const,
  buyer: { name: 'Buyer Co' },
  supplier: { name: 'Test Company' },
  source: 'direct',
  terms: {
    lines: [{ lineId: 'line-1', modelId: 'm-1', modelSpecificationVersion: '1', exteriorColor: 'Белый', interiorColor: 'Чёрный', quantity: '10', unitPrice: { amountMinor: '100000', currency: 'USD' } }],
    route: 'local',
    deliveryTerms: '',
    paymentSchedule: [],
    warrantyTerms: '',
    serviceTerms: '',
  },
  receivingWarehouseId: null,
  hasReceivingWarehouse: true,
  total: { amountMinor: '1000000', currency: 'USD' },
  status: 'fulfilling',
  statusReason: '',
  allocations: [],
  shipments: [],
  lineProgress: [{ orderLineId: 'line-1', shipped: '4' }],
  addenda: [],
  allowedActions: ['allocate', 'ship-quantity'],
  revision: 'rev-3',
  updatedAt: '2026-10-01T06:00:00Z',
};

const model = {
  id: 'm-1',
  specification: { make: 'BYD', model: 'Champion', variant: 'DM-i', year: 2026, version: 1 },
  revision: '1',
};

afterEach(() => {
  cleanup();
  vi.unstubAllGlobals();
});

function renderOrder(onPost: (url: string, init: RequestInit) => void, shown: object = order) {
  vi.stubGlobal(
    'fetch',
    vi.fn(async (input: RequestInfo | URL, init: RequestInit = {}) => {
      const url = String(input);
      if (init.method === 'POST') {
        onPost(url, init);
        return new Response(JSON.stringify({ data: { id: 's-1' }, revision: '1' }), {
          status: 201,
          headers: { 'Content-Type': 'application/json' },
        });
      }
      const json = url.includes('/commerce/orders/o-1/invoices')
        ? { items: [] }
        : url.includes('/commerce/orders/o-1')
          ? { data: shown, revision: order.revision }
          : url.includes('/inventory/vehicle-models')
            ? { items: [model] }
            : url.includes('/inventory/warehouses')
              ? { items: [{ id: 'w-1', name: 'Главный склад', free: '90' }] }
              : { items: [] };
      return new Response(JSON.stringify(json), { status: 200, headers: { 'Content-Type': 'application/json' } });
    }),
  );
  const client = new QueryClient({ defaultOptions: { queries: { retry: false } } });
  render(createElement(QueryClientProvider, { client }, createElement(OrderDialog, { id: order.id, onClose: vi.fn() })));
}

describe('order shipping by quantity', () => {
  it('ships the remaining quantity from the order, with nothing in the own stock', async () => {
    let posted: { url: string; body: unknown; ifMatch: string | null } | undefined;
    renderOrder((url, init) => {
      posted = { url, body: JSON.parse(String(init.body)), ifMatch: new Headers(init.headers).get('If-Match') };
    });
    await screen.findByRole('button', { name: /Назначить VIN —/ });
    expect(screen.getByText('4 из 10')).toBeTruthy();
    fireEvent.click(screen.getByRole('button', { name: 'Отгрузить по количеству' }));
    expect(screen.getAllByRole('dialog')).toHaveLength(1);
    const dialog = screen.getByRole('dialog');
    const quantity = within(dialog).getByLabelText(/количество \(осталось 6\)/) as HTMLInputElement;
    expect(quantity.value).toBe('6');
    fireEvent.change(quantity, { target: { value: '5' } });
    fireEvent.click(within(dialog).getByRole('button', { name: 'Добавить VIN' }));
    fireEvent.change(within(dialog).getByLabelText('VIN 1'), { target: { value: ' lgxc16df0p0000001 ' } });
    fireEvent.click(within(dialog).getByRole('button', { name: 'Отгрузить по количеству' }));

    await waitFor(() => expect(posted).toBeDefined());
    expect(posted).toEqual({
      url: '/api/v1/commerce/orders/o-1/shipments',
      body: { lines: [{ orderLineId: 'line-1', quantity: '5', vins: ['LGXC16DF0P0000001'] }], route: 'local' },
      ifMatch: '"rev-3"',
    });
    expect(await screen.findByText('Отгружено: 5. С VIN: 1. Без VIN: 4.')).toBeTruthy();
  });

  it('does not send a shipment without any quantity', async () => {
    let posts = 0;
    renderOrder(() => posts++);
    fireEvent.click(await screen.findByRole('button', { name: 'Отгрузить по количеству' }));
    const dialog = screen.getByRole('dialog');
    fireEvent.change(within(dialog).getByLabelText(/количество/), { target: { value: '0' } });
    fireEvent.click(within(dialog).getByRole('button', { name: 'Отгрузить по количеству' }));
    expect(await within(dialog).findByText('Укажите количество хотя бы по одной строке')).toBeTruthy();
    expect(posts).toBe(0);
  });

  it('tells the supplier why an order without a receiving warehouse cannot be shipped', async () => {
    renderOrder(() => {}, { ...order, hasReceivingWarehouse: false, allowedActions: ['allocate'] });
    expect(await screen.findByText(/покупатель ещё не выбрал склад получения/)).toBeTruthy();
    expect(screen.queryByRole('button', { name: 'Отгрузить по количеству' })).toBeNull();
  });

  it('keeps allocated VIN shipping reachable for a legacy order inside the same dialog', async () => {
    let body: unknown;
    renderOrder((_url, init) => { body = JSON.parse(String(init.body)); }, {
      ...order, hasReceivingWarehouse: false, allowedActions: ['ship'],
      allocations: [{ orderLineId: 'line-1', vehicleId: 'v-1', vin: 'LGXC16DF0P0000001', status: 'allocated', shipmentId: null }],
    });
    fireEvent.click(await screen.findByRole('button', { name: 'Отгрузить назначенные VIN' }));
    expect(screen.getAllByRole('dialog')).toHaveLength(1);
    fireEvent.click(screen.getByLabelText('LGXC16DF0P0000001'));
    fireEvent.click(screen.getByRole('button', { name: 'Отгрузить назначенные VIN' }));
    await waitFor(() => expect(body).toEqual({ vehicleIds: ['v-1'], route: 'local' }));
  });

  it('lets the buyer choose the receiving warehouse of an existing order', async () => {
    let posted: { url: string; body: unknown } | undefined;
    renderOrder((url, init) => (posted = { url, body: JSON.parse(String(init.body)) }), {
      ...order,
      party: 'buyer',
      hasReceivingWarehouse: false,
      allowedActions: ['set-warehouse'],
    });
    fireEvent.click(await screen.findByRole('button', { name: 'Выбрать склад получения' }));
    const dialog = await screen.findByRole('dialog', { name: 'Выбрать склад получения' });
    const warehouse = within(dialog).getByRole('combobox', { name: /Склад получения/ });
    fireEvent.focus(warehouse);
    fireEvent.click(await within(dialog).findByRole('option', { name: /Главный склад/ }));
    fireEvent.click(within(dialog).getByRole('button', { name: 'Выбрать склад получения' }));
    await waitFor(() => expect(posted).toBeDefined());
    expect(posted).toEqual({ url: '/api/v1/commerce/orders/o-1/receiving-warehouse', body: { warehouseId: 'w-1' } });
  });
});
