// @vitest-environment jsdom
import { createElement } from 'react';
import { QueryClient, QueryClientProvider } from '@tanstack/react-query';
import { cleanup, fireEvent, render, screen, waitFor } from '@testing-library/react';
import { BrowserRouter, MemoryRouter, useLocation } from 'react-router-dom';
import { afterEach, expect, it, vi } from 'vitest';
import { PortalRoutes } from './portal';
import type { PortalSection } from './portal';
import { SessionGate } from './session';
import type { Company } from './session';

const seller: Company = { id: 'S', name: 'Seller', kind: 'seller', access: 'full', userType: 'dealer', verification: 'unverified', verificationReason: '' };
const bank: Company = { id: 'B', name: 'Bank', kind: 'bank', access: 'full', userType: null, verification: 'unverified', verificationReason: '' };
const insurer: Company = { id: 'I', name: 'Insurer', kind: 'insurance', access: 'full', userType: null, verification: 'unverified', verificationReason: '' };
const companies = [seller, bank, insurer];
const response = (body: unknown, status = 200) => new Response(JSON.stringify(body), { status });
function session(companyId: string | null, permissions: string[], memberships = companies) {
  return { data: {
    user: { id: 'u', displayName: 'User', status: 'active', passwordChangeRequired: false }, roles: [], permissions,
    context: { revision: companyId ?? 'platform', companyId, branchScope: { mode: 'ALL', branchIds: [] } },
    accessibleCompanies: memberships, setup: { next: 'none' },
  } };
}
function LocationLine() {
  const location = useLocation();
  return createElement('output', { 'aria-label': 'Location' }, `${location.pathname}${location.search}${location.hash}`);
}
function node(text: string, onRender?: () => void) {
  return createElement(Page, { text, onRender });
}
function Page({ text, onRender }: { text: string; onRender?: () => void }) {
  onRender?.();
  return createElement('p', {}, text);
}
const sections = (forbidden?: () => void): PortalSection[] => [
  { id: 'realization', prefix: '/', brand: 'Seller section', kinds: ['seller'], branches: false, nav: [
    { to: '/dashboard', label: 'Dashboard', element: node('Seller page') },
  ] },
  { id: 'financing', prefix: '/finance', brand: 'Finance section', kinds: ['bank', 'mfo'], nav: [
    { to: '/finance/applications', label: 'Applications', permission: 'financing.read', element: node('Finance page', forbidden) },
    { to: '/finance/settings', label: 'Settings', element: node('Finance settings') },
  ], search: { placeholder: 'Search finance', path: (q) => `/finance/applications?q=${encodeURIComponent(q)}` } },
  { id: 'insurance', prefix: '/insurance', brand: 'Insurance section', kinds: ['insurance'], variant: 'insurance', nav: [
    { to: '/insurance/overview', label: 'Overview', permission: 'insurance.read', element: node('Insurance page') },
    { to: '/insurance/settings', label: 'Settings', element: node('Insurance settings') },
  ] },
  { id: 'admin', prefix: '/admin', brand: 'Admin section', variant: 'admin', nav: [
    { to: '/admin/overview', label: 'Admin overview', permission: 'platform.directory.read', element: node('Admin page') },
  ] },
];
function mount(path: string, list = sections()) {
  const client = new QueryClient({ defaultOptions: { queries: { retry: false }, mutations: { retry: false } } });
  render(createElement(QueryClientProvider, { client },
    createElement(SessionGate, { title: 'Test' },
      createElement(MemoryRouter, { initialEntries: [path] },
        createElement(LocationLine), createElement(PortalRoutes, { sections: list })))));
}
afterEach(() => { cleanup(); vi.unstubAllGlobals(); localStorage.clear(); window.history.replaceState({}, '', '/'); });

it('restores an allowed deep URL with search/hash and renders no forbidden element', async () => {
  vi.stubGlobal('fetch', vi.fn((input: RequestInfo | URL) => {
    if (String(input).endsWith('/identity/session')) return Promise.resolve(response(session('B', ['financing.read'])));
    throw new Error(String(input));
  }));
  mount('/finance/applications?q=abc#part');
  expect(await screen.findByText('Finance page')).toBeTruthy();
  expect(screen.getByLabelText('Location').textContent).toBe('/finance/applications?q=abc#part');
});

