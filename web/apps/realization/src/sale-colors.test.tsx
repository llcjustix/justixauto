// @vitest-environment jsdom
import { QueryClient, QueryClientProvider } from '@tanstack/react-query';
import { MemoryRouter } from 'react-router-dom';
import { SessionGate } from '@justixauto/kit';
import { cleanup, fireEvent, render, screen } from '@testing-library/react';
import { afterEach, describe, expect, it, vi } from 'vitest';
import { DealDialog } from './pages/retail';

const response = (body: unknown) => new Response(JSON.stringify(body), { headers: { 'Content-Type': 'application/json' } });
const session = { data: { user: { id: 'user', displayName: 'User', status: 'active', passwordChangeRequired: false }, roles: [], permissions: [], context: { revision: '1', companyId: 'company', branchScope: { mode: 'ALL', branchIds: [] } }, accessibleCompanies: [{ id: 'company', name: 'Company', kind: 'seller', access: 'full' }], setup: { next: 'none' } } };

afterEach(() => { cleanup(); vi.unstubAllGlobals(); });

describe('sale colors', () => {
  it('uses the saved vehicle snapshot after delivery and keeps a null historical snapshot unknown', async () => {
    vi.stubGlobal('fetch', vi.fn(async (input: RequestInfo | URL) => {
      const url = String(input);
      if (url.endsWith('/identity/session')) return response(session);
      if (url.includes('/retail/deals/saved')) return response({ data: { id: 'saved', branchId: 'branch', customer: { id: 'customer', displayName: 'Покупатель', phone: '' }, leadId: null, vehicleId: 'gone', vehicleSnapshot: { vehicleId: 'gone', vin: 'SAVED-VIN', modelId: 'model', modelSpecificationVersion: 'v1', exteriorColor: 'Красный', interiorColor: 'Бежевый' }, paymentScheme: 'cash', price: { amountMinor: '100', currency: 'USD' }, status: 'delivered', statusReason: '', contractSignedOn: null, contractReference: '', contractFileIds: [], registeredOn: null, plateNumber: '', registrationReference: '', deliveredAt: '2026-10-02T00:00:00Z', allowedActions: [], invoices: [] } });
      if (url.includes('/retail/deals/legacy')) return response({ data: { id: 'legacy', branchId: 'branch', customer: { id: 'customer', displayName: 'Покупатель', phone: '' }, leadId: null, vehicleId: 'gone', vehicleSnapshot: null, paymentScheme: 'cash', price: { amountMinor: '100', currency: 'USD' }, status: 'delivered', statusReason: '', contractSignedOn: null, contractReference: '', contractFileIds: [], registeredOn: null, plateNumber: '', registrationReference: '', deliveredAt: '2026-10-02T00:00:00Z', allowedActions: [], invoices: [] } });
      if (url.includes('/vehicle-units?') || url.includes('/vehicle-models?')) return response({ items: [] });
      return response({ items: [] });
    }));
    const client = new QueryClient({ defaultOptions: { queries: { retry: false, staleTime: Infinity } } });
    const onClose = vi.fn();
    const view = render(<QueryClientProvider client={client}><MemoryRouter><SessionGate title="Test"><DealDialog id="saved" onClose={onClose} /></SessionGate></MemoryRouter></QueryClientProvider>);
    expect(await screen.findByText('SAVED-VIN')).toBeTruthy();
    expect(screen.getByText('Кузов: Красный · Салон: Бежевый · Версия: v1')).toBeTruthy();
    expect(screen.getByRole('tab', { name: 'Оплаты' })).toBeTruthy();
    fireEvent.click(screen.getAllByRole('button', { name: 'Закрыть' }).at(-1)!);
    expect(onClose).toHaveBeenCalledOnce();
    view.unmount();
    render(<QueryClientProvider client={client}><MemoryRouter><SessionGate title="Test"><DealDialog id="legacy" onClose={() => {}} /></SessionGate></MemoryRouter></QueryClientProvider>);
    expect(await screen.findByText('Кузов: Не указан · Салон: Не указан · Версия: Не указан')).toBeTruthy();
  });
});
