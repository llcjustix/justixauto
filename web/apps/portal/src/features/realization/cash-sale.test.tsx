// @vitest-environment jsdom
import { QueryClient, QueryClientProvider } from '@tanstack/react-query';
import { MemoryRouter } from 'react-router-dom';
import { SessionGate } from '@justixauto/kit';
import { cleanup, fireEvent, render, screen, waitFor, within } from '@testing-library/react';
import { afterEach, describe, expect, it, vi } from 'vitest';
import { DealDialog } from './pages/retail';
import type { Deal, Money, RetailInvoice } from './data';

const m = (amountMinor: string): Money => ({ amountMinor, currency: 'USD' });
const response = (body: unknown, status = 200) => new Response(JSON.stringify(body), { status, headers: { 'Content-Type': 'application/json' } });
type Recorded = { url: string; body: Record<string, unknown>; revision: string | null };

function invoice(id: string, status: 'issued' | 'void' = 'issued'): RetailInvoice {
  return { id, dealId: 'd-1', purpose: 'vehicle-payment', installmentNumber: null, amount: m('9007199254740993'), recipientSnapshot: 'Покупатель', dueDate: '2026-10-10', status,
    paid: m('0'), pending: m('0'), outstanding: m('9007199254740993'), available: m('9007199254740993'), allowedActions: ['submit-payment'], paymentEvidence: [], revision: 'invoice-rev' };
}

function cashDeal(): Deal {
  return { id: 'd-1', branchId: 'branch', customer: { id: 'customer', displayName: 'Покупатель', phone: '+998', revision: '1' }, leadId: null, vehicleId: 'vehicle',
    paymentScheme: 'cash', price: m('9007199254740993'), status: 'reserved', statusReason: '', contractSignedOn: null, contractReference: '', contractFileIds: [],
    registeredOn: null, plateNumber: '', registrationReference: '', deliveredAt: null, invoices: [], installmentPlan: null, installmentDraft: null,
    checklist: { contract: false, vehiclePayment: false, registrationPaid: false, registered: false, policyResolved: true }, allowedActions: ['issue-invoice'], history: [], revision: 'deal-rev', updatedAt: '2026-10-02T00:00:00Z' };
}

function harness(initial = cashDeal()) {
  const state = { deal: initial, requests: [] as Recorded[] };
  vi.stubGlobal('fetch', vi.fn(async (input: RequestInfo | URL, init: RequestInit = {}) => {
    const url = String(input);
    if (init.method === 'POST') {
      state.requests.push({ url, body: JSON.parse(String(init.body)), revision: new Headers(init.headers).get('If-Match') });
      return response({ data: state.deal, revision: state.deal.revision }, 201);
    }
    if (url.endsWith('/identity/session')) return response({ data: { user: { id: 'u', displayName: 'User', status: 'active', passwordChangeRequired: false }, roles: [], permissions: [], context: { revision: '1', companyId: 'company', branchScope: { mode: 'ALL', branchIds: [] } }, accessibleCompanies: [{ id: 'company', name: 'Company', kind: 'seller', access: 'full' }], setup: { next: 'none' } } });
    if (url.endsWith('/retail/deals/d-1')) return response({ data: state.deal, revision: state.deal.revision });
    return response({ items: [] });
  }));
  const client = new QueryClient({ defaultOptions: { queries: { retry: false, staleTime: Infinity }, mutations: { retry: false } } });
  render(<QueryClientProvider client={client}><MemoryRouter><SessionGate title="Test"><DealDialog id="d-1" onClose={vi.fn()} /></SessionGate></MemoryRouter></QueryClientProvider>);
  return state;
}

const click = (name: string | RegExp) => fireEvent.click(screen.getByRole('button', { name }));
const change = (label: string | RegExp, value: string) => fireEvent.change(screen.getByLabelText(label), { target: { value } });
async function payments() { fireEvent.click(await screen.findByRole('tab', { name: 'Оплаты', exact: true })); }
afterEach(() => { cleanup(); vi.unstubAllGlobals(); });