it('chooses remembered accessible home at root and ignores revoked history', async () => {
  localStorage.setItem('justixauto:last-section:v1:u:B', 'admin');
  vi.stubGlobal('fetch', vi.fn(() => Promise.resolve(response(session('B', ['platform.directory.read'])))));
  mount('/');
  expect(await screen.findByText('Admin page')).toBeTruthy();
  expect(screen.getByLabelText('Location').textContent).toBe('/admin/overview');
  cleanup();
  vi.stubGlobal('fetch', vi.fn(() => Promise.resolve(response(session('B', [])))));
  mount('/');
  expect(await screen.findByText('Finance settings')).toBeTruthy();
  expect(screen.getByLabelText('Location').textContent).toBe('/finance/settings');
});

it('never renders a denied nested page and redirects to an allowed home', async () => {
  let forbidden = 0;
  vi.stubGlobal('fetch', vi.fn(() => Promise.resolve(response(session('B', [])))));
  mount('/finance/applications/123', sections(() => { forbidden += 1; }));
  expect(await screen.findByText('Finance settings')).toBeTruthy();
  expect(forbidden).toBe(0);
  expect(screen.getByLabelText('Location').textContent).toBe('/finance/settings');
});

it('waits for company PUT and refreshed session before admitting the requested page', async () => {
  let current = 'S';
  let release!: (value: Response) => void;
  const put = new Promise<Response>((resolve) => { release = resolve; });
  const requests: string[] = [];
  vi.stubGlobal('fetch', vi.fn((input: RequestInfo | URL) => {
    const path = String(input); requests.push(path);
    if (path.endsWith('/identity/session/context')) return put;
    if (path.endsWith('/identity/session')) return Promise.resolve(response(session(current, current === 'B' ? ['financing.read'] : [])));
    throw new Error(path);
  }));
  mount('/finance/applications?q=abc#part');
  await waitFor(() => expect(requests.some((path) => path.endsWith('/identity/session/context'))).toBe(true));
  expect(screen.queryByText('Finance page')).toBeNull();
  expect(screen.queryByText('Seller page')).toBeNull();
  current = 'B';
  release(response({ data: null }));
  expect(await screen.findByText('Finance page')).toBeTruthy();
  expect(screen.getByLabelText('Location').textContent).toBe('/finance/applications?q=abc#part');
});

it('requires an explicit company choice when multiple memberships match', async () => {
  const otherBank = { ...bank, id: 'B2', name: 'Other Bank' };
  let current = 'S';
  const requests: string[] = [];
  vi.stubGlobal('fetch', vi.fn((input: RequestInfo | URL) => {
    const path = String(input); requests.push(path);
    if (path.endsWith('/identity/session/context')) { current = 'B2'; return Promise.resolve(response({ data: null })); }
    if (path.endsWith('/identity/session')) return Promise.resolve(response(session(current, ['financing.read'], [seller, bank, otherBank])));
    throw new Error(path);
  }));
  mount('/finance/applications?q=1', sections());
  expect(await screen.findByText('Выберите компанию')).toBeTruthy();
  expect(requests.filter((path) => path.endsWith('/identity/session/context'))).toHaveLength(0);
  fireEvent.click(screen.getByRole('button', { name: 'Other Bank' }));
  expect(await screen.findByText('Finance page')).toBeTruthy();
  expect(screen.getByLabelText('Location').textContent).toBe('/finance/applications?q=1');
});

it('keeps a failed company selection out of the requested route and shows retry', async () => {
  vi.stubGlobal('fetch', vi.fn((input: RequestInfo | URL) => String(input).endsWith('/identity/session/context')
    ? Promise.resolve(response({ error: { code: 'conflict', message: 'Conflict' } }, 412))
    : Promise.resolve(response(session('S', ['financing.read'])))));
  mount('/finance/applications');
  expect(await screen.findByText('Нет связи')).toBeTruthy();
  expect(screen.queryByText('Finance page')).toBeNull();
  expect(screen.getByRole('button', { name: 'Повторить' })).toBeTruthy();
});

