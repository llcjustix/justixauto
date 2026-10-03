// @vitest-environment jsdom
import { QueryClient, QueryClientProvider } from '@tanstack/react-query';
import { cleanup, fireEvent, render, screen, waitFor, within } from '@testing-library/react';
import { afterEach, expect, it, vi } from 'vitest';
import { WarehouseDialog, VehicleDialog } from './pages/inventory';
import { ReceiptIdentification } from './receipt-identification';

vi.mock('./data', async original => ({ ...(await original<typeof import('./data')>()), useBranches: () => ({ data: [] }) }));
const v1 = { version: '1', make: 'BYD', model: 'Song', variant: 'Plus', exteriorColors: ['Белый', 'Синий'], interiorColors: ['Чёрный', 'Бежевый'] };
const v2 = { ...v1, version: '2', exteriorColors: ['Красный'], interiorColors: ['Серый'] };
const model = { id: 'm-1', specification: v2, versions: [v1, v2] };
const batch = { id: 'b-1', warehouseId: 'w-1', modelId: 'm-1', modelSpecificationVersion: '1', exteriorColor: 'Белый', interiorColor: '', confirmedQuantity: '3', identifiedCount: '1', unidentifiedCount: '2', revision: '1' };
const json = (body: unknown, status = 200) => new Response(JSON.stringify(body), { status, headers: { 'Content-Type': 'application/json' } });
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


function setup(options: { identification?: boolean; vehicle?: boolean; facts?: Partial<typeof batch>; detail?: () => Response | Promise<Response>; post?: () => Response | Promise<Response> } = {}) {
  const facts = { ...batch, ...options.facts };
  const posts: { body: any; headers: Headers }[] = [];
  const done = vi.fn();
  const close = vi.fn();
  const client = new QueryClient({ defaultOptions: { queries: { retry: false } } });
  vi.stubGlobal('fetch', vi.fn(async (input: RequestInfo | URL, init: RequestInit = {}) => {
    const url = String(input);
    if (init.method === 'POST') { posts.push({ body: JSON.parse(String(init.body)), headers: new Headers(init.headers) }); return options.post?.() ?? json({ data: {} }, 201); }
    if (url.endsWith('/vehicle-models/m-1')) return options.detail?.() ?? json({ data: model });
    if (url.includes('/vehicle-models')) return json({ items: [{ ...model, specification: v1 }, { ...model, id: 'm-2', specification: { ...v2, model: 'Other' } }] });
    if (url.includes('/vehicle-units/')) return json({ data: { vehicle: { id: 'v-1', vin: 'LGXC16DF0P0000001', exteriorColor: 'Синий', interiorColor: '', placement: null }, specification: { ...v2, exteriorColor: 'Красный', interiorColor: 'Серый' }, history: [] } });
    if (url.includes('/warehouses/w-1/inventory')) return json({ data: {
      warehouse: { id: 'w-1', name: 'Warehouse', country: { label: 'Узбекистан' }, city: 'Ташкент', address: 'Address', capacity: '100', occupied: '5', free: '95', revision: 'rev-1' }, vehicles: [],
      unidentifiedBatches: [facts, { ...batch, id: 'b-2', exteriorColor: 'Синий', interiorColor: 'Бежевый' }],
    } });
    return json({ items: [] });
  }));
  render(<QueryClientProvider client={client}>{options.vehicle ? <VehicleDialog id="v-1" onClose={close} /> : options.identification ? <ReceiptIdentification batch={facts} modelName="BYD Song Plus" refresh={[]} onBack={close} onDone={done} /> : <WarehouseDialog id="w-1" onClose={close} />}</QueryClientProvider>);
  return { posts, done, close };
}

