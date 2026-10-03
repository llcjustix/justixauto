// @vitest-environment jsdom
import { QueryClient, QueryClientProvider } from '@tanstack/react-query';
import { MemoryRouter } from 'react-router-dom';
import { SessionGate } from '@justixauto/kit';
import { cleanup, fireEvent, render, screen, waitFor, within } from '@testing-library/react';
import { afterEach, describe, expect, it, vi } from 'vitest';
import { NewSale } from './sale-form';
import { SalesPage } from './pages/sales';
import { calculateInstallment } from './installment-calculator';

const response = (body: unknown, status = 200) => new Response(JSON.stringify(body), { status, headers: { 'Content-Type': 'application/json' } });
const failure = () => response({ error: { code: 'fixture', message: 'Сервер временно недоступен' } }, 503);
const m = (amountMinor: string) => ({ amountMinor, currency: 'USD' });
function harness(entry: 'direct' | 'stock' = 'direct') {
  const state = { requests: [] as any[], failRefresh: false, onPost: undefined as ((body: any) => Response | Promise<Response>) | undefined };
  vi.stubGlobal('fetch', vi.fn(async (input: RequestInfo | URL, init: RequestInit = {}) => {
    const url = String(input);
    if (init.method === 'POST') {
      const body = JSON.parse(String(init.body)); state.requests.push(body);
      return state.onPost ? state.onPost(body) : response({ data: { id: 'saved', price: body.price, installmentDraft: body.installmentTerms ? calculateInstallment(body.price, body.installmentTerms) : null } }, 201);
    }
    if (url.endsWith('/identity/session')) return response({ data: {
      user: { id: 'u', displayName: 'User', status: 'active', passwordChangeRequired: false }, roles: [], permissions: [],
      context: { revision: '1', companyId: 'company', branchScope: { mode: 'ALL', branchIds: [] } },
      accessibleCompanies: [{ id: 'company', name: 'Company', kind: 'seller', access: 'full' }], setup: { next: 'none' },
    } });
    if (url.includes('/retail/customers?')) return response({ items: [{ id: 'customer', displayName: 'Покупатель', phone: '+998' }] });
    if (url.includes('/retail/leads?')) return response({ items: [
      { id: 'lead', stage: 'qualified', customer: { displayName: 'Покупатель' } },
      { id: 'won', stage: 'qualified', dealId: 'another', customer: { displayName: 'Уже продано' } },
      { id: 'new', stage: 'new', customer: { displayName: 'Новый лид' } },
    ] });
    if (url.includes('/vehicle-units?')) return response({ items: [{ id: 'vehicle', vin: 'VIN', modelId: 'model', modelSpecificationVersion: '1', placement: { warehouseId: 'warehouse', placedAt: '2026-10-01' }, exteriorColor: 'Белый', interiorColor: 'Чёрный', reserved: false }, { id: 'blue', vin: 'BLUE', modelId: 'model', modelSpecificationVersion: '1', placement: { warehouseId: 'warehouse', placedAt: '2026-10-01' }, exteriorColor: 'Синий', interiorColor: 'Бежевый', reserved: false }, { id: 'reserved', vin: 'RESERVED', reserved: true }] });
    if (url.includes('/vehicle-models?')) return response({ items: [{ id: 'model', specification: { make: 'Chevrolet', model: 'Onix', variant: 'LT' } }] });
    if (url.includes('/branches?')) return response({ items: [{ id: 'branch', name: 'Ташкент' }] });
    if (url.includes('/retail/deals?') && state.failRefresh) return failure();
    return response({ items: [] });
  }));
  const client = new QueryClient({ defaultOptions: { queries: { retry: false, staleTime: Infinity }, mutations: { retry: false } } });
  render(<QueryClientProvider client={client}><MemoryRouter initialEntries={[entry === 'stock' ? '/sales?new=1' : '/sales']}><SessionGate title="Test">{entry === 'stock' ? <SalesPage /> : <NewSale />}</SessionGate></MemoryRouter></QueryClientProvider>);
  return state;
}
const change = (label: string, value: string) => fireEvent.change(screen.getByLabelText(label, { selector: 'input,textarea' }), { target: { value } });
const pick = (label: string, name: string | RegExp) => {
  screen.getAllByRole('combobox').forEach(input => fireEvent.blur(input));
  fireEvent.focus(screen.getByRole('combobox', { name: label, exact: true }));
  fireEvent.click(screen.getByRole('option', { name, exact: true }));
};
const click = (name: string) => fireEvent.click(screen.getByRole('button', { name }));
async function fill(entry: 'direct' | 'stock' = 'direct', scheme = 'own-installment') {
  if (entry === 'direct') fireEvent.click(await screen.findByRole('button', { name: 'Новая продажа' }));
  await screen.findByRole('dialog', { name: 'Новая продажа' });
  fireEvent.focus(screen.getByLabelText('Филиал')); fireEvent.click(await screen.findByRole('option', { name: 'Ташкент' }));
  pick('Клиент', 'Покупатель · +998'); pick('Автомобиль', /· VIN ·/);
  pick('Способ оформления', scheme === 'own-installment' ? 'Собственная рассрочка' : scheme === 'cash' ? 'Наличные / перевод' : 'Банк / МФО'); change('Цена', '110');
  if (scheme === 'own-installment') { change('Первый взнос', '10'); change('Срок, месяцев', '3'); change('Дата первого платежа', '2024-01-31'); }
}
afterEach(() => { cleanup(); vi.unstubAllGlobals(); });