it('uses a final absolute search URL inside the finance section', async () => {
  vi.stubGlobal('fetch', vi.fn(() => Promise.resolve(response(session('B', ['financing.read'])))));
  mount('/finance/settings');
  expect(await screen.findByText('Finance settings')).toBeTruthy();
  fireEvent.change(screen.getByLabelText('Глобальный поиск'), { target: { value: 'VIN 123' } });
  fireEvent.submit(screen.getByRole('search'));
  expect(await screen.findByText('Finance page')).toBeTruthy();
  expect(screen.getByLabelText('Location').textContent).toBe('/finance/applications?q=VIN%20123');
});

it('selects an inactive company through the switcher without a document reload', async () => {
  let current: string | null = null;
  const requests: string[] = [];
  vi.stubGlobal('fetch', vi.fn((input: RequestInfo | URL) => {
    const path = String(input); requests.push(path);
    if (path.endsWith('/identity/session/context')) { current = 'B'; return Promise.resolve(response({ data: null })); }
    if (path.endsWith('/identity/session')) return Promise.resolve(response(session(current, ['platform.directory.read', 'financing.read'])));
    throw new Error(path);
  }));
  window.history.replaceState({}, '', '/admin/overview');
  const client = new QueryClient({ defaultOptions: { queries: { retry: false } } });
  render(createElement(QueryClientProvider, { client },
    createElement(SessionGate, { title: 'Test' },
      createElement(BrowserRouter, {}, createElement(PortalRoutes, { sections: sections() })))));
  expect(await screen.findByText('Admin page')).toBeTruthy();
  fireEvent.click(screen.getByRole('button', { name: 'Выбрать компанию: Bank · Finance section' }));
  expect(await screen.findByText('Finance page')).toBeTruthy();
  expect(window.location.pathname).toBe('/finance/applications');
  expect(requests.filter((path) => path.endsWith('/identity/session/context'))).toHaveLength(1);
});

it('hides the switcher for a sole platform Admin without company memberships', async () => {
  vi.stubGlobal('fetch', vi.fn(() => Promise.resolve(response(session(null, ['platform.directory.read'], [])))));
  mount('/admin/overview');
  expect(await screen.findByText('Admin page')).toBeTruthy();
  expect(screen.queryByRole('navigation', { name: 'Разделы' })).toBeNull();
});

it('hides the switcher for a sole active-company section without alternatives', async () => {
  vi.stubGlobal('fetch', vi.fn(() => Promise.resolve(response(session('B', [], [bank])))));
  mount('/finance/settings');
  expect(await screen.findByText('Finance settings')).toBeTruthy();
  expect(screen.queryByRole('navigation', { name: 'Разделы' })).toBeNull();
});

it.each([
  ['S', ['platform.directory.read'], '/dashboard', 'Seller page', [seller]],
  ['B', ['platform.directory.read', 'financing.read'], '/finance/applications', 'Finance page', [bank]],
  ['I', ['platform.directory.read'], '/insurance/settings', 'Insurance settings', [insurer]],
] as const)('provides an internal section switcher for %s with multiple permitted sections', async (active, permissions, path, page, memberships) => {
  vi.stubGlobal('fetch', vi.fn(() => Promise.resolve(response(session(active, [...permissions], memberships)))));
  mount(path);
  expect(await screen.findByText(page)).toBeTruthy();
  const switcher = screen.getByRole('navigation', { name: 'Разделы' });
  expect(switcher.querySelector('a[href="/admin/overview"]')).not.toBeNull();
  expect(switcher.querySelector('a[href^="http"]')).toBeNull();
  expect(switcher.querySelector('button')).toBeNull();
  const current = screen.getByRole('link', { name: active === 'S' ? 'Seller section' : active === 'B' ? 'Finance section' : 'Insurance section' });
  expect(current.classList.contains('active')).toBe(true);
  fireEvent.click(screen.getByRole('link', { name: 'Admin section' }));
  expect(await screen.findByText('Admin page')).toBeTruthy();
  expect(screen.getByLabelText('Location').textContent).toBe('/admin/overview');
});
