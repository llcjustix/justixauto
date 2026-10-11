// @vitest-environment jsdom
import { QueryClient, QueryClientProvider } from '@tanstack/react-query';
import { cleanup, fireEvent, render, screen, waitFor } from '@testing-library/react';
import { MemoryRouter } from 'react-router-dom';
import { afterEach, expect, it, vi } from 'vitest';
import { SessionGate } from '@justixauto/kit';
import { navigation } from './navigation';
import { BillingPage } from './pages/billing';

const accountantPermissions = [
  'retail.read', 'retail.invoices.manage', 'retail.payments.accept',
  'commerce.read', 'commerce.invoices.manage', 'commerce.payments.accept',
  'documents.read', 'documents.upload', 'documents.sensitive.download',
];
const money = { amountMinor: '1000', currency: 'USD' };
const response = (body: unknown) => new Response(JSON.stringify(body), { headers: { 'Content-Type': 'application/json' } });
const session = (permissions: string[]) => ({ data: { user: { id: 'accountant', displayName: 'Бухгалтер', status: 'active', passwordChangeRequired: false }, roles: [], permissions,
  context: { revision: '1', companyId: 'company', branchScope: { mode: 'ALL', branchIds: [] } },
  accessibleCompanies: [{ id: 'company', name: 'Компания', kind: 'seller', access: 'full', userType: 'dealer', verification: 'verified', verificationReason: '' }], setup: { next: 'none' } } });

afterEach(() => { cleanup(); vi.unstubAllGlobals(); });

it('gives the accountant surface to the minimal prepared role, including one grouped review', async () => {
  expect(accountantPermissions).toContain('retail.invoices.manage');
  expect(accountantPermissions).toContain('commerce.invoices.manage');
  expect(accountantPermissions).not.toContain('retail.deals.manage');
  expect(accountantPermissions).not.toContain('commerce.trade');
  expect(navigation.find((item) => item.to === '/billing')?.permission).toEqual(['retail.read', 'commerce.read']);

  const group = { id: 'group-1', dealId: 'issued', installmentPlanId: 'plan', claimedAmount: money, paidOn: '2026-10-01', externalReference: 'CHECK-1', attachmentIds: [], status: 'submitted', decisionReason: '', revision: 'group-r1', allowedActions: ['accept', 'reject'], allocations: [{ invoiceId: 'invoice-1', evidenceId: 'evidence-1', number: 1, amount: money }] };
  const waiting = { id: 'waiting', customer: { id: 'customer-1', displayName: 'Ждёт счёт', phone: '', revision: '1' }, status: 'reserved', invoices: [], revision: '1' };
  const issued = { id: 'issued', customer: { id: 'customer-2', displayName: 'Проверка чека', phone: '', revision: '1' }, status: 'reserved', invoices: [
    { id: 'invoice-1', dealId: 'issued', purpose: 'monthly-installment', installmentNumber: 1, amount: money, recipientSnapshot: '', dueDate: null, status: 'issued', paid: { ...money, amountMinor: '0' }, pending: money, outstanding: money, available: { ...money, amountMinor: '0' }, allowedActions: [], paymentEvidence: [], paymentGroups: [group], revision: '1' },
    { id: 'invoice-2', dealId: 'issued', purpose: 'monthly-installment', installmentNumber: 2, amount: money, recipientSnapshot: '', dueDate: null, status: 'issued', paid: { ...money, amountMinor: '0' }, pending: money, outstanding: money, available: { ...money, amountMinor: '0' }, allowedActions: [], paymentEvidence: [], paymentGroups: [group], revision: '1' },
  ], revision: '1' };
  const posts: string[] = [];
  vi.stubGlobal('fetch', vi.fn(async (input: RequestInfo | URL, init: RequestInit = {}) => {
    const url = String(input);
    if (url.endsWith('/identity/session')) return response(session(accountantPermissions));
    if (init.method === 'POST') { posts.push(url); return response({ data: group, revision: 'group-r2' }); }
    if (url.includes('/commerce/orders')) return response({ items: [] });
    if (url.endsWith('/retail/deals?limit=100')) return response({ items: [waiting, issued] });
    if (url.endsWith('/retail/deals/waiting')) return response({ data: waiting });
    if (url.endsWith('/retail/deals/issued')) return response({ data: issued });
    return response({ items: [] });
  }));
  const client = new QueryClient({ defaultOptions: { queries: { retry: false, staleTime: Infinity } } });
  render(<QueryClientProvider client={client}><MemoryRouter><SessionGate title="Тест"><BillingPage /></SessionGate></MemoryRouter></QueryClientProvider>);
  fireEvent.click(await screen.findByRole('tab', { name: 'От клиентов' }));
  expect(await screen.findByText(/Ждёт счёт ждёт выставления счёта/)).toBeTruthy();
  expect(screen.getAllByRole('button', { name: 'Проверить весь чек' })).toHaveLength(1);
  fireEvent.click(screen.getByRole('button', { name: 'Проверить весь чек' }));
  fireEvent.click(await screen.findByLabelText('Деньги поступили'));
  fireEvent.click(screen.getByRole('button', { name: 'Принять весь чек' }));
  await waitFor(() => expect(posts).toEqual(['/api/v1/retail/installment-payments/group-1/accept']));
});

it.each([
  ['retail-only', ['retail.read'], '/retail/deals?limit=100', '/commerce/orders'],
  ['commerce-only', ['commerce.read'], '/commerce/orders?limit=100', '/retail/deals'],
])('keeps %s Billing queries inside its permitted module', async (_name, permissions, expected, forbidden) => {
  const requests: string[] = [];
  vi.stubGlobal('fetch', vi.fn(async (input: RequestInfo | URL) => {
    const url = String(input); requests.push(url);
    if (url.endsWith('/identity/session')) return response(session(permissions));
    if (url.includes('/commerce/orders')) return response({ items: [] });
    if (url.includes('/retail/deals')) return response({ items: [] });
    return response({ items: [] });
  }));
  const client = new QueryClient({ defaultOptions: { queries: { retry: false, staleTime: Infinity } } });
  render(<QueryClientProvider client={client}><MemoryRouter><SessionGate title="Тест"><BillingPage /></SessionGate></MemoryRouter></QueryClientProvider>);
  await waitFor(() => expect(requests.some((url) => url.includes(expected))).toBe(true));
  expect(requests.some((url) => url.includes(forbidden))).toBe(false);
});