it.each(['identified', 'unidentified'])('colors manual receipt %s pins one pair and preserves the single dialog and submit guard', async mode => {
  let release!: (response: Response) => void;
  const run = setup({ post: () => new Promise(resolve => { release = resolve; }) });
  fireEvent.click(await screen.findByRole('button', { name: 'Принять автомобили' }));
  await openChoices('Модель', 'BYD Song Plus');
  await choose('Модель', 'BYD Song Plus');
  await openChoices('Цвет кузова', 'Синий');
  expect(screen.getAllByRole('dialog')).toHaveLength(1);
  expect((screen.getByRole('button', { name: 'Принять автомобили' }) as HTMLButtonElement).disabled).toBe(true);
  await choose('Цвет кузова', 'Синий');
  await choose('Цвет салона', 'Бежевый');
  fireEvent.change(screen.getByLabelText('Дата приёмки'), { target: { value: '2026-10-02T10:00' } });
  await choose('Приёмка', mode === 'identified' ? 'С VIN' : 'Количество, VIN позже');
  if (mode === 'identified') fireEvent.change(screen.getByLabelText('VIN 1'), { target: { value: ' lgxc16df0p0000001 ' } });
  else fireEvent.change(screen.getByLabelText('Количество (без VIN)'), { target: { value: '2' } });
  const form = screen.getByRole('button', { name: 'Принять автомобили' }).closest('form')!;
  fireEvent.click(screen.getByRole('button', { name: 'Принять автомобили' }));
  fireEvent.submit(form);
  expect(run.posts).toHaveLength(1);
  expect(run.posts[0]?.body).toEqual({ modelId: 'm-1', modelSpecificationVersion: '1', exteriorColor: 'Синий', interiorColor: 'Бежевый', stock: mode === 'identified' ? { mode, vins: ['LGXC16DF0P0000001'] } : { mode, quantity: '2' }, receivedAt: new Date('2026-10-02T10:00').toISOString() });
  expect(run.posts[0]?.headers.get('If-Match')).toBe('"rev-1"');
  release(json({ data: {} }, 201));
  await screen.findByText('Автомобили приняты. Кузов: Синий · Салон: Бежевый · Версия: 1.');
  expect(screen.getAllByRole('dialog')).toHaveLength(1);
});

it('colors manual receipt blocks loading failed or absent exact version, retries and clears colors when model changes', async () => {
  let release!: (value: Response) => void;
  let calls = 0;
  const run = setup({ detail: () => ++calls === 1 ? new Promise(resolve => { release = resolve; }) : calls === 2 ? json({ error: { message: 'Offline' } }, 500) : json({ data: model }) });
  fireEvent.click(await screen.findByRole('button', { name: 'Принять автомобили' }));
  await openChoices('Модель', 'BYD Song Plus');
  await choose('Модель', 'BYD Song Plus');
  await waitFor(() => expect(release).toBeDefined());
  expect((screen.getByRole('button', { name: 'Принять автомобили' }) as HTMLButtonElement).disabled).toBe(true);
  release(json({ data: { ...model, versions: [v2] } }));
  fireEvent.click(await screen.findByRole('button', { name: 'Повторить загрузку спецификации' }));
  await waitFor(() => expect(calls).toBe(2));
  await waitFor(() => expect(screen.queryByText('Загрузка спецификации…')).toBeNull());
  fireEvent.click(screen.getByRole('button', { name: 'Повторить загрузку спецификации' }));
  await openChoices('Цвет кузова', 'Синий');
  await choose('Цвет кузова', 'Синий');
  await choose('Цвет салона', 'Бежевый');
  await choose('Модель', 'BYD Other Plus');
  expect((screen.getByRole('combobox', { name: 'Цвет кузова' }) as HTMLInputElement).value).toBe('Выберите цвет');
  expect((screen.getByRole('button', { name: 'Принять автомобили' }) as HTMLButtonElement).disabled).toBe(true);
  expect(run.posts).toHaveLength(0);
});

it.each(['exterior', 'interior'])('colors identification resolves only missing %s from pinned v1 and preserves known batch facts', async missing => {
  const run = setup({ identification: true, facts: missing === 'exterior' ? { exteriorColor: '', interiorColor: 'ЧЁРНЫЙ' } : { exteriorColor: 'БЕЛЫЙ', interiorColor: '' } });
  const label = missing === 'exterior' ? 'Цвет кузова' : 'Цвет салона';
  const selected = missing === 'exterior' ? 'Синий' : 'Бежевый';
  await openChoices(label, selected);
  expect(screen.queryByLabelText(missing === 'exterior' ? 'Цвет салона' : 'Цвет кузова')).toBeNull();
  expect(screen.queryByRole('option', { name: 'Красный' })).toBeNull();
  fireEvent.change(screen.getByLabelText('VIN 1'), { target: { value: 'LGXC16DF0P0000001' } });
  expect((screen.getByRole('button', { name: 'Сохранить VIN' }) as HTMLButtonElement).disabled).toBe(true);
  await choose(label, selected);
  fireEvent.click(screen.getByRole('button', { name: 'Сохранить VIN' }));
  await waitFor(() => expect(run.done).toHaveBeenCalledOnce());
  expect(run.posts[0]?.body).toEqual({ atomic: true, items: [{ modelId: 'm-1', vin: 'LGXC16DF0P0000001' }], [missing === 'exterior' ? 'exteriorColor' : 'interiorColor']: selected });
});

