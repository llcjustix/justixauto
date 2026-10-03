// @vitest-environment jsdom
import { QueryClient, QueryClientProvider } from '@tanstack/react-query';
import { cleanup, fireEvent, render, screen, waitFor } from '@testing-library/react';
import { afterEach, expect, it, vi } from 'vitest';
import { ReceiptIdentification } from './receipt-identification';
import { WarehouseDialog } from './pages/inventory';

vi.mock('./data', async (original) => ({ ...(await original<typeof import('./data')>()), useBranches: () => ({ data: [] }) }));

const batch = { id: 'batch-1', warehouseId: 'warehouse-1', modelId: 'model-1', modelSpecificationVersion: '1', exteriorColor: 'Белый', interiorColor: 'Чёрный', revision: '3', confirmedQuantity: '4', identifiedCount: '1', unidentifiedCount: '3' };
const json = (body: unknown, status = 200) => new Response(JSON.stringify(body), { status, headers: { 'Content-Type': 'application/json' } });
afterEach(() => { cleanup(); vi.unstubAllGlobals(); });

function setup(options: { failure?: boolean; missing?: boolean; warehouse?: boolean; count?: string } = {}) {
  let failure = options.failure ?? false;
  let remaining = options.count ?? '3';
  const posts: { url: string; body: unknown; headers: Headers }[] = [];
  const done = vi.fn();
  const client = new QueryClient({ defaultOptions: { queries: { retry: false } } });
  const invalidation = vi.spyOn(client, 'invalidateQueries');
  vi.stubGlobal('fetch', vi.fn(async (input: RequestInfo | URL, init: RequestInit = {}) => {
    const url = String(input);
    if (init.method === 'POST') {
      posts.push({ url, body: JSON.parse(String(init.body)), headers: new Headers(init.headers) });
      if (failure) return json({ error: { code: 'validation', message: 'Партия изменилась', fields: { items: 'VIN уже существует' } } }, 412);
      remaining = String(Number(remaining) - 1);
      return json({ data: {} }, 201);
    }
    if (url.includes('/inventory/warehouses/warehouse-1/inventory')) return json({ data: {
      warehouse: { id: 'warehouse-1', name: 'Warehouse', country: { label: 'Узбекистан' }, city: 'Ташкент', address: 'Address', capacity: '100', occupied: '4', free: '96', revision: '1' }, vehicles: [],
      unidentifiedBatches: options.missing ? [] : [{ ...batch, unidentifiedCount: remaining, identifiedCount: String(4 - Number(remaining)) }],
    } });
    if (url.includes('/vehicle-models')) return json({ items: [{ id: 'model-1', specification: { make: 'Make', model: 'Model', variant: 'Trim' } }] });
    return json({ items: [] });
  }));
  render(<QueryClientProvider client={client}>{options.warehouse ? <WarehouseDialog id="warehouse-1" onClose={vi.fn()} /> :
    <ReceiptIdentification batch={batch} modelName="Make Model" refresh={[[ 'order', 'order-1' ]]} onBack={vi.fn()} onDone={done} />}</QueryClientProvider>);
  return { done, posts, invalidation, allow: () => { failure = false; } };
}

it('normalizes the exact batch/model payload and refreshes partial progress without quantity correction', async () => {
  const run = setup();
  fireEvent.change(await screen.findByLabelText('VIN 1'), { target: { value: ' lgxc16df0p0000001 ' } });
  fireEvent.click(screen.getByRole('button', { name: 'Сохранить VIN' }));
  await waitFor(() => expect(run.done).toHaveBeenCalledWith(expect.stringContaining('Осталось без VIN: 2')));
  expect(run.posts).toHaveLength(1);
  expect(run.posts[0]?.url).toBe('/api/v1/inventory/receipt-batches/batch-1/identifications');
  expect(run.posts[0]?.body).toEqual({ atomic: true, items: [{ vin: 'LGXC16DF0P0000001', modelId: 'model-1' }] });
  expect(run.posts[0]?.headers.has('If-Match')).toBe(false);
  for (const key of [['order', 'order-1'], ['orders'], ['stock'], ['vehicles'], ['warehouses']])
    expect(run.invalidation).toHaveBeenCalledWith({ queryKey: key });
});

it('retains rows and aggregate conflict errors, then permits one corrected atomic submission', async () => {
  const run = setup({ failure: true });
  fireEvent.change(await screen.findByLabelText('VIN 1'), { target: { value: 'LGXC16DF0P0000001' } });
  fireEvent.click(screen.getByRole('button', { name: 'Сохранить VIN' }));
  expect(await screen.findByText(/VIN уже существует/)).toBeTruthy();
  expect((await screen.findByLabelText('VIN 1') as HTMLInputElement).value).toBe('LGXC16DF0P0000001');
  expect(run.done).not.toHaveBeenCalled();
  run.allow();
  fireEvent.change(screen.getByLabelText('VIN 1'), { target: { value: 'LGXC16DF0P0000002' } });
  fireEvent.click(screen.getByRole('button', { name: 'Сохранить VIN' }));
  await waitFor(() => expect(run.done).toHaveBeenCalledOnce());
  expect(run.posts).toHaveLength(2);
});

it('has read-only completion when fresh stock no longer lists the pending batch', async () => {
  const run = setup({ missing: true });
  expect(await screen.findByText(/больше нет автомобилей/)).toBeTruthy();
  expect(screen.queryByRole('button', { name: 'Сохранить VIN' })).toBeNull();
  expect(run.posts).toHaveLength(0);
});

it('caps each identification request at 1000 even for a larger batch', async () => {
  setup({ count: '1500' });
  expect(await screen.findByText(/За один раз — до 1000 VIN/)).toBeTruthy();
});

it('reuses the editor in the warehouse modal and preserves receipt, capacity and correction commands', async () => {
  setup({ warehouse: true });
  await screen.findByRole('button', { name: 'Исправить кол-во' });
  expect(screen.getByRole('button', { name: 'Принять автомобили' })).toBeTruthy();
  expect(screen.getByRole('button', { name: 'Изменить вместимость' })).toBeTruthy();
  fireEvent.click(screen.getByRole('button', { name: 'Ввести VIN' }));
  await screen.findByLabelText('VIN 1');
  expect(screen.getAllByRole('dialog')).toHaveLength(1);
  fireEvent.click(screen.getByRole('button', { name: 'Назад' }));
  expect(screen.getByRole('button', { name: 'Исправить кол-во' })).toBeTruthy();
});
