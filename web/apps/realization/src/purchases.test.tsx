// @vitest-environment jsdom
import { createElement } from 'react';
import { QueryClient, QueryClientProvider } from '@tanstack/react-query';
import { cleanup, fireEvent, render, screen, waitFor, within } from '@testing-library/react';
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
  source: 'direct',
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

const partner = {
  id: 'p-1',
  direction: 'outgoing',
  counterparty: { id: 'c-2', name: 'My Company 1', country: 'Узбекистан' },
  status: 'active',
  statusReason: '',
  allowedActions: [],
};
const model = {
  id: 'm-1',
  specification: { make: 'Chevrolet', model: 'Cobalt', variant: 'LTZ', year: 2025, version: 1 },
  revision: '1',
};

function renderPage(onPost?: (url: string, body: unknown) => void) {
  vi.stubGlobal(
    'fetch',
    vi.fn(async (input: RequestInfo | URL, init?: RequestInit) => {
      const url = String(input);
      if (init?.method === 'POST') {
        onPost?.(url, JSON.parse(String(init.body)));
        return new Response(JSON.stringify({ data: order('o-9', 'buyer', 'My Company 1'), revision: '1' }), {
          status: 201,
          headers: { 'Content-Type': 'application/json' },
        });
      }
      if (url.includes('/commerce/rfqs')) throw new Error('purchase requests are retired');
      const items = url.includes('/commerce/orders')
        ? [order('o-1', 'buyer', 'My Company 1'), order('o-2', 'supplier', 'Test Company')]
        : url.includes('/commerce/partnerships')
          ? [partner]
          : url.includes('/inventory/vehicle-models')
            ? [model]
            : [];
      return new Response(JSON.stringify({ items }), { status: 200, headers: { 'Content-Type': 'application/json' } });
    }),
  );
  const client = new QueryClient({ defaultOptions: { queries: { retry: false } } });
  render(
    createElement(QueryClientProvider, { client }, createElement(MemoryRouter, null, createElement(PurchasesPage))),
  );
}

describe('purchases', () => {
  it('lists only our orders and offers no quotation requests', async () => {
    renderPage();
    expect(await screen.findByText('My Company 1')).toBeTruthy();
    expect(screen.queryByText('Test Company')).toBeNull();
    expect(screen.queryByRole('button', { name: 'Создать запрос' })).toBeNull();
  });

  it('orders any quantity from a partner without an offer', async () => {
    let posted: { url: string; body: unknown } | undefined;
    renderPage((url, body) => (posted = { url, body }));
    await screen.findByText('My Company 1');

    fireEvent.click(screen.getByRole('button', { name: 'Создать заказ' }));
    const dialog = await screen.findByRole('dialog', { name: 'Новый заказ поставщику' });
    await within(dialog).findByRole('option', { name: 'My Company 1' });
    fireEvent.change(within(dialog).getByLabelText(/Поставщик/), { target: { value: 'c-2' } });
    await within(dialog).findByRole('option', { name: /Chevrolet Cobalt/ });
    fireEvent.change(within(dialog).getByLabelText('Модель'), { target: { value: 'm-1' } });
    fireEvent.change(within(dialog).getByLabelText('Количество'), { target: { value: '50' } });
    fireEvent.change(within(dialog).getByLabelText('Цена'), { target: { value: '25000' } });
    expect(within(dialog).getByText('1 250 000.00 USD', { selector: 'tfoot td' })).toBeTruthy();
    fireEvent.click(within(dialog).getByRole('button', { name: 'Отправить заказ' }));

    await waitFor(() => expect(posted).toBeDefined());
    expect(posted!.url).toMatch(/\/commerce\/orders$/);
    expect(posted!.body).toMatchObject({
      supplierCompanyId: 'c-2',
      terms: { lines: [{ modelId: 'm-1', quantity: '50', unitPrice: { amountMinor: '2500000', currency: 'USD' } }] },
    });
  });

  it('explains what is missing before sending', async () => {
    let posted = false;
    renderPage(() => (posted = true));
    await screen.findByText('My Company 1');
    fireEvent.click(screen.getByRole('button', { name: 'Создать заказ' }));
    const dialog = await screen.findByRole('dialog', { name: 'Новый заказ поставщику' });
    fireEvent.click(within(dialog).getByRole('button', { name: 'Отправить заказ' }));
    expect(await within(dialog).findByText('Заполните поле «Поставщик»')).toBeTruthy();
    expect(posted).toBe(false);
  });
});
