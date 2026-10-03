// @vitest-environment jsdom
import { createElement } from 'react';
import { QueryClient, QueryClientProvider } from '@tanstack/react-query';
import { cleanup, fireEvent, render, screen, waitFor, within } from '@testing-library/react';
import { afterEach, describe, expect, it, vi } from 'vitest';
import { OrderDialog } from './pages/trade';

const allocationUrl = '/api/v1/commerce/orders/o-1/allocations';
const guidance =
  'Назначаются свободные автомобили с VIN со склада. Если список пуст, откройте «Склады», примите автомобили с VIN или введите VIN в ранее принятой партии.';

const order = {
  id: 'o-1',
  party: 'buyer' as const,
  buyer: { name: 'Test Company' },
  supplier: { name: 'Supplier' },
  source: 'direct',
  terms: {
    lines: [{ lineId: 'line-1', modelId: 'm-1', quantity: '2', unitPrice: { amountMinor: '100000', currency: 'USD' } }],
    route: 'local',
    deliveryTerms: '',
    paymentSchedule: [],
    warrantyTerms: '',
    serviceTerms: '',
  },
  total: { amountMinor: '200000', currency: 'USD' },
  status: 'fulfilling',
  statusReason: '',
  allocations: [],
  shipments: [],
  addenda: [],
  allowedActions: ['allocate'],
  revision: 'rev-7',
  updatedAt: '2026-09-30T12:00:00Z',
};

const model = {
  id: 'm-1',
  specification: { make: 'Chevrolet', model: 'Cobalt', variant: 'LTZ', year: 2025, version: 1 },
  revision: '1',
};

const vehicle = {
  id: 'v-1',
  vin: 'XW8ZZZ61ZHG000001',
  modelId: 'm-1',
  modelSpecificationVersion: '1',
  placement: { warehouseId: 'w-1', placedAt: '2026-09-30T09:00:00Z' },
  reserved: false,
  revision: '1',
};

afterEach(() => {
  cleanup();
  vi.unstubAllGlobals();
});

function renderOrder(vehicles: unknown[], onPost?: (url: string, init: RequestInit) => void) {
  vi.stubGlobal(
    'fetch',
    vi.fn(async (input: RequestInfo | URL, init: RequestInit = {}) => {
      const url = String(input);
      if (init.method === 'POST') {
        onPost?.(url, init);
        return new Response(JSON.stringify({ data: order, revision: 'rev-8' }), {
          status: 201,
          headers: { 'Content-Type': 'application/json' },
        });
      }
      const json = url.includes('/commerce/orders/o-1/invoices')
        ? { items: [] }
        : url.includes('/commerce/orders/o-1')
          ? { data: order, revision: order.revision }
          : url.includes('/inventory/vehicle-units')
            ? { items: vehicles }
            : url.includes('/inventory/vehicle-models')
              ? { items: [model] }
              : { items: [] };
      return new Response(JSON.stringify(json), { status: 200, headers: { 'Content-Type': 'application/json' } });
    }),
  );
  const client = new QueryClient({ defaultOptions: { queries: { retry: false } } });
  render(createElement(QueryClientProvider, { client }, createElement(OrderDialog, { id: order.id, onClose: vi.fn() })));
}

async function openAllocation() {
  fireEvent.click(await screen.findByRole('button', { name: /Назначить VIN —/ }));
  expect(screen.getAllByRole('dialog')).toHaveLength(1);
  return screen.getByRole('dialog');
}

describe('order VIN allocation', () => {
  it('guides an empty inventory and keeps an empty selection local', async () => {
    let posts = 0;
    renderOrder([], () => posts++);
    const dialog = await openAllocation();
    expect(within(dialog).getByText(guidance)).toBeTruthy();
    fireEvent.click(within(dialog).getByRole('button', { name: 'Назначить VIN со склада' }));

    expect(await within(dialog).findByText('Выберите хотя бы один автомобиль')).toBeTruthy();
    expect(posts).toBe(0);
  });

  it('keeps an unselected eligible vehicle local', async () => {
    let posts = 0;
    renderOrder([vehicle], () => posts++);
    const dialog = await openAllocation();
    fireEvent.click(within(dialog).getByRole('button', { name: 'Назначить VIN со склада' }));

    expect(await within(dialog).findByText('Выберите хотя бы один автомобиль')).toBeTruthy();
    expect(posts).toBe(0);
  });

  it('posts the selected VIN allocation with its revision and refreshes', async () => {
    let posted: { url: string; body: unknown; ifMatch: string | null } | undefined;
    renderOrder([vehicle], (url, init) => {
      posted = { url, body: JSON.parse(String(init.body)), ifMatch: new Headers(init.headers).get('If-Match') };
    });
    const dialog = await openAllocation();
    fireEvent.click(await within(dialog).findByLabelText(/XW8ZZZ61ZHG000001/));
    fireEvent.click(within(dialog).getByRole('button', { name: 'Назначить VIN со склада' }));

    await waitFor(() => expect(posted).toBeDefined());
    expect(posted).toEqual({
      url: allocationUrl,
      body: { items: [{ orderLineId: 'line-1', vehicleId: 'v-1' }] },
      ifMatch: '"rev-7"',
    });
    expect(await screen.findByText('Назначено VIN: 1.')).toBeTruthy();
    expect(screen.getAllByRole('dialog')).toHaveLength(1);
  });
});
