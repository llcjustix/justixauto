// @vitest-environment jsdom
import { QueryClient, QueryClientProvider } from '@tanstack/react-query';
import { MemoryRouter } from 'react-router-dom';
import { SessionGate } from '@justixauto/kit';
import { cleanup, render, screen } from '@testing-library/react';
import { afterEach, describe, expect, it, vi } from 'vitest';
import { VehiclesPage } from './pages/stock';

const response = (body: unknown) => new Response(JSON.stringify(body), { headers: { 'Content-Type': 'application/json' } });

afterEach(() => { cleanup(); vi.unstubAllGlobals(); });

describe('stock colors', () => {
  it('shows each VIN actual body and interior colors independently of its model palette', async () => {
    vi.stubGlobal('fetch', vi.fn(async (input: RequestInfo | URL) => {
      const url = String(input);
      if (url.endsWith('/identity/session')) return response({ data: { user: { id: 'user', displayName: 'User', status: 'active', passwordChangeRequired: false }, roles: [], permissions: [], context: { revision: '1', companyId: 'company', branchScope: { mode: 'ALL', branchIds: [] } }, accessibleCompanies: [{ id: 'company', name: 'Company', kind: 'seller', access: 'full' }], setup: { next: 'none' } } });
      if (url.includes('/vehicle-units?')) return response({ items: [
        { id: 'black', vin: 'BLACK-VIN', modelId: 'model', modelSpecificationVersion: 'v1', exteriorColor: 'Чёрный', interiorColor: 'Чёрный', placement: { warehouseId: 'warehouse' }, reserved: false },
        { id: 'white', vin: 'WHITE-VIN', modelId: 'model', modelSpecificationVersion: 'v1', exteriorColor: 'Белый', placement: { warehouseId: 'warehouse' }, reserved: false },
      ] });
      if (url.includes('/vehicle-models?')) return response({ items: [{ id: 'model', specification: { make: 'Chevrolet', model: 'Onix', variant: 'LT', year: 2024, bodyType: 'sedan', exteriorColor: 'Каталожный', interiorColor: 'Каталожный' } }] });
      if (url.endsWith('/warehouses')) return response({ items: [] });
      if (url.includes('/listings?')) return response({ items: [] });
      return response({ items: [] });
    }));
    const client = new QueryClient({ defaultOptions: { queries: { retry: false, staleTime: Infinity } } });
    render(<QueryClientProvider client={client}><MemoryRouter><SessionGate title="Test"><VehiclesPage /></SessionGate></MemoryRouter></QueryClientProvider>);
    expect(await screen.findByText('Кузов: Чёрный · Салон: Чёрный · Версия: v1')).toBeTruthy();
    expect(screen.getByText('Кузов: Белый · Салон: Не указан · Версия: v1')).toBeTruthy();
    expect(screen.queryByText(/Каталожный/)).toBeNull();
  });
});
