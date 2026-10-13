// @vitest-environment jsdom
import { QueryClient, QueryClientProvider } from '@tanstack/react-query';
import { cleanup, fireEvent, render, screen, waitFor } from '@testing-library/react';
import { afterEach, describe, expect, it, vi } from 'vitest';
import type { Order, Vehicle } from './data';
import { OrderFulfilmentPanel } from './order-fulfilment';
import type { OrderFulfilmentAction } from './order-fulfilment';

const vin = (index: number) => `LGXC16DF0P${String(index).padStart(7, '0')}`;
const vehicle = (index: number, overrides: Partial<Vehicle> = {}): Vehicle => ({ id: `v-${index}`, vin: vin(index), modelId: 'm-1', modelSpecificationVersion: '1', exteriorColor: 'Белый', interiorColor: 'Чёрный', placement: { warehouseId: 'w-1', placedAt: '' }, reserved: false, revision: '1', ...overrides });
const order: Order = {
  id: 'o-1', party: 'supplier', buyer: { name: 'Buyer' }, supplier: { name: 'Supplier' }, source: 'direct',
  terms: { lines: [{ lineId: 'l-1', modelId: 'm-1', modelSpecificationVersion: '1', exteriorColor: 'Белый', interiorColor: 'Чёрный', quantity: '10', unitPrice: { amountMinor: '100', currency: 'USD' } }], route: 'local', deliveryTerms: '', paymentSchedule: [], warrantyTerms: '', serviceTerms: '' },
  total: { amountMinor: '1000', currency: 'USD' }, status: 'fulfilling', statusReason: '', allocations: [], shipments: [], addenda: [],
  lineProgress: [{ orderLineId: 'l-1', shipped: '4', allocated: '2', identified: '1', unidentified: '3', receiptQuantityAdjusted: false }],
  allowedActions: ['allocate', 'ship-quantity', 'ship'], revision: 'rev-3', updatedAt: '', hasReceivingWarehouse: true,
};
const json = (body: unknown, status = 200) => new Response(JSON.stringify(body), { status, headers: { 'Content-Type': 'application/json' } });
const paletteV1 = { version: '1', exteriorColors: ['Белый', 'Синий'], interiorColors: ['Чёрный', 'Бежевый'] };
const paletteV2 = { version: '2', exteriorColors: ['Красный'], interiorColors: ['Серый'] };
afterEach(() => { cleanup(); vi.unstubAllGlobals(); });

async function choose(label: string, option: string) {
  const input = await screen.findByRole('combobox', { name: label, exact: true });
  await waitFor(() => expect(input.matches(':disabled')).toBe(false));
  fireEvent.focus(input);
  fireEvent.change(input, { target: { value: option } });
  fireEvent.click(await screen.findByRole('option', { name: option, exact: true }));
  fireEvent.blur(input);
}
async function openChoices(label: string, option: string) {
  const input = await screen.findByRole('combobox', { name: label, exact: true });
  await waitFor(() => expect(input.matches(':disabled')).toBe(false));
  fireEvent.focus(input);
  await screen.findByRole('option', { name: option, exact: true });
}


function setup(options: {
  action?: OrderFulfilmentAction;
  shown?: Order;
  stock?: (params: URLSearchParams) => Vehicle[] | Promise<Vehicle[]>;
  post?: (init: RequestInit) => Response | Promise<Response>;
  detail?: () => Response | Promise<Response>;
} = {}) {
  const reads: URLSearchParams[] = [];
  const posts: RequestInit[] = [];
  vi.stubGlobal('fetch', vi.fn(async (input: RequestInfo | URL, init: RequestInit = {}) => {
    const url = String(input);
    if (init.method === 'POST') { posts.push(init); return options.post?.(init) ?? json({ data: {}, revision: 'rev-4' }, 201); }
    if (/\/vehicle-models\/m-/.test(url)) return options.detail?.() ?? json({ data: { id: 'm-1', specification: paletteV2, versions: [paletteV1, paletteV2] } });
    if (url.includes('/vehicle-models')) return json({ items: ['m-1', 'm-2'].map((id) => ({ id, specification: { make: 'BYD', model: id, variant: 'DM-i' } })) });
    if (url.includes('/vehicle-units')) {
      const params = new URL(url, 'http://localhost').searchParams;
      reads.push(params);
      return json({ items: await (options.stock?.(params) ?? []) });
    }
    if (url.includes('/warehouses')) return json({ items: [{ id: 'w-1', name: 'Основной' }, { id: 'w-2', name: 'Второй' }] });
    return json({ items: [] });
  }));
  const client = new QueryClient({ defaultOptions: { queries: { retry: false }, mutations: { retry: false } } });
  const invalidate = vi.spyOn(client, 'invalidateQueries');
  const done = vi.fn();
  const back = vi.fn();
  const props = { order: options.shown ?? order, action: options.action ?? { kind: 'allocate', lineId: 'l-1' } as OrderFulfilmentAction, refresh: [['order', 'o-1'], ['orders'], ['vehicles']], onDone: done, onBack: back };
  const view = render(<QueryClientProvider client={client}><OrderFulfilmentPanel {...props} /></QueryClientProvider>);
  return { posts, reads, done, invalidate, update: (next: Partial<typeof props>) => view.rerender(<QueryClientProvider client={client}><OrderFulfilmentPanel {...props} {...next} /></QueryClientProvider>) };
}

