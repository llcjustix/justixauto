// @vitest-environment jsdom
import { createElement } from 'react';
import { QueryClient, QueryClientProvider } from '@tanstack/react-query';
import { cleanup, render, screen } from '@testing-library/react';
import { MemoryRouter } from 'react-router-dom';
import { afterEach, describe, expect, it, vi } from 'vitest';
import { PurchasesPage } from './pages/purchases';

afterEach(() => {
  cleanup();
  vi.unstubAllGlobals();
});

const order = (id: string, party: 'buyer' | 'supplier', supplier: string) => ({
  id,
  party,
  buyer: { name: 'Test Company' },
  supplier: { name: supplier },
  source: 'offer',
  terms: { lines: [] },
  total: { amountMinor: '100000', currency: 'USD' },
  status: 'awaiting-supplier',
  statusReason: '',
  allocations: [],
  shipments: [],
  addenda: [],
  allowedActions: [],
  revision: '1',
  updatedAt: '2026-09-27T15:34:09Z',
});

describe('purchases', () => {
  it('lists only our orders and creates orders only from supplier offers', async () => {
    vi.stubGlobal(
      'fetch',
      vi.fn(async (input: RequestInfo | URL) => {
        const url = String(input);
        if (url.includes('/commerce/rfqs')) throw new Error('purchase requests are retired');
        const items = url.includes('/commerce/orders')
          ? [order('o-1', 'buyer', 'My Company 1'), order('o-2', 'supplier', 'Test Company')]
          : [];
        return new Response(JSON.stringify({ items }), {
          status: 200,
          headers: { 'Content-Type': 'application/json' },
        });
      }),
    );
    const client = new QueryClient({ defaultOptions: { queries: { retry: false } } });
    render(
      createElement(QueryClientProvider, { client }, createElement(MemoryRouter, null, createElement(PurchasesPage))),
    );

    expect(await screen.findByText('My Company 1')).toBeTruthy();
    expect(screen.queryByText('Test Company')).toBeNull();
    expect(screen.getByRole('button', { name: 'Создать заказ' })).toBeTruthy();
    expect(screen.queryByRole('button', { name: 'Создать запрос' })).toBeNull();
  });
});
