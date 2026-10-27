// @vitest-environment jsdom
import { QueryClient, QueryClientProvider } from '@tanstack/react-query';
import { cleanup, fireEvent, render, screen, waitFor, within } from '@testing-library/react';
import { afterEach, expect, it, vi } from 'vitest';
import { SessionGate } from '@justixauto/kit';
import { VehicleMoveAction } from './vehicle-move';
import type { Vehicle, Warehouse } from './data';

const json = (body: unknown, status = 200) =>
  new Response(JSON.stringify(body), { status, headers: { 'Content-Type': 'application/json' } });
const vehicle: Vehicle = {
  id: 'v-1',
  vin: 'XW8ZZZ61ZHG000001',
  modelId: 'm-1',
  modelSpecificationVersion: '1',
  placement: { warehouseId: 'w-1', placedAt: '2026-10-11T10:00:00Z' },
  reserved: false,
  ownerCompanyId: 'company',
  custodianCompanyId: 'company',
  revision: '1',
};
const warehouses: Warehouse[] = [
  {
    id: 'w-1',
    branchId: null,
    name: 'Источник',
    city: 'Ташкент',
    address: 'A',
    country: { label: 'Узбекистан' },
    capacity: '10',
    occupied: '1',
    free: '9',
    revision: '1',
  },
  {
    id: 'w-2',
    branchId: null,
    name: 'Получатель',
    city: 'Ташкент',
    address: 'B',
    country: { label: 'Узбекистан' },
    capacity: '10',
    occupied: '0',
    free: '10',
    revision: '1',
  },
];
const session = (permissions: string[]) => ({
  data: {
    user: { id: 'u-1', displayName: 'Сотрудник', status: 'active', passwordChangeRequired: false },
    roles: [],
    permissions,
    context: { revision: '1', companyId: 'company', branchScope: { mode: 'ALL', branchIds: [] } },
    accessibleCompanies: [
      {
        id: 'company',
        name: 'Компания',
        kind: 'seller',
        access: 'full',
        userType: 'dealer',
        verification: 'verified',
        verificationReason: '',
      },
    ],
    setup: { next: 'none' },
  },
});

afterEach(() => {
  cleanup();
  vi.unstubAllGlobals();
});

function setup(permissions = ['inventory.vehicles.move'], response = json({ data: vehicle })) {
  const requests: { url: string; body?: unknown }[] = [];
  vi.stubGlobal(
    'fetch',
    vi.fn(async (input: RequestInfo | URL, init: RequestInit = {}) => {
      const url = String(input);
      if (url.endsWith('/identity/session')) return json(session(permissions));
      if (init.method === 'POST') {
        requests.push({ url, body: JSON.parse(String(init.body)) });
        return response;
      }
      return json({ data: {} });
    }),
  );
  const client = new QueryClient({ defaultOptions: { queries: { retry: false } } });
  const invalidation = vi.spyOn(client, 'invalidateQueries');
  render(
    <QueryClientProvider client={client}>
      <SessionGate title="Тест">
        <VehicleMoveAction vehicle={vehicle} warehouses={warehouses} />
      </SessionGate>
    </QueryClientProvider>,
  );
  return { requests, invalidation };
}

it('limits targets to same-company warehouses and sends the exact move payload', async () => {
  const run = setup();
  fireEvent.click(await screen.findByRole('button', { name: 'Переместить' }));
  fireEvent.focus(screen.getByLabelText('На склад'));
  fireEvent.click(await screen.findByRole('option', { name: /Получатель/ }));
  fireEvent.change(screen.getByLabelText('Когда'), { target: { value: '2026-10-11T11:00' } });
  fireEvent.click(within(screen.getByRole('dialog')).getByRole('button', { name: 'Переместить' }));
  await waitFor(() =>
    expect(run.requests).toEqual([
      {
        url: '/api/v1/inventory/vehicle-units/v-1/warehouse-moves',
        body: {
          fromWarehouseId: 'w-1',
          toWarehouseId: 'w-2',
          occurredAt: new Date('2026-10-11T11:00').toISOString(),
          evidenceBindingIds: [],
        },
      },
    ]),
  );
  for (const key of [['vehicle', 'v-1'], ['vehicles'], ['warehouses'], ['stock', 'w-1'], ['stock', 'w-2']])
    expect(run.invalidation).toHaveBeenCalledWith({ queryKey: key });
});

it('hides movement without the actual permission', async () => {
  setup([]);
  await waitFor(() => expect(screen.queryByRole('button', { name: 'Переместить' })).toBeNull());
});

it('retains the dialog and refreshes current placement on source mismatch', async () => {
  const run = setup(['inventory.vehicles.move'], json({ error: { code: 'placement_changed', message: 'stale' } }, 409));
  fireEvent.click(await screen.findByRole('button', { name: 'Переместить' }));
  fireEvent.focus(screen.getByLabelText('На склад'));
  fireEvent.click(await screen.findByRole('option', { name: /Получатель/ }));
  fireEvent.change(screen.getByLabelText('Когда'), { target: { value: '2026-10-11T11:00' } });
  fireEvent.click(within(screen.getByRole('dialog')).getByRole('button', { name: 'Переместить' }));
  expect(await screen.findByText('Размещение автомобиля изменилось. Обновите данные.')).toBeTruthy();
  expect(screen.getByLabelText('На склад')).toBeTruthy();
  expect(run.requests).toHaveLength(1);
  expect(run.invalidation).toHaveBeenCalledWith({ queryKey: ['vehicle', 'v-1'] });
});

it('does not offer movement when the vehicle has no placement', async () => {
  const parkedElsewhere = { ...vehicle, placement: null };
  vi.stubGlobal(
    'fetch',
    vi.fn(async (input: RequestInfo | URL) =>
      String(input).endsWith('/identity/session') ? json(session(['inventory.vehicles.move'])) : json({ data: {} }),
    ),
  );
  const client = new QueryClient({ defaultOptions: { queries: { retry: false } } });
  render(
    <QueryClientProvider client={client}>
      <SessionGate title="Тест">
        <VehicleMoveAction vehicle={parkedElsewhere} warehouses={warehouses} />
      </SessionGate>
    </QueryClientProvider>,
  );
  await waitFor(() => expect(screen.queryByRole('button', { name: 'Переместить' })).toBeNull());
});
