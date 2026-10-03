// @vitest-environment jsdom
import { QueryClient, QueryClientProvider } from '@tanstack/react-query';
import { MemoryRouter } from 'react-router-dom';
import { SessionGate } from '@justixauto/kit';
import { cleanup, fireEvent, render, screen, waitFor } from '@testing-library/react';
import { afterEach, describe, expect, it, vi } from 'vitest';
import { OffersPage } from './pages/offers';
import { OfferDialog } from './pages/trade';

const response = (body: unknown, status = 200) => new Response(JSON.stringify(body), { status, headers: { 'Content-Type': 'application/json' } });
const money = (amountMinor: string) => ({ amountMinor, currency: 'USD' });
const terms = (amountMinor: string, lineId = 'line') => ({ lines: [{ lineId, modelId: 'model', quantity: '1', unitPrice: money(amountMinor), modelSpecificationVersion: '1', exteriorColor: 'Белый', interiorColor: 'Чёрный' }], route: 'stock', deliveryTerms: '', paymentSchedule: [], warrantyTerms: '', serviceTerms: '' });
const version = (id: string, number: number, amountMinor: string, publishedAt: string | null) => ({ id, number, terms: terms(amountMinor), total: money(amountMinor), publishedAt });

function harness(kind: 'supplier' | 'own' | 'listing', listingStatus: 'draft' | 'published' = 'draft') {
  const requests: { url: string; method?: string; body?: unknown }[] = [];
  let failDetail = false;
  let finishDetailFailure: (() => void) | undefined;
  const supplier = { id: 'supplier', name: 'Поставщик' };
  const published = version('published', 1, '12345', '2026-10-01T00:00:00Z');
  const draft = version('draft', 2, '98765', null);
  const offer = { id: kind === 'own' ? 'own-offer' : 'supplier-offer', supplier: kind === 'own' ? { id: 'company', name: 'Наша компания' } : supplier,
    status: kind === 'own' ? 'draft' : 'published', statusReason: '', publishedVersion: published, versions: kind === 'own' ? [published, draft] : undefined, allowedActions: [], revision: '1' };
  const listing = { id: 'listing', vehicleId: 'vehicle', text: 'Onix в наличии', askingPrice: money('55555'), status: listingStatus, revision: '1', updatedAt: '2026-10-01T00:00:00Z' };
  vi.stubGlobal('fetch', vi.fn(async (input: RequestInfo | URL, init: RequestInit = {}) => {
    const url = String(input); requests.push({ url, method: init.method, body: init.body ? JSON.parse(String(init.body)) : undefined });
    if (init.method === 'POST') return response({ data: { id: 'deal', price: JSON.parse(String(init.body)).price } }, 201);
    if (url.endsWith('/identity/session')) return response({ data: { user: { id: 'user', displayName: 'User', status: 'active', passwordChangeRequired: false }, roles: [], permissions: [], context: { revision: '1', companyId: 'company', branchScope: { mode: 'ALL', branchIds: [] } }, accessibleCompanies: [{ id: 'company', name: 'Company', kind: 'seller', access: 'full' }], setup: { next: 'none' } } });
    if (url.includes('/commerce/offers/')) return failDetail ? new Promise<Response>(resolve => { finishDetailFailure = () => resolve(response({ error: { code: 'fixture', message: 'Не удалось обновить карточку' } }, 503)); }) : response({ data: offer });
    if (url.includes('/commerce/offers?')) return response({ items: [offer] });
    if (url.includes('/retail/listings/')) return failDetail ? new Promise<Response>(resolve => { finishDetailFailure = () => resolve(response({ error: { code: 'fixture', message: 'Не удалось обновить карточку' } }, 503)); }) : response({ data: listing });
    if (url.includes('/retail/listings?')) return response({ items: [listing] });
    if (url.includes('/retail/customers?')) return response({ items: [{ id: 'customer', displayName: 'Клиент', phone: '+998', revision: '1' }] });
    if (url.includes('/retail/leads?')) return response({ items: [] });
    if (url.includes('/identity/companies/company/branches')) return response({ items: [{ id: 'branch', name: 'Ташкент', address: '', revision: '1' }] });
    if (url.includes('/inventory/vehicle-models')) return response({ items: [{ id: 'model', specification: { make: 'Chevrolet', model: 'Onix', variant: 'LT' } }] });
    if (url.includes('/inventory/vehicle-units')) return response({ items: [{ id: 'vehicle', vin: 'VIN', modelId: 'model', modelSpecificationVersion: '1', placement: { warehouseId: 'warehouse', placedAt: '2026-10-01' }, exteriorColor: 'Белый', interiorColor: 'Чёрный', reserved: false }] });
    if (url.includes('/inventory/warehouses')) return response({ items: [] });
    return response({ items: [] });
  }));
  const client = new QueryClient({ defaultOptions: { queries: { retry: false, staleTime: Infinity }, mutations: { retry: false } } });
  const wrap = (node: React.ReactNode, entry = '/') => render(<QueryClientProvider client={client}><MemoryRouter initialEntries={[entry]}><SessionGate title="Test">{node}</SessionGate></MemoryRouter></QueryClientProvider>);
  return { requests, wrap, offer, listing, client, failDetail: () => { failDetail = true; }, finishDetailFailure: () => finishDetailFailure?.() };
}