it('colors identification keeps draft through failed missing-version retry and blocks until exact version is loaded', async () => {
  let calls = 0;
  const run = setup({ identification: true, detail: () => ++calls === 1 ? json({ data: { ...model, versions: [v2] } }) : calls === 2 ? json({ error: { message: 'Offline' } }, 500) : json({ data: model }) });
  fireEvent.change(await screen.findByLabelText('VIN 1'), { target: { value: 'LGXC16DF0P0000001' } });
  fireEvent.click(await screen.findByRole('button', { name: 'Повторить загрузку спецификации' }));
  await waitFor(() => expect(calls).toBe(2));
  await waitFor(() => expect((screen.getByRole('button', { name: 'Повторить загрузку спецификации' }) as HTMLButtonElement).closest('fieldset')?.disabled).toBe(false));
  expect((screen.getByRole('button', { name: 'Сохранить VIN' }) as HTMLButtonElement).disabled).toBe(true);
  fireEvent.click(screen.getByRole('button', { name: 'Повторить загрузку спецификации' }));
  await openChoices('Цвет салона', 'Бежевый');
  expect((screen.getByLabelText('VIN 1') as HTMLInputElement).value).toBe('LGXC16DF0P0000001');
  await choose('Цвет салона', 'Бежевый');
  fireEvent.click(screen.getByRole('button', { name: 'Сохранить VIN' }));
  await waitFor(() => expect(run.posts).toHaveLength(1));
});

it('colors warehouse separates same-model batch pairs and opens their exact identification facts', async () => {
  setup();
  const pair = await screen.findByText('Кузов: Синий · Салон: Бежевый · Версия: 1');
  const row = pair.closest('tr')!;
  fireEvent.click(within(row).getByRole('button', { name: 'Ввести VIN' }));
  await screen.findByText(/Партия b-2/);
  expect(screen.getByText('Кузов: Синий · Салон: Бежевый · Версия: 1')).toBeTruthy();
  expect(screen.queryByLabelText('Цвет кузова')).toBeNull();
});

it('colors vehicle detail displays actual and independently unknown fields instead of catalogue colors', async () => {
  setup({ vehicle: true });
  expect(await screen.findByText(/Кузов: Синий · Салон: Не указан/)).toBeTruthy();
  expect(screen.queryByText(/Красный/)).toBeNull();
});

it.each(['Цвет кузова', 'Цвет салона'])('manual receipt keeps singleton defaults but rejects edited %s until a real pick', async label => {
  const run = setup({ detail: () => json({ data: { ...model, versions: [{ ...v2, version: '1' }] } }) });
  fireEvent.click(await screen.findByRole('button', { name: 'Принять автомобили' }));
  await choose('Модель', 'BYD Song Plus');
  const input = await screen.findByRole('combobox', { name: label });
  const color = label === 'Цвет кузова' ? 'Красный' : 'Серый';
  await waitFor(() => expect((input as HTMLInputElement).value).toBe(color));
  fireEvent.change(screen.getByLabelText('Дата приёмки'), { target: { value: '2026-10-02T10:00' } });
  fireEvent.change(screen.getByLabelText('VIN 1'), { target: { value: 'LGXC16DF0P0000001' } });
  fireEvent.change(input, { target: { value: 'Unknown' } });
  const button = screen.getByRole('button', { name: 'Принять автомобили' });
  expect((button as HTMLButtonElement).disabled).toBe(true);
  fireEvent.submit(button.closest('form')!);
  expect(run.posts).toHaveLength(0);
  await choose(label, color);
  fireEvent.click(button);
  await waitFor(() => expect(run.posts).toHaveLength(1));
  expect(run.posts[0]?.body).toMatchObject({ exteriorColor: 'Красный', interiorColor: 'Серый', modelSpecificationVersion: '1' });
});