describe('line-scoped stock fulfilment', () => {
  it('queries owned eligible warehouse stock for the exact model and excludes invalid rows', async () => {
    const run = setup({ stock: () => [vehicle(1), vehicle(2, { modelId: 'm-2' }), vehicle(3, { reserved: true }), vehicle(4, { placement: null })] });
    await screen.findByLabelText(`Выбрать ${vin(1)}`);
    expect(screen.getAllByRole('checkbox')).toHaveLength(1);
    expect(Object.fromEntries(run.reads[0])).toEqual({ modelId: 'm-1', placement: 'warehouse', eligible: 'true', limit: '50', offset: '0' });
    expect(screen.getByText(/Осталось: 4/)).toBeTruthy();
  });

  it('retains page-one selection on page two and sends exact IDs once with If-Match', async () => {
    let release!: (response: Response) => void;
    const run = setup({ stock: (params) => params.get('offset') === '50' ? [vehicle(51)] : Array.from({ length: 50 }, (_, i) => vehicle(i)), post: () => new Promise((resolve) => { release = resolve; }) });
    fireEvent.click(await screen.findByLabelText(`Выбрать ${vin(1)}`));
    fireEvent.click(screen.getByRole('button', { name: 'Следующая страница' }));
    fireEvent.click(await screen.findByLabelText(`Выбрать ${vin(51)}`));
    expect(screen.getByRole('button', { name: `Убрать ${vin(1)}` })).toBeTruthy();
    const submit = screen.getByRole('button', { name: 'Назначить VIN со склада' });
    fireEvent.click(submit); fireEvent.click(submit);
    expect(run.posts).toHaveLength(1);
    expect(JSON.parse(String(run.posts[0].body))).toEqual({ items: [{ orderLineId: 'l-1', vehicleId: 'v-1' }, { orderLineId: 'l-1', vehicleId: 'v-51' }] });
    expect(new Headers(run.posts[0].headers).get('If-Match')).toBe('"rev-3"');
    release(json({ data: {} }, 201));
    await waitFor(() => expect(run.done).toHaveBeenCalledWith('Назначено VIN: 2.'));
    expect(run.done).toHaveBeenCalledTimes(1);
    const keys = run.invalidate.mock.calls.map(([arg]) => JSON.stringify(arg?.queryKey));
    expect(keys).toContain('["eligible-vehicles"]');
    expect(keys).toContain('["warehouses"]');
    expect(keys).toContain('["stock"]');
    expect(keys.filter((key) => key === '["vehicles"]')).toHaveLength(1);
  });

  it('enforces remaining capacity and lets users deselect outside the current filter', async () => {
    const shown = { ...order, terms: { ...order.terms, lines: [{ ...order.terms.lines[0], quantity: '7' }] } };
    setup({ shown, stock: (params) => params.get('warehouseId') === 'w-2' ? [] : [vehicle(1), vehicle(2)] });
    fireEvent.click(await screen.findByLabelText(`Выбрать ${vin(1)}`));
    expect((screen.getByLabelText(`Выбрать ${vin(2)}`) as HTMLInputElement).disabled).toBe(true);
    await choose('Склад', 'Второй');
    await screen.findByText('Нет подходящих автомобилей на этой странице.');
    fireEvent.click(screen.getByRole('button', { name: `Убрать ${vin(1)}` }));
    expect(screen.getByText('Выбрано: 0 из 1. Осталось по строке: 1.')).toBeTruthy();
  });

  it('does not offer late search results and resets selection when changing line/model', async () => {
    let release!: (rows: Vehicle[]) => void;
    const shown = { ...order, terms: { ...order.terms, lines: [...order.terms.lines, { ...order.terms.lines[0], lineId: 'l-2', modelId: 'm-2' }] } };
    const run = setup({ shown, stock: (params) => {
      if (params.get('search') === 'OLD') return new Promise((resolve) => { release = resolve; });
      if (params.get('search') === 'NEW') return [];
      return params.get('modelId') === 'm-2' ? [vehicle(2, { modelId: 'm-2' })] : [vehicle(1)];
    } });
    fireEvent.click(await screen.findByLabelText(`Выбрать ${vin(1)}`));
    fireEvent.change(screen.getByLabelText('Поиск VIN'), { target: { value: 'OLD' } });
    await waitFor(() => expect(release).toBeDefined());
    fireEvent.change(screen.getByLabelText('Поиск VIN'), { target: { value: 'NEW' } });
    release([vehicle(3)]);
    await screen.findByText('Нет подходящих автомобилей на этой странице.');
    expect(screen.queryByLabelText(`Выбрать ${vin(3)}`)).toBeNull();
    run.update({ action: { kind: 'allocate', lineId: 'l-2' } });
    await screen.findByLabelText(`Выбрать ${vin(2)}`);
    expect(screen.queryByRole('button', { name: `Убрать ${vin(1)}` })).toBeNull();
    expect(screen.getByText('Выбрано: 0 из 10. Осталось по строке: 10.')).toBeTruthy();
  });

  it('selects an explicit page count and fills remaining capacity across server pages', async () => {
    const shown = { ...order, terms: { ...order.terms, lines: [{ ...order.terms.lines[0], quantity: '1015' }] } };
    const run = setup({ shown, stock: (params) => {
      const start = Number(params.get('offset'));
      const count = Number(params.get('limit'));
      return Array.from({ length: count }, (_, index) => vehicle(start + index));
    } });
    await screen.findByLabelText(`Выбрать ${vin(0)}`);
    fireEvent.click(screen.getByRole('button', { name: 'Выбрать на странице: 50' }));
    expect(screen.getByText('Выбрано: 50 из 1000. Осталось по строке: 1009.')).toBeTruthy();
    fireEvent.click(screen.getByRole('button', { name: 'Заполнить лимит из результатов: до 1000' }));
    await screen.findByText('Выбрано: 1000 из 1000. Осталось по строке: 1009.');
    expect(run.reads.some((params) => params.get('offset') === '900' && params.get('limit') === '100')).toBe(true);
    expect(screen.getByText(/Оставшиеся назначьте следующей партией/)).toBeTruthy();
  });

  it('retains a conflicted stock choice and requires an explicit correction/retry', async () => {
    const run = setup({ stock: () => [vehicle(1)], post: () => json({ error: { code: 'conflict', message: 'Автомобиль уже занят', fields: { items: 'Проверьте выбор' } } }, 409) });
    fireEvent.click(await screen.findByLabelText(`Выбрать ${vin(1)}`));
    fireEvent.click(screen.getByRole('button', { name: 'Назначить VIN со склада' }));
    await screen.findByText(/items: Проверьте выбор/);
    expect(screen.getByRole('button', { name: `Убрать ${vin(1)}` })).toBeTruthy();
    await waitFor(() => expect(run.reads.length).toBe(2));
    expect(run.posts).toHaveLength(1);
    expect(run.done).not.toHaveBeenCalled();
  });
});

