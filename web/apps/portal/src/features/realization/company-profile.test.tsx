import { createElement } from 'react';
import { QueryClient, QueryClientProvider } from '@tanstack/react-query';
import { cleanup, fireEvent, render, screen, waitFor, within } from '@testing-library/react';
import { afterEach, describe, expect, it, vi } from 'vitest';

const session = {
  company: { id: 'company-1', kind: 'seller' },
  can: (permission: string): boolean => permission === 'company.edit',
  refresh: vi.fn(async () => undefined),
};

vi.mock('@justixauto/kit', async (importOriginal) => ({
  ...(await importOriginal<typeof import('@justixauto/kit')>()),
  useSession: () => session,
  useScopeLabel: () => 'Реализация',
  CompanyEmployees: () => null,
  CompanyRoles: () => null,
}));
vi.mock('../data', () => ({
  useBranches: () => ({ data: [], isLoading: false, error: null }),
  useWarehouses: () => ({ data: [], isLoading: false, error: null }),
}));
vi.mock('./inventory', () => ({ WarehouseDialog: () => null }));

const { SettingsPage } = await import('./pages/settings');

const company = {
  id: 'company-1',
  name: 'Авто плюс',
  legalName: 'Авто плюс ООО',
  legalForm: 'other',
  country: { label: 'Казахстан' },
  region: null,
  registration: '',
  email: 'info@example.test',
  address: 'Алматы',
  phone: '',
  access: 'active',
  userType: 'dealer',
  verification: 'verified',
  verificationReason: '',
  taxId: '',
  bankRequisites: '',
};

afterEach(() => {
  cleanup();
  vi.unstubAllGlobals();
  session.can = (permission: string) => permission === 'company.edit';
  session.refresh.mockClear();
});

function renderSettings(canEdit = true, profile = company) {
  session.can = (permission: string) => canEdit && permission === 'company.edit';
  const fetch = vi.fn(async (input: RequestInfo | URL, init?: RequestInit) => {
    if (String(input).endsWith('/identity/companies/company-1')) {
      if (init?.method === 'PATCH')
        return new Response(JSON.stringify({ data: profile, revision: '8' }), {
          status: 200,
          headers: { 'Content-Type': 'application/json' },
        });
      return new Response(JSON.stringify({ data: profile, revision: '7' }), {
        status: 200,
        headers: { 'Content-Type': 'application/json' },
      });
    }
    throw new Error(`Unexpected request ${String(input)}`);
  });
  vi.stubGlobal('fetch', fetch);
  render(
    createElement(
      QueryClientProvider,
      { client: new QueryClient({ defaultOptions: { queries: { retry: false } } }) },
      createElement(SettingsPage),
    ),
  );
  return fetch;
}

describe('Realization company legal form', () => {
  it('shows, prefills and preserves legal form while saving an unrelated edit', async () => {
    const fetch = renderSettings();
    expect(await screen.findByText('Другое')).toBeTruthy();
    fireEvent.click(screen.getByRole('button', { name: 'Изменить данные' }));
    const dialog = await screen.findByRole('dialog', { name: 'Изменить данные' });
    expect((within(dialog).getByLabelText('Организационная форма') as HTMLInputElement).value).toBe('Другое');
    fireEvent.change(within(dialog).getByLabelText('Название'), { target: { value: 'Новое название' } });
    fireEvent.click(within(dialog).getByRole('button', { name: 'Изменить данные' }));

    await waitFor(() =>
      expect(fetch).toHaveBeenCalledWith(expect.stringContaining('/identity/companies/company-1'), expect.any(Object)),
    );
    const [, init] = fetch.mock.calls.find(([, request]) => request?.method === 'PATCH')!;
    expect(JSON.parse(String(init?.body))).toMatchObject({ name: 'Новое название', legalForm: 'other' });
    expect(new Headers(init?.headers).get('If-Match')).toBe('"7"');
  });

  it('shows the fallback and retains the existing edit gate', async () => {
    renderSettings(false, { ...company, legalForm: '' });
    expect(await screen.findByText('Не указана')).toBeTruthy();
    expect(screen.queryByRole('button', { name: 'Изменить данные' })).toBeNull();
  });
});
