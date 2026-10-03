// @vitest-environment jsdom
import { QueryClient, QueryClientProvider } from '@tanstack/react-query';
import { SessionGate, useSession } from '@justixauto/kit';
import { act, cleanup, fireEvent, render, screen, waitFor } from '@testing-library/react';
import { afterEach, describe, expect, it, vi } from 'vitest';
import { SaleForm } from './sale-form';
import type { SaleSourceSeed } from './sale-offer-source';

const m = (amountMinor = '12345') => ({ amountMinor, currency: 'USD' });
const car = { id: 'car', vin: 'VIN', modelId: 'model', modelSpecificationVersion: 'v1', exteriorColor: 'White', interiorColor: 'Black', placement: { warehouseId: 'w' }, reserved: false };
const line = { lineId: 'line', modelId: 'model', modelSpecificationVersion: 'v1', exteriorColor: 'White', interiorColor: 'Black', quantity: '20', unitPrice: m() };
const version = { id: 'published', number: 1, publishedAt: 'today', terms: { lines: [line] }, total: m('999999') };
const offer = () => ({ id: 'supplier', supplier: { id: 'supplier-company', name: 'Supplier' }, status: 'published', publishedVersion: version });
const ownOffer = () => ({ ...offer(), id: 'own', supplier: { id: 'company', name: 'Own' }, versions: [version, { ...version, id: 'draft', number: 2, publishedAt: null }] });
const listing = () => ({ id: 'listing', text: 'Client offer', status: 'draft', vehicleId: 'car', askingPrice: m(), revision: '1' });
const response = (body: unknown, status = 200) => new Response(JSON.stringify(body), { status, headers: { 'Content-Type': 'application/json' } });
const fail = () => response({ error: { message: 'Нет доступа к предложениям' } }, 403);
function SwitchCompany() { const session = useSession(); return <button onClick={() => void session.refresh()}>Switch context</button>; }
function harness(seed?: SaleSourceSeed) {
  const state = { gets: [] as string[], posts: [] as any[], supplier: offer(), own: ownOffer(), listing: listing(), vehicles: [car],
    permissionFailure: false, inventoryFailure: false, empty: false, company: 'company', revision: '1',
    pendingDetail: undefined as (() => Promise<Response>) | undefined, pendingInventory: undefined as (() => Promise<Response>) | undefined,
  };
  vi.stubGlobal('fetch', vi.fn(async (input: RequestInfo | URL, init: RequestInit = {}) => {
    const url = String(input);
    if (url.endsWith('/identity/session')) return response({ data: {
      user: { id: 'user', displayName: 'User', status: 'active' }, roles: [], permissions: [],
      context: { revision: state.revision, companyId: state.company, branchScope: { mode: 'ALL', branchIds: [] } },
      accessibleCompanies: [{ id: 'company', name: 'Company', access: 'full' }, { id: 'other', name: 'Other', access: 'full' }], setup: { next: 'none' },
    } });
    if (init.method === 'POST') { const body = JSON.parse(String(init.body)); state.posts.push(body); return response({ data: { id: 'saved', price: body.price } }); }
    state.gets.push(url);
    if (url.includes('/commerce/offers?')) return state.permissionFailure ? fail() : response({ items: state.empty ? [] : [url.includes('scope=own') ? state.own : state.supplier] });
    if (url.includes('/commerce/offers/')) return state.pendingDetail ? state.pendingDetail() : state.permissionFailure ? fail() : response({ data: url.endsWith('/own') ? state.own : state.supplier });
    if (url.includes('/retail/listings?')) return response({ items: state.empty ? [] : [state.listing] });
    if (url.includes('/retail/listings/')) return response({ data: state.listing });
    if (url.includes('/vehicle-units?')) return state.pendingInventory ? state.pendingInventory() : state.inventoryFailure ? fail() : response({ items: state.vehicles });
    if (url.includes('/vehicle-models?')) return response({ items: [{ id: 'model', specification: { make: 'Make', model: 'Model', variant: '' } }] });
    if (url.includes('/retail/customers?')) return response({ items: [{ id: 'customer', displayName: 'Customer' }] });
    if (url.includes('/retail/leads?')) return response({ items: [{ id: 'lead', stage: 'qualified', customer: { displayName: 'Lead' } }] });
    if (url.includes('/branches?')) return response({ items: [{ id: 'branch', name: 'Branch' }] });
    throw new Error(`Unexpected ${url}`);
  }));
  const client = new QueryClient({ defaultOptions: { queries: { retry: false, staleTime: Infinity }, mutations: { retry: false } } });
  const close = vi.fn();
  render(<QueryClientProvider client={client}><SessionGate title="Test"><SwitchCompany /><SaleForm sourceSeed={seed} onClose={close} /></SessionGate></QueryClientProvider>);
  return { state, client, close };
}
const click = (name: string) => fireEvent.click(screen.getByRole('button', { name, exact: true }));
const change = (name: string, value: string) => fireEvent.change(screen.getByLabelText(name, { selector: 'input' }), { target: { value } });
const value = (name: string) => (screen.getByLabelText(name, { selector: 'input' }) as HTMLInputElement).value;
function pick(name: string, option: string | RegExp) {
  screen.getAllByRole('combobox').forEach(input => fireEvent.blur(input));
  fireEvent.focus(screen.getByRole('combobox', { name, exact: true }));
  fireEvent.click(screen.getByRole('option', { name: option, exact: true }));
}
async function ready() {
  await screen.findByLabelText('Филиал'); fireEvent.focus(screen.getByLabelText('Филиал'));
  fireEvent.click(await screen.findByRole('option', { name: 'Branch' }));
}
function fields() { pick('Клиент', 'Customer'); pick('Автомобиль', /· VIN ·/); pick('Способ оформления', 'Наличные / перевод'); change('Цена', '777.77'); }
function force() { fireEvent.submit(screen.getByLabelText('Цена').closest('form')!); }
async function apply() { await waitFor(() => expect((screen.getByRole('button', { name: 'Применить предложение' }) as HTMLButtonElement).disabled).toBe(false)); click('Применить предложение'); }
async function chooseSupplier() { click('Выбрать предложение'); fireEvent.focus(screen.getByLabelText('Предложение')); fireEvent.click(await screen.findByRole('option', { name: 'Supplier · supplier' })); }
afterEach(() => { cleanup(); vi.unstubAllGlobals(); });