describe('colors fulfilment', () => {
  it('filters actual pairs on initial and bulk pages without requiring the ordered specification version', async () => {
    const matching = vehicle(101, { modelSpecificationVersion: '9' });
    const run = setup({ stock: params => params.get('offset') === '100' ? [matching] : Array.from({ length: 100 }, (_, index) => vehicle(index, { exteriorColor: index % 2 ? 'Синий' : '' })) });
    await screen.findByText('Нет подходящих автомобилей на этой странице.');
    fireEvent.click(screen.getByRole('button', { name: 'Заполнить лимит из результатов: до 4' }));
    await screen.findByRole('button', { name: `Убрать ${matching.vin}` });
    fireEvent.click(screen.getByRole('button', { name: 'Назначить VIN со склада' }));
    await waitFor(() => expect(run.posts).toHaveLength(1));
    expect(JSON.parse(String(run.posts[0].body))).toEqual({ items: [{ orderLineId: 'l-1', vehicleId: 'v-101' }] });
    expect(run.reads.some(params => params.get('offset') === '100')).toBe(true);
  });

  it('clears a stale selected pair after ordered facts change and respects independently known interior', async () => {
    const run = setup({ stock: () => [vehicle(1), vehicle(2, { exteriorColor: 'Синий', interiorColor: '' }), vehicle(3, { exteriorColor: 'Синий' })] });
    fireEvent.click(await screen.findByLabelText(`Выбрать ${vin(1)}`));
    run.update({ order: { ...order, terms: { ...order.terms, lines: [{ ...order.terms.lines[0]!, exteriorColor: 'Синий' }] } } });
    await waitFor(() => expect(screen.queryByRole('button', { name: `Убрать ${vin(1)}` })).toBeNull());
    expect(screen.queryByLabelText(`Выбрать ${vin(2)}`)).toBeNull();
    expect(screen.getByLabelText(`Выбрать ${vin(3)}`)).toBeTruthy();
    run.update({ order: { ...order, terms: { ...order.terms, lines: [{ ...order.terms.lines[0]!, exteriorColor: '' }] } } });
    expect(screen.getByLabelText(`Выбрать ${vin(1)}`)).toBeTruthy();
    expect(screen.queryByLabelText(`Выбрать ${vin(2)}`)).toBeNull();
  });

  it('requires explicit incoming version on unpinned history and keeps saved terms unchanged', async () => {
    const shown = { ...order, terms: { ...order.terms, lines: [{ ...order.terms.lines[0]!, modelSpecificationVersion: '', exteriorColor: '', interiorColor: '' }] } };
    const before = JSON.stringify(shown.terms);
    const run = setup({ shown, action: { kind: 'ship-quantity' } });
    await openChoices('Версия входящей спецификации', 'Версия 1');
    expect((screen.getByRole('combobox', { name: 'Версия входящей спецификации' }) as HTMLInputElement).value).toBe('Выберите версию');
    fireEvent.click(screen.getByRole('button', { name: 'Отгрузить по количеству' }));
    expect(run.posts).toHaveLength(0);
    await choose('Версия входящей спецификации', 'Версия 1');
    expect((screen.getByRole('combobox', { name: 'Цвет кузова' }) as HTMLInputElement).value).toBe('Выберите цвет');
    await choose('Цвет кузова', 'Синий');
    await choose('Цвет салона', 'Бежевый');
    expect(screen.getByText('Поступление: Кузов: Синий · Салон: Бежевый · Версия: 1')).toBeTruthy();
    fireEvent.click(screen.getByRole('button', { name: 'Отгрузить по количеству' }));
    await waitFor(() => expect(run.posts).toHaveLength(1));
    expect(JSON.parse(String(run.posts[0].body)).lines[0]).toEqual({ orderLineId: 'l-1', quantity: '4', vins: [], modelSpecificationVersion: '1', exteriorColor: 'Синий', interiorColor: 'Бежевый' });
    expect(JSON.stringify(shown.terms)).toBe(before);
  });

  it('resolves only missing pinned colors from v1 after current v2 and keeps two same-model rows distinct', async () => {
    const shown = { ...order, terms: { ...order.terms, lines: [{ ...order.terms.lines[0]!, interiorColor: '' }, { ...order.terms.lines[0]!, lineId: 'l-2', exteriorColor: 'Синий' }] } };
    const run = setup({ shown, action: { kind: 'ship-quantity' } });
    await openChoices('Цвет салона', 'Бежевый');
    expect(screen.queryByLabelText('Цвет кузова')).toBeNull();
    expect(screen.queryByRole('option', { name: 'Серый' })).toBeNull();
    await choose('Цвет салона', 'Бежевый');
    fireEvent.click(screen.getByRole('button', { name: 'Отгрузить по количеству' }));
    await waitFor(() => expect(run.posts).toHaveLength(1));
    expect(JSON.parse(String(run.posts[0].body)).lines).toEqual([{ orderLineId: 'l-1', quantity: '4', vins: [], interiorColor: 'Бежевый' }, { orderLineId: 'l-2', quantity: '10', vins: [] }]);
  });

  it('blocks missing loading and failed exact specification until explicit retry succeeds', async () => {
    let release!: (value: Response) => void;
    let calls = 0;
    const shown = { ...order, terms: { ...order.terms, lines: [{ ...order.terms.lines[0]!, interiorColor: '' }] } };
    const run = setup({ shown, action: { kind: 'ship-quantity' }, detail: () => ++calls === 1 ? new Promise(resolve => { release = resolve; }) : calls === 2 ? json({ error: { message: 'Offline' } }, 500) : json({ data: { id: 'm-1', versions: [paletteV1, paletteV2] } }) });
    await waitFor(() => expect(release).toBeDefined());
    fireEvent.click(screen.getByRole('button', { name: 'Отгрузить по количеству' }));
    release(json({ data: { id: 'm-1', specification: paletteV2, versions: [paletteV2] } }));
    fireEvent.click(await screen.findByRole('button', { name: 'Повторить загрузку спецификаций' }));
    await waitFor(() => expect(calls).toBe(2));
    await waitFor(() => expect((screen.getByRole('button', { name: 'Повторить загрузку спецификаций' }) as HTMLButtonElement).closest('fieldset')?.disabled).toBe(false));
    fireEvent.click(screen.getByRole('button', { name: 'Отгрузить по количеству' }));
    expect(run.posts).toHaveLength(0);
    fireEvent.click(screen.getByRole('button', { name: 'Повторить загрузку спецификаций' }));
    await openChoices('Цвет салона', 'Бежевый');
    await choose('Цвет салона', 'Бежевый');
    fireEvent.click(screen.getByRole('button', { name: 'Отгрузить по количеству' }));
    await waitFor(() => expect(run.posts).toHaveLength(1));
  });
});