const pick = async (label: string, name: string | RegExp) => {
  const input = screen.getByRole('combobox', { name: label, exact: true });
  fireEvent.focus(input); fireEvent.click(await screen.findByRole('option', { name, exact: true }));
};
afterEach(() => { cleanup(); vi.unstubAllGlobals(); });

describe('sale offer entry', () => {
  it.each(['supplier', 'own', 'listing'] as const)('cached card read %s disables during refetch and hides after failure', async kind => {
    const h = harness(kind); h.wrap(kind === 'listing' ? <OffersPage /> : <OfferDialog id={h.offer.id} onClose={() => {}} />, kind === 'listing' ? '/offers?channel=clients' : '/');
    if (kind === 'listing') {
      fireEvent.click(await screen.findByRole('button', { name: 'Редактировать' }));
      await screen.findByRole('button', { name: 'Создать продажу' });
    } else {
      await screen.findByRole('button', { name: 'Создать продажу' });
    }
    h.failDetail();
    const key = kind === 'listing' ? ['listing', h.listing.id] : ['offer', h.offer.id];
    const refresh = h.client.invalidateQueries({ queryKey: key });
    await waitFor(() => expect((screen.getByRole('button', { name: 'Создать продажу' }) as HTMLButtonElement).disabled).toBe(true));
    h.finishDetailFailure();
    await refresh;
    await waitFor(() => expect(screen.queryByRole('button', { name: 'Создать продажу' })).toBeNull());
  });

  it('uses the supplier published version and does not turn the card action into a purchase', async () => {
    const h = harness('supplier'); const onStartOrder = vi.fn(); h.wrap(<OfferDialog id={h.offer.id} onClose={() => {}} onStartOrder={onStartOrder} />);
    await screen.findByText('Версия №1'); fireEvent.click(screen.getByRole('button', { name: 'Создать продажу' }));
    await screen.findByRole('dialog', { name: 'Новая продажа' }); await screen.findByRole('button', { name: 'Применить предложение' });
    await waitFor(() => expect((screen.getByRole('button', { name: 'Применить предложение' }) as HTMLButtonElement).disabled).toBe(false));
    fireEvent.click(screen.getByRole('button', { name: 'Применить предложение' }));
    await waitFor(() => expect((screen.getByLabelText('Цена') as HTMLInputElement).value).toBe('123.45'));
    expect(onStartOrder).not.toHaveBeenCalled();
    expect(h.requests.filter(request => request.method === 'POST')).toHaveLength(0);
  });

  it('uses the shown own draft instead of its separately published version', async () => {
    const h = harness('own'); h.wrap(<OfferDialog id={h.offer.id} onClose={() => {}} />);
    await screen.findByText('Версия №2'); fireEvent.click(screen.getByRole('button', { name: 'Создать продажу' }));
    await screen.findByRole('button', { name: 'Применить предложение' }); await waitFor(() => expect((screen.getByRole('button', { name: 'Применить предложение' }) as HTMLButtonElement).disabled).toBe(false)); fireEvent.click(screen.getByRole('button', { name: 'Применить предложение' }));
    await waitFor(() => expect((screen.getByLabelText('Цена') as HTMLInputElement).value).toBe('987.65'));
  });

  it.each(['draft', 'published'] as const)('starts a %s client listing in the same sale dialog and posts only after Apply and Submit', async status => {
    const h = harness('listing', status); h.wrap(<OffersPage />, '/offers?channel=clients');
    fireEvent.click(await screen.findByRole('button', { name: status === 'draft' ? 'Редактировать' : 'Открыть' }));
    await screen.findByRole('dialog', { name: 'Предложение клиентам' }); fireEvent.click(await screen.findByRole('button', { name: 'Создать продажу' }));
    await screen.findByRole('dialog', { name: 'Новая продажа' }); expect(screen.getAllByRole('dialog')).toHaveLength(1);
    await screen.findByRole('button', { name: 'Применить предложение' }); await waitFor(() => expect((screen.getByRole('button', { name: 'Применить предложение' }) as HTMLButtonElement).disabled).toBe(false)); fireEvent.click(screen.getByRole('button', { name: 'Применить предложение' }));
    await waitFor(() => expect((screen.getByLabelText('Цена') as HTMLInputElement).value).toBe('555.55'));
    await pick('Клиент', /Клиент/); await pick('Филиал', 'Ташкент'); await pick('Автомобиль', /VIN/); await pick('Способ оформления', 'Наличные / перевод');
    fireEvent.change(screen.getByLabelText('Цена'), { target: { value: '777.77' } }); fireEvent.click(screen.getByRole('button', { name: 'Создать продажу' }));
    await screen.findByText(/Продажа создана/);
    expect(h.requests.filter(request => request.method === 'POST').map(request => request.body)).toEqual([{ customerId: 'customer', leadId: null, vehicleId: 'vehicle', branchId: 'branch', paymentScheme: 'cash', price: money('77777') }]);
  });
});
