import { createElement } from 'react';
import { QueryClient, QueryClientProvider } from '@tanstack/react-query';
import { cleanup, fireEvent, render, screen, waitFor, within } from '@testing-library/react';
import { MemoryRouter } from 'react-router-dom';
import { afterEach, describe, expect, it, vi } from 'vitest';
import { CompaniesPage } from './pages';

afterEach(() => {
  cleanup();
  vi.unstubAllGlobals();
});

const company = {
  id: 'company-1', kind: 'seller', name: 'Авто плюс', legalName: '', country: { label: 'Казахстан' }, region: null,
  registration: '', email: '', address: '', phone: '', access: 'active', accessReason: '', userType: null,
  verification: 'submitted', verificationReason: '', revision: '7',
};
const response = (body: unknown) => new Response(JSON.stringify(body), { status: 200, headers: { 'Content-Type': 'application/json' } });

function renderCompanies() {
  const client = new QueryClient({ defaultOptions: { queries: { retry: false } } });
  return render(
    createElement(QueryClientProvider, { client }, createElement(MemoryRouter, null, createElement(CompaniesPage, { kind: 'seller' }))),
  );
}

describe('company verification review', () => {
  it('assigns the selected business type to a legacy seller exactly once', async () => {
    const fetch = vi.fn(async (input: RequestInfo | URL, init?: RequestInit) => {
      const url = String(input);
      if (url.includes('/admin/companies/company-1/users')) return response({ items: [] });
      if (url.endsWith('/identity/admin/companies?kind=seller&limit=100')) return response({ items: [company] });
      if (url.endsWith('/identity/companies/company-1')) return response({ data: company, revision: '7' });
      if (url.endsWith('/identity/admin/companies/company-1/set-user-type')) return response({ data: { ...company, userType: 'distributor' }, revision: '8' });
      throw new Error(`Unexpected request ${url} ${init?.method}`);
    });
    vi.stubGlobal('fetch', fetch);
    renderCompanies();

    fireEvent.click(await screen.findByText('Авто плюс'));
    fireEvent.click(await screen.findByRole('button', { name: 'Назначить бизнес-тип' }));
    const dialog = await screen.findByRole('dialog', { name: 'Назначить бизнес-тип' });
    fireEvent.click(within(dialog).getByRole('radio', { name: 'Дистрибьютор' }));
    fireEvent.click(within(dialog).getByRole('button', { name: 'Назначить бизнес-тип' }));

    await waitFor(() => expect(fetch).toHaveBeenCalledWith(expect.stringContaining('/set-user-type'), expect.any(Object)));
    const [, init] = fetch.mock.calls.find(([url]) => String(url).endsWith('/set-user-type'))!;
    expect(JSON.parse(String(init?.body))).toEqual({ userType: 'distributor' });
    expect(new Headers(init?.headers).get('If-Match')).toBe('"7"');
  });

  it.each(['dealer', 'distributor'])('does not offer a type assignment for an already-fixed %s seller', async (userType) => {
    const fixed = { ...company, userType };
    const fetch = vi.fn(async (input: RequestInfo | URL) => {
      const url = String(input);
      if (url.includes('/admin/companies/company-1/users')) return response({ items: [] });
      if (url.endsWith('/identity/admin/companies?kind=seller&limit=100')) return response({ items: [fixed] });
      if (url.endsWith('/identity/companies/company-1')) return response({ data: fixed, revision: '7' });
      throw new Error(`Unexpected request ${url}`);
    });
    vi.stubGlobal('fetch', fetch);
    renderCompanies();

    fireEvent.click(await screen.findByText('Авто плюс'));
    expect(screen.queryByRole('button', { name: 'Назначить бизнес-тип' })).toBeNull();
    fireEvent.click(await screen.findByRole('button', { name: 'Изменить' }));
    const dialog = await screen.findByRole('dialog', { name: 'Изменить: Авто плюс' });
    expect(within(dialog).queryByText('Бизнес-тип')).toBeNull();
  });

  it('allows the platform admin to return submitted details with a reason', async () => {
    const fetch = vi.fn(async (input: RequestInfo | URL, init?: RequestInit) => {
      const url = String(input);
      if (url.includes('/admin/companies/company-1/users')) return response({ items: [] });
      if (url.endsWith('/identity/admin/companies?kind=seller&limit=100')) return response({ items: [company] });
      if (url.endsWith('/identity/companies/company-1')) return response({ data: company, revision: '7' });
      if (url.endsWith('/identity/admin/companies/company-1/return-verification')) return response({ data: { ...company, verification: 'returned', verificationReason: 'Добавьте реквизиты' }, revision: '8' });
      throw new Error(`Unexpected request ${url} ${init?.method}`);
    });
    vi.stubGlobal('fetch', fetch);
    renderCompanies();

    fireEvent.click(await screen.findByText('Авто плюс'));
    fireEvent.click(await screen.findByRole('button', { name: 'Вернуть на доработку' }));
    const dialog = await screen.findByRole('dialog', { name: 'Вернуть на доработку' });
    fireEvent.change(within(dialog).getByLabelText('Причина возврата'), { target: { value: 'Добавьте реквизиты' } });
    fireEvent.click(within(dialog).getByRole('button', { name: 'Вернуть на доработку' }));

    await waitFor(() => expect(fetch).toHaveBeenCalledWith(expect.stringContaining('/return-verification'), expect.any(Object)));
    const [, init] = fetch.mock.calls.find(([url]) => String(url).endsWith('/return-verification'))!;
    expect(JSON.parse(String(init?.body))).toEqual({ reason: 'Добавьте реквизиты' });
    expect(new Headers(init?.headers).get('If-Match')).toBe('"7"');
  });
});
