// @vitest-environment jsdom
import { createElement } from 'react';
import { QueryClient, QueryClientProvider } from '@tanstack/react-query';
import { cleanup, fireEvent, render, screen, waitFor, within } from '@testing-library/react';
import { afterEach, describe, expect, it, vi } from 'vitest';
import { PERM_COMPANY_USERS } from './company-employees';
import { CompanyRoles } from './company-roles';

vi.mock('./session', () => ({
  useSession: () => ({
    view: { user: { id: 'admin-1' } },
    company: { id: 'c-1', kind: 'seller' },
    can: (p: string) => p === PERM_COMPANY_USERS,
  }),
}));

afterEach(() => {
  cleanup();
  vi.unstubAllGlobals();
});

const response = (body: unknown) =>
  new Response(JSON.stringify(body), { status: 200, headers: { 'Content-Type': 'application/json' } });

function stubApi(onWrite?: (url: string, body: unknown, method?: string) => void) {
  vi.stubGlobal(
    'fetch',
    vi.fn(async (input: RequestInfo | URL, init?: RequestInit) => {
      const url = String(input);
      if (init?.method === 'POST' || init?.method === 'PATCH') {
        onWrite?.(url, JSON.parse(String(init.body)), init.method);
        return response({ data: {}, revision: '2' });
      }
      if (url.endsWith('/identity/companies/c-1/roles')) {
        return response({
          items: [
            {
              id: 'r-admin',
              name: 'Company administrator',
              system: true,
              permissionKeys: ['retail.read'],
              revision: '1',
            },
            { id: 'r-cash', name: 'Кассир', system: false, permissionKeys: ['retail.read'], revision: '1' },
          ],
        });
      }
      if (url.endsWith('/identity/companies/c-1/permissions')) {
        return response({
          items: [
            { key: 'retail.read', scope: 'company', name: 'Продажи: просмотр', assignable: true },
            { key: 'inventory.read', scope: 'company', name: 'Склад: просмотр', assignable: true },
          ],
        });
      }
      throw new Error(`Unexpected request ${url} ${init?.method}`);
    }),
  );
}

function renderRoles() {
  const client = new QueryClient({ defaultOptions: { queries: { retry: false } } });
  return render(createElement(QueryClientProvider, { client }, createElement(CompanyRoles)));
}

describe('company roles', () => {
  it('lists the built-in admin read-only and own roles with permission names', async () => {
    stubApi();
    renderRoles();
    const own = (await screen.findByText('Кассир')).closest('tr')!;
    const builtIn = screen.getByText('Company administrator').closest('tr')!;
    await waitFor(() => expect(within(own).getByText('Продажи: просмотр')).toBeTruthy());
    expect(within(builtIn).getByText('Все разрешения компании')).toBeTruthy();
    expect(within(builtIn).queryByRole('button', { name: 'Изменить' })).toBeNull();
    expect(within(own).getByRole('button', { name: 'Изменить' })).toBeTruthy();
  });

  it('creates a company role from company permissions', async () => {
    let written: { url: string; body: unknown } | undefined;
    stubApi((url, body) => (written = { url, body }));
    renderRoles();
    await screen.findByText('Кассир');

    fireEvent.click(screen.getByRole('button', { name: '+ Создать роль' }));
    const dialog = await screen.findByRole('dialog', { name: 'Новая роль' });
    fireEvent.change(within(dialog).getByLabelText('Название'), { target: { value: 'Кладовщик' } });
    fireEvent.click(await within(dialog).findByText('Склад: просмотр'));
    fireEvent.click(within(dialog).getByRole('button', { name: 'Создать' }));

    await waitFor(() => expect(written).toBeDefined());
    expect(written!.url).toContain('/identity/companies/c-1/roles');
    expect(written!.body).toEqual({ name: 'Кладовщик', permissionKeys: ['inventory.read'] });
  });
});