it('manual receipt rejects edited model and empty mode without losing VIN and quantity drafts', async () => {
  const run = setup({ detail: () => json({ data: { ...model, versions: [{ ...v2, version: '1' }] } }) });
  fireEvent.click(await screen.findByRole('button', { name: 'Принять автомобили' }));
  await choose('Модель', 'BYD Song Plus');
  await waitFor(() => expect((screen.getByRole('combobox', { name: 'Цвет кузова' }) as HTMLInputElement).value).toBe('Красный'));
  fireEvent.change(screen.getByLabelText('Дата приёмки'), { target: { value: '2026-10-02T10:00' } });
  fireEvent.change(screen.getByLabelText('VIN 1'), { target: { value: 'LGXC16DF0P0000001' } });
  const form = screen.getByRole('button', { name: 'Принять автомобили' }).closest('form')!;
  fireEvent.change(screen.getByRole('combobox', { name: 'Модель' }), { target: { value: 'Unknown' } });
  fireEvent.submit(form);
  expect(run.posts).toHaveLength(0);
  expect(screen.queryByRole('combobox', { name: 'Цвет кузова' })).toBeNull();
  await choose('Модель', 'BYD Song Plus');
  await waitFor(() => expect((screen.getByRole('combobox', { name: 'Цвет кузова' }) as HTMLInputElement).value).toBe('Красный'));
  await choose('Приёмка', 'Количество, VIN позже');
  fireEvent.change(screen.getByLabelText('Количество (без VIN)'), { target: { value: '7' } });
  fireEvent.change(screen.getByRole('combobox', { name: 'Приёмка' }), { target: { value: 'Unknown' } });
  fireEvent.submit(form);
  expect(screen.getByText('Выберите режим приёмки')).toBeTruthy();
  expect(run.posts).toHaveLength(0);
  expect(screen.queryByLabelText('Количество (без VIN)')).toBeNull();
  await choose('Приёмка', 'С VIN');
  expect((screen.getByLabelText('VIN 1') as HTMLInputElement).value).toBe('LGXC16DF0P0000001');
  await choose('Приёмка', 'Количество, VIN позже');
  expect((screen.getByLabelText('Количество (без VIN)') as HTMLInputElement).value).toBe('7');
  fireEvent.submit(form);
  await waitFor(() => expect(run.posts).toHaveLength(1));
  expect(run.posts[0]?.body.stock).toEqual({ mode: 'unidentified', quantity: '7' });
});

it.each(['Цвет кузова', 'Цвет салона'])('identification rejects edited singleton %s and keeps independently known history', async label => {
  const missingExterior = label === 'Цвет кузова';
  const run = setup({ identification: true, facts: { modelSpecificationVersion: '2', exteriorColor: missingExterior ? '' : 'БЕЛЫЙ', interiorColor: missingExterior ? 'ЧЁРНЫЙ' : '' } });
  const color = missingExterior ? 'Красный' : 'Серый';
  const input = await screen.findByRole('combobox', { name: label });
  await waitFor(() => expect((input as HTMLInputElement).value).toBe(color));
  expect(screen.queryByRole('combobox', { name: missingExterior ? 'Цвет салона' : 'Цвет кузова' })).toBeNull();
  fireEvent.change(screen.getByLabelText('VIN 1'), { target: { value: 'LGXC16DF0P0000001' } });
  fireEvent.change(input, { target: { value: 'Unknown' } });
  expect((screen.getByRole('button', { name: 'Сохранить VIN' }) as HTMLButtonElement).disabled).toBe(true);
  fireEvent.click(screen.getByRole('button', { name: 'Сохранить VIN' }));
  expect(run.posts).toHaveLength(0);
  await choose(label, color);
  fireEvent.click(screen.getByRole('button', { name: 'Сохранить VIN' }));
  await waitFor(() => expect(run.posts).toHaveLength(1));
  expect(run.posts[0]?.body).toEqual({ atomic: true, items: [{ vin: 'LGXC16DF0P0000001', modelId: 'm-1' }], [missingExterior ? 'exteriorColor' : 'interiorColor']: color });
});
