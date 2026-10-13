// @vitest-environment jsdom
import { createElement, useState } from 'react';
import { QueryClient, QueryClientProvider, useMutation, useQuery } from '@tanstack/react-query';
import { cleanup, fireEvent, render, screen, waitFor } from '@testing-library/react';
import { afterEach, expect, it, vi } from 'vitest';
import { get, post } from './http';
import { SessionGate, useSession } from './session';
import type { Company } from './session';

function deferred<T>() {
  let resolve!: (value: T) => void;
  const promise = new Promise<T>((done) => { resolve = done; });
  return { promise, resolve };
}

const response = (body: unknown, status = 200) => new Response(JSON.stringify(body), { status });
function session(user = 'u', company: string | null = 'A', branchIds: string[] = [], permissions = ['read'], passwordChangeRequired = false) {
  return { data: {
    user: { id: user, displayName: user, status: 'active', passwordChangeRequired }, roles: [], permissions,
    context: { revision: company === 'A' ? 'r-A' : 'r-B', companyId: company, branchScope: { mode: branchIds.length ? 'SELECTED' : 'ALL', branchIds } },
    accessibleCompanies: ['A', 'B'].map((id) => ({ id, name: id, kind: 'seller', access: 'full', userType: 'dealer', verification: 'verified', verificationReason: '' })),
    setup: { next: 'none' },
  } };
}

function mount(node: React.ReactNode, kinds?: Company['kind'][]) {
  const client = new QueryClient({ defaultOptions: { queries: { retry: false }, mutations: { retry: false } } });
  render(createElement(QueryClientProvider, { client }, createElement(SessionGate, { title: 'Test', kinds }, node)));
  return client;
}

function Workspace() {
  const s = useSession();
  const [dialog, setDialog] = useState(true);
  const query = useQuery({ queryKey: ['same-key'], queryFn: async () => (await get<{ label: string }>('/records')).data });
  const mutation = useMutation({ mutationFn: () => post('/action'), onSuccess: () => setDialog(true) });
  return createElement('div', {},
    createElement('p', {}, `Identity ${s.view.user.id} / ${s.company?.id ?? 'none'} / ${s.scopeKey}`),
    createElement('p', {}, `Data ${query.data?.label ?? 'loading'}`),
    createElement('button', { onClick: () => void s.selectCompany('B').catch(() => {}) }, 'Switch B'),
    createElement('button', { onClick: () => void s.setBranchScope('SELECTED', ['branch-1']).catch(() => {}) }, 'Select branch'),
    createElement('button', { onClick: () => void s.refresh().catch(() => {}) }, 'Refresh'),
    createElement('button', { onClick: () => void s.logout().catch(() => {}) }, 'Logout'),
    createElement('button', { onClick: () => void get('/denied').catch(() => {}) }, 'Request denied'),
    dialog && createElement('div', { role: 'dialog' },
      createElement('button', { onClick: () => mutation.mutate() }, 'Submit action'),
      createElement('button', { onClick: () => setDialog(false) }, 'Close dialog')),
  );
}

afterEach(() => { cleanup(); vi.unstubAllGlobals(); });

it('unmounts the old dialog and cache before company PUT resolves, then rejects late old data and mutation', async () => {
  let company = 'A';
  const put = deferred<Response>();
  const oldData = deferred<Response>();
  const oldMutation = deferred<Response>();
  const requests: { path: string; init: RequestInit }[] = [];
  vi.stubGlobal('fetch', vi.fn((input: RequestInfo | URL, init: RequestInit = {}) => {
    const path = String(input); requests.push({ path, init });
    if (path.endsWith('/identity/session')) return Promise.resolve(response(session('u', company)));
    if (path.endsWith('/identity/session/context')) return put.promise;
    if (path.endsWith('/records')) return company === 'A' ? oldData.promise : Promise.resolve(response({ data: { label: 'B' } }));
    if (path.endsWith('/action')) return oldMutation.promise;
    throw new Error(path);
  }));
  const client = mount(createElement(Workspace));
  fireEvent.click(await screen.findByRole('button', { name: 'Submit action' }));
  await waitFor(() => expect(requests.some((r) => r.path.endsWith('/action'))).toBe(true));
  fireEvent.click(screen.getByRole('button', { name: 'Switch B' }));
  expect(screen.queryByRole('dialog')).toBeNull();
  expect(screen.queryByText(/Identity u \/ A/)).toBeNull();
  expect(client.getQueryData(['same-key'])).toBeUndefined();
  expect(requests.find((r) => r.path.endsWith('/identity/session/context'))?.init.headers).toBeInstanceOf(Headers);
  expect((requests.find((r) => r.path.endsWith('/identity/session/context'))!.init.headers as Headers).get('If-Match')).toBe('"r-A"');
  company = 'B';
  put.resolve(response({ data: null }));
  expect(await screen.findByText(/Data B/)).toBeTruthy();
  oldData.resolve(response({ data: { label: 'A' } }));
  oldMutation.resolve(response({ data: { id: 'old' } }));
  await waitFor(() => expect(client.getQueryData(['same-key'])).toEqual({ label: 'B' }));
  expect(screen.getByText(/Data B/)).toBeTruthy();
});