describe('structured quantity fulfilment', () => {
  it('subtracts shipped and allocated stock, supports partial quantities and ship-all, and keeps VINs optional', async () => {
    const run = setup({ action: { kind: 'ship-quantity' } });
    const input = await screen.findByLabelText(/количество \(осталось 4\)/);
    expect((input as HTMLInputElement).value).toBe('4');
    fireEvent.change(input, { target: { value: '2' } });
    fireEvent.click(screen.getByRole('button', { name: 'Отгрузить весь остаток' }));
    expect((input as HTMLInputElement).value).toBe('4');
    fireEvent.change(input, { target: { value: '2' } });
    fireEvent.click(screen.getByRole('button', { name: 'Отгрузить по количеству' }));
    await waitFor(() => expect(run.done).toHaveBeenCalledWith('Отгружено: 2. С VIN: 0. Без VIN: 2.'));
    expect(JSON.parse(String(run.posts[0].body))).toEqual({ lines: [{ orderLineId: 'l-1', quantity: '2', vins: [] }], route: 'local' });
  });

  it('retains VINs above a lowered quantity and rejects invalid quantities without posting', async () => {
    const run = setup({ action: { kind: 'ship-quantity' } });
    const input = await screen.findByLabelText(/количество/);
    fireEvent.click(screen.getByRole('button', { name: 'Добавить VIN' }));
    fireEvent.change(screen.getByLabelText('VIN 1'), { target: { value: vin(1) } });
    fireEvent.click(screen.getByRole('button', { name: 'Добавить VIN' }));
    fireEvent.change(screen.getByLabelText('VIN 2'), { target: { value: vin(2) } });
    fireEvent.change(input, { target: { value: '1' } });
    expect(screen.getByText(/VIN больше допустимого: 2 из 1/)).toBeTruthy();
    fireEvent.click(screen.getByRole('button', { name: 'Отгрузить по количеству' }));
    expect((screen.getByLabelText('VIN 2') as HTMLInputElement).value).toBe(vin(2));
    for (const value of ['5', '1.5', '-1']) {
      fireEvent.change(input, { target: { value } });
      fireEvent.click(screen.getByRole('button', { name: 'Отгрузить по количеству' }));
    }
    expect(run.posts).toHaveLength(0);
  });

  it('retains aggregate server errors and normalized VIN intent after 412, then uses the new revision on explicit retry', async () => {
    let attempts = 0;
    const run = setup({ action: { kind: 'ship-quantity' }, post: () => ++attempts === 1 ? json({ error: { code: 'stale_revision', message: 'Заказ изменён', fields: { vins: 'VIN уже используется' } } }, 412) : json({ data: {} }, 201) });
    const input = await screen.findByLabelText(/количество/);
    fireEvent.change(input, { target: { value: '3' } });
    fireEvent.click(screen.getByRole('button', { name: 'Добавить VIN' }));
    fireEvent.change(screen.getByLabelText('VIN 1'), { target: { value: ' lgxc16df0p0000001 ' } });
    fireEvent.click(screen.getByRole('button', { name: 'Отгрузить по количеству' }));
    await screen.findByText(/vins: VIN уже используется/);
    expect((screen.getByLabelText('VIN 1') as HTMLInputElement).value).toBe(' lgxc16df0p0000001 ');
    expect((input as HTMLInputElement).value).toBe('3');
    expect(run.posts).toHaveLength(1);
    run.update({ order: { ...order, revision: 'rev-4' } });
    await waitFor(() => expect((screen.getByRole('button', { name: 'Отгрузить по количеству' }) as HTMLButtonElement).disabled).toBe(false));
    fireEvent.click(screen.getByRole('button', { name: 'Отгрузить по количеству' }));
    await waitFor(() => expect(run.done).toHaveBeenCalledWith('Отгружено: 3. С VIN: 1. Без VIN: 2.'));
    expect(new Headers(run.posts[1].headers).get('If-Match')).toBe('"rev-4"');
    expect(JSON.parse(String(run.posts[1].body)).lines[0].vins).toEqual([vin(1)]);
  });

  it('rejects duplicate VINs across different shipment lines', async () => {
    const shown = { ...order, terms: { ...order.terms, lines: [...order.terms.lines, { ...order.terms.lines[0], lineId: 'l-2', modelId: 'm-2' }] } };
    const run = setup({ shown, action: { kind: 'ship-quantity' } });
    for (const button of screen.getAllByRole('button', { name: 'Добавить VIN' })) fireEvent.click(button);
    for (const input of screen.getAllByLabelText('VIN 1')) fireEvent.change(input, { target: { value: vin(1) } });
    fireEvent.click(screen.getByRole('button', { name: 'Отгрузить по количеству' }));
    await screen.findByText('Повторяющийся VIN в разных строках заказа');
    expect(run.posts).toHaveLength(0);
  });

  it('does not re-enable a successful shipment when refresh fails', async () => {
    const run = setup({ action: { kind: 'ship-quantity' } });
    run.invalidate.mockRejectedValue(new Error('Refresh failed'));
    fireEvent.click(screen.getByRole('button', { name: 'Отгрузить по количеству' }));
    await waitFor(() => expect(run.done).toHaveBeenCalledWith(expect.stringContaining('Не удалось обновить списки')));
    fireEvent.click(screen.getByRole('button', { name: 'Отправка…' }));
    expect(run.posts).toHaveLength(1);
  });

  it('preserves the allocated VIN shipment payload and legacy receipt explanation', async () => {
    const shown = { ...order, hasReceivingWarehouse: false, allocations: [{ orderLineId: 'l-1', vehicleId: 'v-1', vin: vin(1), status: 'allocated', shipmentId: null }] };
    const run = setup({ shown, action: { kind: 'ship-allocated' } });
    expect(screen.getByText(/сохранена ручная приёмка/)).toBeTruthy();
    fireEvent.click(screen.getByLabelText(vin(1)));
    await choose('Маршрут', 'Автомобиль в пути');
    fireEvent.click(screen.getByRole('button', { name: 'Отгрузить назначенные VIN' }));
    await waitFor(() => expect(run.done).toHaveBeenCalledWith('Отгружено: 1. С VIN: 1. Без VIN: 0.'));
    expect(JSON.parse(String(run.posts[0].body))).toEqual({ vehicleIds: ['v-1'], route: 'in-transit' });
  });
});