describe('shared sale creation with generated terms', () => {
  it.each(['direct', 'stock'] as const)('%s entry submits identical exact terms and context, then displays authoritative saved draft', async entry => {
    const state = harness(entry); await fill(entry); pick('Лид (необязательно)', /Покупатель ·/);
    expect(screen.getAllByRole('dialog')).toHaveLength(1);
    expect(screen.queryByRole('option', { name: /RESERVED|Уже продано|Новый лид/ })).toBeNull();
    const preview = within(screen.getByRole('region', { name: 'Предварительный график' }));
    expect(preview.getByText('2024-02-29')).toBeTruthy();
    expect(within(preview.getByText('2024-03-31').closest('tr')!).getByText('33.34 USD')).toBeTruthy();
    state.onPost = body => response({ data: { id: 'saved', price: body.price, installmentDraft: calculateInstallment(body.price, { ...body.installmentTerms, firstDueDate: '2025-05-05' }) } }, 201);
    click('Создать продажу'); await screen.findByText(/Продажа создана/);
    expect(state.requests).toEqual([{ customerId: 'customer', leadId: 'lead', vehicleId: 'vehicle', branchId: 'branch', paymentScheme: 'own-installment', price: m('11000'), installmentTerms: { downPayment: m('1000'), termMonths: 3, firstDueDate: '2024-01-31' } }]);
    expect(screen.getByText('2025-05-05')).toBeTruthy(); expect(screen.queryByText('2024-01-31')).toBeNull();
    expect(screen.queryByRole('button', { name: 'Создать продажу' })).toBeNull();
  });
  it.each(['cash', 'partner-finance'])('keeps %s context and omits terms, with optional lead null', async scheme => {
    const state = harness(); await fill('direct', scheme);
    expect(screen.queryByLabelText('Первый взнос')).toBeNull(); click('Создать продажу'); await screen.findByText(/Продажа создана/);
    expect(state.requests[0]).toEqual({ customerId: 'customer', leadId: null, vehicleId: 'vehicle', branchId: 'branch', paymentScheme: scheme, price: m('11000') });
  });
  it('updates price/down/count/date live, handles presets and removes invalid previews', async () => {
    const state = harness(); await fill();
    change('Цена', '210'); change('Первый взнос', '20'); change('Срок, месяцев', '2'); change('Дата первого платежа', '2023-01-30');
    const region = within(screen.getByRole('region', { name: 'Предварительный график' }));
    expect(region.getByText('190.00 USD')).toBeTruthy(); expect(region.getByText('2023-02-28')).toBeTruthy();
    pick('Вариант срока', '12 месяцев'); expect((screen.getByLabelText('Срок, месяцев') as HTMLInputElement).value).toBe('12');
    change('Вариант срока', 'unmatched');
    expect((screen.getByRole('button', { name: 'Создать продажу' }) as HTMLButtonElement).disabled).toBe(true);
    fireEvent.submit(screen.getByLabelText('Цена').closest('form')!); expect(state.requests).toHaveLength(0);
    expect((screen.getByLabelText('Срок, месяцев') as HTMLInputElement).value).toBe('12');
    change('Вариант срока', ''); pick('Вариант срока', 'Другой срок'); expect(screen.queryByRole('region', { name: 'Предварительный график' })).toBeNull();
    expect((screen.getByRole('button', { name: 'Создать продажу' }) as HTMLButtonElement).disabled).toBe(true);
    change('Срок, месяцев', '2'); change('Первый взнос', '210'); expect(screen.queryByRole('region', { name: 'Предварительный график' })).toBeNull();
    expect(state.requests).toHaveLength(0);
  });
  it('preserves large exact price and entered fields after server error', async () => {
    const state = harness(); await fill();
    change('Цена', '900719925474099.31'); state.onPost = () => failure();
    click('Создать продажу'); await screen.findByText('Сервер временно недоступен');
    expect(state.requests[0].price.amountMinor).toBe('90071992547409931');
    expect((screen.getByLabelText('Цена') as HTMLInputElement).value).toBe('900719925474099.31');
    expect((screen.getByLabelText('Дата первого платежа') as HTMLInputElement).value).toBe('2024-01-31');
    expect((screen.getByLabelText('Клиент') as HTMLInputElement).value).toBe('Покупатель · +998');
  });
  it('blocks duplicate creation and closing while submission is pending', async () => {
    const state = harness(); await fill(); let finish!: () => void;
    state.onPost = body => new Promise(resolve => { finish = () => resolve(response({ data: { id: 'saved', price: body.price } }, 201)); });
    const button = screen.getByRole('button', { name: 'Создать продажу' }); fireEvent.click(button); fireEvent.click(button); click('Закрыть');
    expect(screen.getAllByRole('dialog')).toHaveLength(1); expect(state.requests).toHaveLength(1);
    expect((screen.getByLabelText('Цена') as HTMLInputElement).disabled).toBe(true);
    finish(); await screen.findByText(/Продажа создана/);
  });
  it('retries failed post-success refresh without creating a second sale', async () => {
    const state = harness('stock'); await fill('stock');
    state.onPost = body => { state.failRefresh = true; return response({ data: { id: 'saved', price: body.price, installmentDraft: calculateInstallment(body.price, body.installmentTerms) } }, 201); };
    click('Создать продажу'); await screen.findByRole('button', { name: 'Повторить обновление' });
    state.failRefresh = false; click('Повторить обновление'); await waitFor(() => expect(screen.queryByRole('button', { name: 'Повторить обновление' })).toBeNull());
    expect(state.requests).toHaveLength(1);
  });
  it('labels actual VIN colors in the vehicle choice itself and sends no paint input', async () => {
    const state = harness(); await fill('direct', 'cash');
    expect((screen.getByLabelText('Автомобиль') as HTMLInputElement).value).toContain('VIN · Кузов: Белый · Салон: Чёрный');
    // No separate colour filters: every eligible VIN is offered with its actual colours.
    expect(screen.queryByLabelText('Цвет кузова')).toBeNull();
    expect(screen.queryByLabelText('Цвет салона')).toBeNull();
    fireEvent.focus(screen.getByLabelText('Автомобиль'));
    expect(screen.getByRole('option', { name: /BLUE · Кузов: Синий · Салон: Бежевый/ })).toBeTruthy();
    expect(screen.getByRole('option', { name: /VIN · Кузов: Белый · Салон: Чёрный/ })).toBeTruthy();
    pick('Автомобиль', /· BLUE ·/); pick('Автомобиль', /· VIN ·/); click('Создать продажу'); await screen.findByText(/Продажа создана/);
    expect(state.requests[0]).toMatchObject({ vehicleId: 'vehicle' });
    expect(state.requests[0]).not.toHaveProperty('exteriorColor');
    expect(state.requests[0]).not.toHaveProperty('interiorColor');
  });
  it('rejects unmatched required selections at Submit and preserves exact entered price', async () => {
    const state = harness(); await fill('direct', 'cash'); change('Цена', '900719925474099.31');
    for (const [label, name] of [['Клиент', 'Покупатель · +998'], ['Автомобиль', /· VIN ·/], ['Филиал', 'Ташкент'], ['Способ оформления', 'Наличные / перевод'], ['Валюта', 'EUR']] as const) {
      change(label, 'unmatched');
      expect((screen.getByRole('button', { name: 'Создать продажу' }) as HTMLButtonElement).disabled).toBe(true);
      fireEvent.submit(screen.getByLabelText('Цена').closest('form')!); expect(state.requests).toHaveLength(0);
      change(label, ''); pick(label, name);
    }
    pick('Лид (необязательно)', /Покупатель ·/); pick('Лид (необязательно)', 'Без лида');
    click('Создать продажу'); await screen.findByText(/Продажа создана/);
    expect(state.requests[0]).toMatchObject({ vehicleId: 'vehicle', leadId: null, price: { amountMinor: '90071992547409931', currency: 'EUR' } });
  });
});