it('keeps an uncertain company transition out of the old workspace', async () => {
  const put = deferred<Response>();
  vi.stubGlobal('fetch', vi.fn((input: RequestInfo | URL) => String(input).endsWith('/identity/session/context')
    ? put.promise : Promise.resolve(response(session()))));
  mount(createElement(Workspace));
  fireEvent.click(await screen.findByRole('button', { name: 'Switch B' }));
  expect(screen.queryByRole('dialog')).toBeNull();
  put.resolve(response({ error: { code: 'conflict', message: 'Conflict' } }, 412));
  expect(await screen.findByText('Нет связи')).toBeTruthy();
  expect(screen.queryByText(/Identity u \/ A/)).toBeNull();
});

it('does not admit an older session refresh after a newer permission change', async () => {
  const first = deferred<Response>();
  const second = deferred<Response>();
  let sessions = 0;
  vi.stubGlobal('fetch', vi.fn((input: RequestInfo | URL) => {
    if (String(input).endsWith('/identity/session')) {
      sessions += 1;
      return sessions === 1 ? Promise.resolve(response(session())) : sessions === 2 ? first.promise : second.promise;
    }
    return Promise.resolve(response({ data: { label: 'same' } }));
  }));
  mount(createElement(Workspace));
  fireEvent.click(await screen.findByRole('button', { name: 'Refresh' }));
  fireEvent.click(screen.getByRole('button', { name: 'Refresh' }));
  second.resolve(response(session('u', 'A', [], ['changed'])));
  await waitFor(() => expect(screen.getByText(/Identity u \/ A/)).toBeTruthy());
  first.resolve(response(session('old-user', 'B')));
  await waitFor(() => expect(screen.queryByText(/Identity old-user/)).toBeNull());
});

it('changes branch scope, clears the old dialog, and admits only the refreshed branch', async () => {
  let branch: string[] = [];
  const put = deferred<Response>();
  vi.stubGlobal('fetch', vi.fn((input: RequestInfo | URL, init: RequestInit = {}) => {
    const path = String(input);
    if (path.endsWith('/identity/session')) return Promise.resolve(response(session('u', 'A', branch)));
    if (path.endsWith('/identity/session/branch-scope')) {
      expect((init.headers as Headers).get('If-Match')).toBe('"r-A"');
      return put.promise;
    }
    return Promise.resolve(response({ data: { label: branch.length ? 'branch-1' : 'all' } }));
  }));
  const client = mount(createElement(Workspace));
  const before = (await screen.findByText(/Identity u \/ A/)).textContent;
  fireEvent.click(screen.getByRole('button', { name: 'Select branch' }));
  expect(screen.queryByRole('dialog')).toBeNull();
  expect(client.getQueryData(['same-key'])).toBeUndefined();
  branch = ['branch-1'];
  put.resolve(response({ data: null }));
  const after = (await screen.findByText(/Identity u \/ A/)).textContent;
  expect(after).not.toBe(before);
  expect(await screen.findByText('Data branch-1')).toBeTruthy();
});