it('autocomplete warehouse changes and all sentinel reset page offset while preserving selected VINs', async () => {
  const run = setup({ stock: params => params.get('warehouseId') ? [] : Array.from({ length: 50 }, (_, index) => vehicle(Number(params.get('offset')) + index)) });
  fireEvent.click(await screen.findByLabelText(`Выбрать ${vin(1)}`));
  fireEvent.click(screen.getByRole('button', { name: 'Следующая страница' }));
  await screen.findByLabelText(`Выбрать ${vin(51)}`);
  await choose('Склад', 'Второй');
  await waitFor(() => expect(run.reads.at(-1)?.get('warehouseId')).toBe('w-2'));
  expect(run.reads.at(-1)?.get('offset')).toBe('0');
  expect(screen.getByRole('button', { name: `Убрать ${vin(1)}` })).toBeTruthy();
  await choose('Склад', 'Все склады');
  await screen.findByLabelText(`Выбрать ${vin(1)}`);
  expect((screen.getByRole('combobox', { name: 'Склад' }) as HTMLInputElement).value).toBe('Все склады');
  expect(run.reads.at(-1)?.has('warehouseId')).toBe(false);
  expect(run.reads.at(-1)?.get('offset')).toBe('0');
});

it.each(['Цвет кузова', 'Цвет салона'])('autocomplete incoming version resets colors and edited singleton %s blocks submit', async label => {
  const shown = { ...order, terms: { ...order.terms, lines: [{ ...order.terms.lines[0]!, modelSpecificationVersion: '', exteriorColor: '', interiorColor: '' }] } };
  const run = setup({ shown, action: { kind: 'ship-quantity' } });
  await choose('Версия входящей спецификации', 'Версия 1');
  await choose('Цвет кузова', 'Синий');
  await choose('Цвет салона', 'Бежевый');
  await choose('Версия входящей спецификации', 'Версия 2');
  expect((screen.getByRole('combobox', { name: 'Цвет кузова' }) as HTMLInputElement).value).toBe('Красный');
  expect((screen.getByRole('combobox', { name: 'Цвет салона' }) as HTMLInputElement).value).toBe('Серый');
  const button = screen.getByRole('button', { name: 'Отгрузить по количеству' });
  fireEvent.change(screen.getByRole('combobox', { name: label }), { target: { value: 'Unknown' } });
  fireEvent.click(button);
  expect(run.posts).toHaveLength(0);
  await choose(label, label === 'Цвет кузова' ? 'Красный' : 'Серый');
  fireEvent.change(screen.getByRole('combobox', { name: 'Версия входящей спецификации' }), { target: { value: 'Unknown' } });
  fireEvent.click(button);
  expect(run.posts).toHaveLength(0);
  expect((screen.getByRole('combobox', { name: 'Цвет кузова' }) as HTMLInputElement).value).toBe('Выберите цвет');
  await choose('Версия входящей спецификации', 'Версия 2');
  fireEvent.click(button);
  await waitFor(() => expect(run.posts).toHaveLength(1));
  expect(JSON.parse(String(run.posts[0].body)).lines[0]).toEqual({ orderLineId: 'l-1', quantity: '4', vins: [], modelSpecificationVersion: '2', exteriorColor: 'Красный', interiorColor: 'Серый' });
});

it.each(['ship-quantity', 'ship-allocated'] as const)('autocomplete route edits block %s until a route is selected', async kind => {
  const shown = { ...order, allocations: [{ orderLineId: 'l-1', vehicleId: 'v-1', vin: vin(1), status: 'allocated', shipmentId: null }] };
  const run = setup({ shown, action: { kind } });
  if (kind === 'ship-allocated') fireEvent.click(screen.getByLabelText(vin(1)));
  fireEvent.change(screen.getByRole('combobox', { name: 'Маршрут' }), { target: { value: 'Unknown' } });
  const button = screen.getByRole('button', { name: kind === 'ship-allocated' ? 'Отгрузить назначенные VIN' : 'Отгрузить по количеству' });
  fireEvent.click(button);
  expect(run.posts).toHaveLength(0);
  expect(screen.getByText('Выберите маршрут')).toBeTruthy();
  await choose('Маршрут', 'Автомобиль в пути');
  fireEvent.click(button);
  await waitFor(() => expect(run.posts).toHaveLength(1));
  expect(JSON.parse(String(run.posts[0].body)).route).toBe('in-transit');
});
