// @vitest-environment jsdom
import { QueryClient, QueryClientProvider } from '@tanstack/react-query';
import { SessionGate } from '@justixauto/kit';
import { cleanup, fireEvent, render, screen, waitFor } from '@testing-library/react';
import { afterEach, describe, expect, it, vi } from 'vitest';
import { SaleInsurance } from './sale-insurance';
import type { Deal } from './data';

const response = (body: unknown, status = 200) => new Response(JSON.stringify(body), { status, headers: { 'Content-Type': 'application/json' } });
const deal: Deal = { id: 'd-1', branchId: 'b', customer: { id: 'c', displayName: 'Покупатель', phone: '', revision: '1' }, leadId: null, vehicleId: 'v', paymentScheme: 'own-installment', price: { amountMinor: '1000', currency: 'USD' }, status: 'reserved', statusReason: '', contractSignedOn: null, contractReference: '', contractFileIds: [], registeredOn: null, plateNumber: '', registrationReference: '', deliveredAt: null, allowedActions: ['record-contract'], revision: 'deal-r1', updatedAt: '' };
afterEach(() => { cleanup(); vi.unstubAllGlobals(); });
function renderSale(permission = true, scoreFails = false) {
  const requests: string[] = [];
  vi.stubGlobal('fetch', vi.fn(async (input: RequestInfo | URL, init: RequestInit = {}) => {
    const url = String(input); if (init.method === 'POST') requests.push(url);
    if (url.endsWith('/identity/session')) return response({ data: { user: { id: 'u', displayName: 'U', status: 'active', passwordChangeRequired: false }, roles: [], permissions: permission ? ['retail.deals.manage'] : [], context: { revision: '1', companyId: 'seller', branchScope: { mode: 'ALL', branchIds: [] } }, accessibleCompanies: [], setup: { next: 'none' } } });
    if (url.includes('demo-scoring')) return scoreFails ? response({ message: 'stale' }, 409) : response({ data: deal });
    if (url.includes('/insurance/applications')) return response({ items: [] });
    return response({ data: deal });
  }));
  const client = new QueryClient({ defaultOptions: { queries: { retry: false } } });
  render(<QueryClientProvider client={client}><SessionGate title="T"><SaleInsurance deal={deal} refresh={[['deal', 'd-1']]} /></SessionGate></QueryClientProvider>);
  return requests;
}

describe('sale insurance', () => {
  it('keeps demo scoring visibly separate from coverage and surfaces a stale-revision failure', async () => {
    const requests = renderSale(true, true);
    expect(await screen.findByText(/не является полисом, покрытием/)).toBeTruthy();
    fireEvent.click(screen.getByRole('button', { name: /Запустить/ }));
    await waitFor(() => expect(screen.getByText(/Не удалось запустить demo-оценку/)).toBeTruthy());
    expect(requests).toContain('/api/v1/retail/deals/d-1/demo-scoring');
  });
  it('does not offer scoring or insurer approval to a seller without the server permission', async () => {
    renderSale(false);
    await screen.findByText('Недостаточно прав для запуска demo-оценки.');
    expect(screen.queryByRole('button', { name: /Запустить/ })).toBeNull();
    expect(screen.queryByRole('button', { name: 'Одобрить' })).toBeNull();
  });
});
