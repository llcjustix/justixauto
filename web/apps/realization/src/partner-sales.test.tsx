// @vitest-environment jsdom
import { createElement } from 'react';
import { QueryClient, QueryClientProvider } from '@tanstack/react-query';
import { cleanup, render, screen } from '@testing-library/react';
import { MemoryRouter } from 'react-router-dom';
import { afterEach, describe, expect, it, vi } from 'vitest';

vi.mock('@justixauto/kit', async (importOriginal) => ({
  ...(await importOriginal<typeof import('@justixauto/kit')>()),
  useSession: () => ({ can: () => true, company: { id: 'supplier-1', kind: 'seller' } }),
}));

const { SalesPage } = await import('./pages/sales');

afterEach(() => {
  cleanup();
  vi.unstubAllGlobals();
});

const rfq = (id: string, buyer: string, supplier: string, status: string) => ({
  id,
  buyer: { id: buyer, name: `Покупатель ${buyer}` },
  supplier: { id: supplier, name: `Поставщик ${supplier}` },
  lines: [],
  status,
  statusReason: '',
  quotations: [],
  allowedActions: [],
  revision: '1',
  updatedAt: '2026-09-27T15:34:09Z',
});

describe('wholesale sales, supplier side', () => {
  it('opens on incoming quotation requests and hides our own purchase requests', async () => {
    vi.stubGlobal(
      'fetch',
      vi.fn(async (input: RequestInfo | URL) => {
        const items = String(input).includes('/commerce/rfqs')
          ? [rfq('r-1', 'buyer-1', 'supplier-1', 'sent'), rfq('r-2', 'supplier-1', 'other-9', 'sent')]
          : [];
        return new Response(JSON.stringify({ items }), {
          status: 200,
          headers: { 'Content-Type': 'application/json' },
        });
      }),
    );
    const client = new QueryClient({ defaultOptions: { queries: { retry: false } } });
    render(
      createElement(
        QueryClientProvider,
        { client },
        createElement(MemoryRouter, { initialEntries: ['/?channel=partners'] }, createElement(SalesPage)),
      ),
    );

    expect(await screen.findByText('Покупатель buyer-1')).toBeTruthy();
    expect(screen.getByText('Ждёт котировки')).toBeTruthy();
    expect(screen.getByRole('button', { name: 'Ответить' })).toBeTruthy();
    expect(screen.queryByText('Покупатель supplier-1')).toBeNull();
  });
});
