// @vitest-environment jsdom
import { QueryClient, QueryClientProvider } from '@tanstack/react-query';
import { SessionGate, useSession } from '@justixauto/kit';
import { cleanup, fireEvent, render, screen, waitFor } from '@testing-library/react';
import { afterEach, expect, it, vi } from 'vitest';
import { OrderCreate } from './order-create';
import { OrderActionButton } from './order-actions';
import { NewSale } from './sale-form';

const response = (body: unknown) => new Response(JSON.stringify(body), { headers: { 'Content-Type': 'application/json' } });

function SwitchCompany() {
  const session = useSession();
  return <button onClick={() => void session.refresh()}>Обновить компанию</button>;
}

function setup() {
  const state = { verified: false, posts: 0 };
  vi.stubGlobal('fetch', vi.fn(async (input: RequestInfo | URL, init: RequestInit = {}) => {
    if (String(input).endsWith('/identity/session')) return response({ data: {
      user: { id: 'u', displayName: 'Пользователь', status: 'active', passwordChangeRequired: false }, roles: [], permissions: [],
      context: { revision: state.verified ? '2' : '1', companyId: state.verified ? 'verified' : 'unverified', branchScope: { mode: 'ALL', branchIds: [] } },
      accessibleCompanies: [
        { id: 'unverified', name: 'На проверке', kind: 'seller', access: 'full', userType: 'dealer', verification: 'returned', verificationReason: 'Проверьте ИНН' },
        { id: 'verified', name: 'Подтверждена', kind: 'seller', access: 'full', userType: 'dealer', verification: 'verified', verificationReason: '' },
      ], setup: { next: 'none' },
    } });
    if (init.method === 'POST') state.posts += 1;
    return response({ data: { id: 'created' } });
  }));
  const client = new QueryClient({ defaultOptions: { queries: { retry: false }, mutations: { retry: false } } });
  return { state, wrap: (node: React.ReactNode) => render(<QueryClientProvider client={client}><SessionGate title="Тест">{node}</SessionGate></QueryClientProvider>) };
}

afterEach(() => { cleanup(); vi.unstubAllGlobals(); });

it('blocks direct sale and order entry for a returned seller without posting', async () => {
  const { state, wrap } = setup();
  wrap(<><NewSale autoOpen /><OrderCreate mode={{ kind: 'direct' }} onClose={vi.fn()} onCreated={vi.fn()} /></>);
  expect((await screen.findAllByText(/Проверка компании возвращена/)).length).toBeGreaterThanOrEqual(2);
  expect(screen.getAllByRole('link', { name: 'Открыть настройки компании' }).length).toBeGreaterThanOrEqual(2);
  expect(state.posts).toBe(0);
});

it('refreshing the selected company enables new sales while servicing remains available', async () => {
  const { state, wrap } = setup();
  const serviced = vi.fn().mockResolvedValue({});
  wrap(<><SwitchCompany /><NewSale /><OrderActionButton label="Сообщить об оплате" onSubmit={serviced} /></>);
  const sale = await screen.findByRole('button', { name: 'Новая продажа' });
  expect((sale as HTMLButtonElement).disabled).toBe(true);
  fireEvent.click(screen.getByRole('button', { name: 'Сообщить об оплате' }));
  fireEvent.click((await screen.findAllByRole('button', { name: 'Сообщить об оплате' })).at(-1)!);
  await waitFor(() => expect(serviced).toHaveBeenCalledTimes(1));
  state.verified = true;
  fireEvent.click(screen.getByRole('button', { name: 'Обновить компанию' }));
  await waitFor(() => expect((screen.getByRole('button', { name: 'Новая продажа' }) as HTMLButtonElement).disabled).toBe(false));
});
