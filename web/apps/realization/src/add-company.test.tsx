import { createElement } from 'react';
import { QueryClient, QueryClientProvider } from '@tanstack/react-query';
import { cleanup, fireEvent, render, screen, waitFor, within } from '@testing-library/react';
import { afterEach, describe, expect, it, vi } from 'vitest';

const refresh = vi.fn(async () => undefined);
vi.mock('@justixauto/kit', async (importOriginal) => ({
  ...(await importOriginal<typeof import('@justixauto/kit')>()),
  useSession: () => ({ refresh, can: () => true, company: { id: 'c-1', kind: 'seller' } }),
}));

const { AddCompanyAction } = await import('./pages/settings');

afterEach(() => {
  cleanup();
  vi.unstubAllGlobals();
  refresh.mockClear();
});

describe('add company from Realization', () => {
  it('creates a company and refreshes the session so the selector lists it', async () => {
    let body: unknown;
    vi.stubGlobal(
      'fetch',
      vi.fn(async (input: RequestInfo | URL, init?: RequestInit) => {
        if (String(input).endsWith('/identity/companies') && init?.method === 'POST') {
          body = JSON.parse(String(init.body));
          return new Response(JSON.stringify({ data: {}, revision: '1' }), {
            status: 201,
            headers: { 'Content-Type': 'application/json' },
          });
        }
        throw new Error(`Unexpected request ${String(input)}`);
      }),
    );
    const client = new QueryClient();
    render(createElement(QueryClientProvider, { client }, createElement(AddCompanyAction)));

    fireEvent.click(screen.getByRole('button', { name: '+ Добавить компанию' }));
    const dialog = await screen.findByRole('dialog', { name: 'Новая компания' });
    fireEvent.change(within(dialog).getByLabelText('Название компании'), { target: { value: 'Авто плюс 2' } });
    fireEvent.click(within(dialog).getByRole('button', { name: 'Создать' }));

    await waitFor(() => expect(refresh).toHaveBeenCalled());
    expect(body).toMatchObject({ company: { name: 'Авто плюс 2' } });
  });
});