describe('sale offer form', () => {
  it('manual creation never reads commerce/listings and keeps the POST whitelist', async () => {
    const { state } = harness(); await ready(); fields(); state.permissionFailure = true;
    click('Создать продажу'); await screen.findByText(/Продажа создана/);
    expect(state.gets.some(path => /commerce|listings/.test(path))).toBe(false);
    expect(state.posts).toEqual([{ customerId: 'customer', leadId: null, vehicleId: 'car', branchId: 'branch', paymentScheme: 'cash', price: m('77777') }]);
  });
  it.each([
    [{ kind: 'supplier-offer', id: 'supplier', versionId: 'published' }, 'Предложение поставщика'],
    [{ kind: 'own-offer', id: 'own', versionId: 'draft' }, 'Своё предложение партнёрам'],
    [{ kind: 'own-listing', id: 'listing' }, 'Своё предложение клиентам'],
  ] as const)('applies %s explicitly, keeps price editable and preflights without adding source fields', async (seed, label) => {
    const { state } = harness(seed); await ready(); fields(); pick('Лид (необязательно)', /Lead ·/);
    expect(value('Цена')).toBe('777.77'); force(); expect(state.posts).toHaveLength(0);
    await apply(); expect(value('Цена')).toBe('123.45'); expect(screen.getByLabelText('Применённое предложение').textContent).toContain(label);
    if (seed.kind !== 'supplier-offer') expect(screen.getByLabelText('Применённое предложение').textContent).toContain('Черновик');
    change('Цена', '900719925474099.31'); click('Создать продажу'); await screen.findByText(/Продажа создана/);
    expect(state.posts).toEqual([{ customerId: 'customer', leadId: 'lead', vehicleId: 'car', branchId: 'branch', paymentScheme: 'cash', price: m('90071992547409931') }]);
    expect(state.gets.filter(path => path.includes('/vehicle-units?'))).toHaveLength(3); // initial, preflight, success refresh
  });
  it('requires one multi-line choice, clears typed line/source IDs and incompatible VIN, and explicit detach restores manual', async () => {
    const { state } = harness(); state.supplier.publishedVersion = { ...version, terms: { lines: [line, { ...line, lineId: 'blue', exteriorColor: 'Blue' }] } };
    await ready(); fields(); await chooseSupplier(); await screen.findByLabelText('Строка предложения');
    force(); expect(state.posts).toHaveLength(0); pick('Строка предложения', /Строка 2$/); await apply();
    expect(value('Автомобиль')).toBe(''); expect(screen.getByText(/На складе нет подходящего VIN/)).toBeTruthy();
    click('Изменить / применить заново'); pick('Строка предложения', /Строка 1$/); change('Строка предложения', 'typed'); force(); expect(state.posts).toHaveLength(0);
    change('Предложение', 'typed'); force(); expect(state.posts).toHaveLength(0);
    click('Продолжить вручную'); expect(value('Цена')).toBe('123.45'); pick('Автомобиль', /· VIN ·/); click('Создать продажу'); await screen.findByText(/Продажа создана/);
  });
  it('isolates source permission failure, retries, shows empty and detaches without blocking manual', async () => {
    const { state } = harness(); await ready(); fields(); state.permissionFailure = true; click('Выбрать предложение');
    await screen.findByText(/Не удалось загрузить предложение/); force(); expect(state.posts).toHaveLength(0);
    state.permissionFailure = false; state.empty = true; click('Обновить предложения'); await screen.findByText(/Нет доступных предложений/);
    click('Продолжить вручную'); click('Создать продажу'); await screen.findByText(/Продажа создана/);
  });
  it('keeps edited price on refresh, invalidates changed listing, and reapplication alone copies its new price', async () => {
    const { state, client } = harness({ kind: 'own-listing', id: 'listing' }); state.listing.status = 'published'; await ready(); fields(); await apply();
    change('Цена', '888'); await act(async () => { await client.invalidateQueries({ queryKey: ['sale-source'] }); }); expect(value('Цена')).toBe('888');
    state.listing = { ...state.listing, revision: '2', askingPrice: m('45678') };
    await act(async () => { await client.invalidateQueries({ queryKey: ['sale-source'] }); }); expect(value('Цена')).toBe('888');
    force(); expect(state.posts).toHaveLength(0); click('Изменить / применить заново'); await apply(); expect(value('Цена')).toBe('456.78');
    change('Цена', '999'); click('Отключить предложение'); await act(async () => { await client.invalidateQueries({ queryKey: ['sale-source'] }); });
    expect(value('Цена')).toBe('999'); expect(screen.queryByLabelText('Применённое предложение')).toBeNull();
  });
  it.each(['withdrawn', 'stock', 'failure'])('fails closed on %s during submit preflight and blocks duplicate submit and close', async mode => {
    const { state, close } = harness({ kind: 'supplier-offer', id: 'supplier', versionId: 'published' }); await ready(); fields(); await apply();
    let finish!: (value: Response) => void; state.pendingDetail = () => new Promise(resolve => { finish = resolve; });
    if (mode === 'stock') state.vehicles = [{ ...car, reserved: true }];
    click('Создать продажу'); force(); click('Закрыть'); expect(close).not.toHaveBeenCalled(); expect(state.posts).toHaveLength(0);
    expect((screen.getByLabelText('Цена') as HTMLInputElement).disabled).toBe(true);
    await act(async () => { finish(mode === 'failure' ? fail() : response({ data: { ...state.supplier, status: mode === 'withdrawn' ? 'withdrawn' : 'published' } })); });
    expect(state.posts).toHaveLength(0); force(); expect(state.posts).toHaveLength(0);
  });
  it('ignores late detail after detach and after switching source kind', async () => {
    const { state } = harness(); await ready(); fields(); let finish!: (value: Response) => void;
    state.pendingDetail = () => new Promise(resolve => { finish = resolve; }); await chooseSupplier();
    expect(screen.getByText('Загрузка предложений…')).toBeTruthy(); click('Продолжить вручную');
    await act(async () => { finish(response({ data: state.supplier })); }); expect(value('Цена')).toBe('777.77');
    click('Выбрать предложение'); pick('Источник цены', 'Своё предложение клиентам');
    fireEvent.focus(screen.getByLabelText('Предложение')); fireEvent.click(await screen.findByRole('option', { name: 'Client offer · Черновик' })); await apply();
    expect(screen.getByLabelText('Применённое предложение').textContent).toContain('Своё предложение клиентам');
  });
  it('blocks a typed VIN and unhealthy inventory even on forced Submit', async () => {
    const { state, client } = harness(); await ready(); fields(); change('Автомобиль', 'VIN'); force(); expect(state.posts).toHaveLength(0);
    change('Автомобиль', ''); pick('Автомобиль', /· VIN ·/); state.inventoryFailure = true;
    await act(async () => { await client.invalidateQueries({ queryKey: ['vehicles'] }); }); force(); expect(state.posts).toHaveLength(0);
  });
  it('blocks while stock reloads and discards a late response for another source kind', async () => {
    const { state, client } = harness(); await ready(); fields();
    let finishStock!: (value: Response) => void;
    state.pendingInventory = () => new Promise(resolve => { finishStock = resolve; });
    let reload!: Promise<void>;
    act(() => { reload = client.invalidateQueries({ queryKey: ['vehicles'] }); }); force(); expect(state.posts).toHaveLength(0);
    await act(async () => { finishStock(response({ items: [car] })); await reload; }); state.pendingInventory = undefined;
    let finishDetail!: (value: Response) => void;
    state.pendingDetail = () => new Promise(resolve => { finishDetail = resolve; }); await chooseSupplier();
    pick('Источник цены', 'Своё предложение клиентам');
    fireEvent.focus(screen.getByLabelText('Предложение')); fireEvent.click(await screen.findByRole('option', { name: 'Client offer · Черновик' }));
    await act(async () => { finishDetail(response({ data: state.supplier })); });
    await apply(); expect(screen.getByLabelText('Применённое предложение').textContent).toContain('Своё предложение клиентам');
  });
  it('requires an explicit latest own-version choice after selection refresh', async () => {
    const { state, client } = harness({ kind: 'own-offer', id: 'own', versionId: 'draft' }); await ready(); fields();
    await screen.findByText(/Цена за один автомобиль/);
    state.own.versions.push({ ...version, id: 'new-draft', number: 3, publishedAt: null });
    await act(async () => { await client.invalidateQueries({ queryKey: ['sale-source'] }); });
    await screen.findByText(/Показанная версия изменилась/); force(); expect(state.posts).toHaveLength(0); expect(value('Цена')).toBe('777.77');
    click('Выбрать актуальную версию'); await apply(); expect(screen.getByLabelText('Применённое предложение').textContent).toContain('Версия №3');
  });
  it('cancels selection without changing values and rejects obsolete company preflight', async () => {
    const { state } = harness({ kind: 'supplier-offer', id: 'supplier', versionId: 'published' }); await ready(); fields(); await apply(); change('Цена', '888');
    click('Изменить / применить заново'); change('Предложение', 'typing'); click('Отменить выбор предложения'); expect(value('Цена')).toBe('888');
    let finish!: (value: Response) => void; state.pendingDetail = () => new Promise(resolve => { finish = resolve; }); click('Создать продажу');
    state.company = 'other'; state.revision = '2'; click('Switch context'); await waitFor(() => expect(value('Цена')).toBe(''));
    await act(async () => { finish(response({ data: state.supplier })); }); expect(state.posts).toHaveLength(0);
    expect(screen.queryByLabelText('Применённое предложение')).toBeNull();
  });
});