it('leaves a sole company unselected when kinds are omitted', async () => {
  const fetch = vi.fn((input: RequestInfo | URL) => String(input).endsWith('/identity/session')
    ? Promise.resolve(response(session('admin', null)))
    : Promise.resolve(response({ data: { label: 'platform' } })));
  vi.stubGlobal('fetch', fetch);
  mount(createElement(Workspace));
  expect(await screen.findByText(/Identity admin \/ none/)).toBeTruthy();
  expect(fetch.mock.calls.some(([url]) => String(url).endsWith('/identity/session/context'))).toBe(false);
});

it('keeps sole-company selection for legacy kind-limited entries', async () => {
  let company: string | null = null;
  const fetch = vi.fn((input: RequestInfo | URL) => {
    const path = String(input);
    if (path.endsWith('/identity/session/context')) { company = 'A'; return Promise.resolve(response({ data: null })); }
    if (path.endsWith('/identity/session')) {
      const body = session('u', company);
      body.data.accessibleCompanies = body.data.accessibleCompanies.slice(0, 1);
      return Promise.resolve(response(body));
    }
    return Promise.resolve(response({ data: { label: 'A' } }));
  });
  vi.stubGlobal('fetch', fetch);
  mount(createElement(Workspace), ['seller']);
  expect(await screen.findByText(/Identity u \/ A/)).toBeTruthy();
  expect(fetch.mock.calls.filter(([url]) => String(url).endsWith('/identity/session/context'))).toHaveLength(1);
});

it('removes content when a refresh requires a password change', async () => {
  let forced = false;
  vi.stubGlobal('fetch', vi.fn((input: RequestInfo | URL) => {
    const path = String(input);
    if (path.endsWith('/identity/session')) return Promise.resolve(response(session('u', 'A', [], ['read'], forced)));
    return Promise.resolve(response({ data: { label: 'A' } }));
  }));
  const client = mount(createElement(Workspace));
  await screen.findByRole('button', { name: 'Refresh' });
  forced = true;
  fireEvent.click(screen.getByRole('button', { name: 'Refresh' }));
  expect(await screen.findByText('Смена пароля')).toBeTruthy();
  expect(screen.queryByRole('dialog')).toBeNull();
  expect(client.getQueryData(['same-key'])).toBeUndefined();
});

it('retires the current workspace on 401', async () => {
  vi.stubGlobal('fetch', vi.fn((input: RequestInfo | URL) => String(input).endsWith('/identity/session')
    ? Promise.resolve(response(session()))
    : String(input).endsWith('/denied')
      ? Promise.resolve(response({ error: { code: 'unauthorized' } }, 401))
      : Promise.resolve(response({ data: { label: 'A' } }))));
  const client = mount(createElement(Workspace));
  fireEvent.click(await screen.findByRole('button', { name: 'Request denied' }));
  expect(await screen.findByText('Вход')).toBeTruthy();
  expect(screen.queryByRole('dialog')).toBeNull();
  expect(client.getQueryData(['same-key'])).toBeUndefined();
});

it('clears old-user cache and admits a new login under the same query key', async () => {
  let user = 'first';
  const fetch = vi.fn((input: RequestInfo | URL, init: RequestInit = {}) => {
    const path = String(input);
    if (path.endsWith('/identity/session/login')) { user = 'second'; return Promise.resolve(response({ data: null })); }
    if (path.endsWith('/identity/session/logout')) return Promise.resolve(response({ data: null }));
    if (path.endsWith('/identity/session')) return Promise.resolve(response(session(user)));
    if (path.endsWith('/records')) return Promise.resolve(response({ data: { label: user } }));
    throw new Error(`${path} ${init.method}`);
  });
  vi.stubGlobal('fetch', fetch);
  const client = mount(createElement(Workspace));
  expect(await screen.findByText('Data first')).toBeTruthy();
  fireEvent.click(screen.getByRole('button', { name: 'Logout' }));
  expect(await screen.findByText('Вход')).toBeTruthy();
  expect(client.getQueryData(['same-key'])).toBeUndefined();
  fireEvent.change(screen.getByLabelText('Логин'), { target: { value: 'second' } });
  fireEvent.change(screen.getByLabelText('Пароль'), { target: { value: 'password' } });
  fireEvent.click(screen.getByRole('button', { name: 'Войти' }));
  expect(await screen.findByText('Data second')).toBeTruthy();
  expect(screen.queryByText('Data first')).toBeNull();
});