describe('cash sale', () => {
  it('posts the fixed full-price vehicle invoice without editable amount or currency', async () => {
    const state = harness();
    fireEvent.click(await screen.findByRole('button', { name: 'Выставить счёт за автомобиль' }));
    expect(screen.queryByLabelText('Сумма')).toBeNull();
    expect(screen.queryByLabelText('Валюта')).toBeNull();
    expect(screen.getByText(/90 071 992 547 409.93 USD.*зафиксированы/)).toBeTruthy();
    change('Оплатить до', '2026-10-10'); click('Выставить счёт за автомобиль');
    await waitFor(() => expect(state.requests).toHaveLength(1));
    expect(state.requests[0]).toEqual({ url: '/api/v1/retail/deals/d-1/invoices', revision: '"deal-rev"', body: { recipientSnapshot: 'Покупатель', dueDate: '2026-10-10', purpose: 'vehicle-payment', amount: m('9007199254740993') } });
  });

  it('suppresses only an issued vehicle invoice and permits creation after void', async () => {
    const issued = cashDeal(); issued.invoices = [invoice('legacy-issued')]; harness(issued);
    await screen.findByRole('tabpanel', { name: 'Сделка и выдача' });
    expect(screen.queryByRole('button', { name: 'Выставить счёт за автомобиль' })).toBeNull();
    cleanup(); vi.unstubAllGlobals();
    const voided = cashDeal(); voided.invoices = [invoice('voided', 'void')]; harness(voided);
    expect(await screen.findByRole('button', { name: 'Выставить счёт за автомобиль' })).toBeTruthy();
  });

  it('keeps the manual registration-fee action editable and isolates cash vehicle payment', async () => {
    const state = harness(); fireEvent.click(await screen.findByRole('button', { name: 'Выставить счёт' }));
    const purpose = screen.getByRole('combobox', { name: 'Назначение' }); fireEvent.click(purpose);
    fireEvent.click(await screen.findByRole('option', { name: 'Регистрация' }));
    expect(screen.queryByRole('option', { name: 'Оплата автомобиля' })).toBeNull();
    change('Сумма', '5.25'); change('Оплатить до', '2026-10-10'); click('Выставить счёт');
    await waitFor(() => expect(state.requests).toHaveLength(1));
    expect(state.requests[0]!.body).toMatchObject({ purpose: 'registration', amount: m('525') });
  });

  it('uses explicit registration policy before the cash fallback and keeps installments required', async () => {
    const explicitFalse = cashDeal(); explicitFalse.checklist!.registrationOptional = false; harness(explicitFalse);
    expect((await screen.findByLabelText('Готовность к выдаче')).textContent).not.toContain('необязательно');
    cleanup(); vi.unstubAllGlobals();
    const fallback = cashDeal(); harness(fallback);
    expect((await screen.findByLabelText('Готовность к выдаче')).textContent).toContain('Оплата регистрации (необязательно)');
    cleanup(); vi.unstubAllGlobals();
    const installment = cashDeal(); installment.paymentScheme = 'own-installment'; installment.checklist!.registrationOptional = false; harness(installment);
    expect((await screen.findByLabelText('Готовность к выдаче')).textContent).not.toContain('необязательно');
  });

  it('preserves repeated partial payments, balances, and pending versus accepted facts', async () => {
    const d = cashDeal(); const vehicle = invoice('vehicle'); vehicle.amount = m('1000'); vehicle.outstanding = m('1000'); vehicle.available = m('1000'); d.invoices = [vehicle];
    const state = harness(d); await payments(); click('Внести оплату'); change('Сумма', '5.00'); change('Дата оплаты', '2026-10-02'); change(/Номер платёжки/, 'PAY-1'); click('Внести оплату');
    await waitFor(() => expect(state.requests).toHaveLength(1));
    expect(state.requests[0]!.url).toBe('/api/v1/retail/invoices/vehicle/evidence');
    expect(state.requests[0]!.body).toMatchObject({ claimedAmount: m('500'), externalReference: 'PAY-1' });
  });
});
