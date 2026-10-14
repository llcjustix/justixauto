// @vitest-environment jsdom
import { QueryClient, QueryClientProvider } from '@tanstack/react-query';
import { cleanup, fireEvent, render, screen, waitFor, within } from '@testing-library/react';
import { MemoryRouter } from 'react-router-dom';
import { afterEach, expect, it, vi } from 'vitest';
import { SessionGate } from '@justixauto/kit';
import { navigation } from './navigation';
import { BillingPage } from './pages/billing';
import { InvoicePanel } from './pages/trade';
import { FinanceWorkspace } from '../accounting/finance-workspace';

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
  const waiting = { id: 'waiting', customer: { id: 'customer-1', displayName: 'Ждёт счёт', phone: '', revision: '1' }, status: 'reserved', invoices: [], allowedActions: ['issue-invoice'], revision: '1' };
  const issued = { id: 'issued', customer: { id: 'customer-2', displayName: 'Проверка чека', phone: '', revision: '1' }, status: 'reserved', invoices: [
    { id: 'invoice-1', dealId: 'issued', purpose: 'monthly-installment', installmentNumber: 1, amount: money, recipientSnapshot: '', dueDate: null, status: 'issued', paid: { ...money, amountMinor: '0' }, pending: money, outstanding: money, available: { ...money, amountMinor: '0' }, allowedActions: [], paymentEvidence: [], paymentGroups: [group], revision: '1' },
    { id: 'invoice-2', dealId: 'issued', purpose: 'monthly-installment', installmentNumber: 2, amount: money, recipientSnapshot: '', dueDate: null, status: 'issued', paid: { ...money, amountMinor: '0' }, pending: money, outstanding: money, available: { ...money, amountMinor: '0' }, allowedActions: [], paymentEvidence: [], paymentGroups: [group], revision: '1' },
  ], allowedActions: ['issue-invoice'], revision: '1' };
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
  render(<QueryClientProvider client={client}><MemoryRouter><SessionGate title="Тест"><FinanceWorkspace><BillingPage /></FinanceWorkspace></SessionGate></MemoryRouter></QueryClientProvider>);
  fireEvent.click(await screen.findByRole('tab', { name: 'От клиентов' }));
  expect(await screen.findByRole('heading', { name: 'Ждёт счёт' })).toBeTruthy();
  expect(screen.getAllByRole('heading', { name: 'Проверка чека' })).toHaveLength(2);
  expect(screen.getAllByRole('button', { name: 'Проверить весь чек' })).toHaveLength(1);
  fireEvent.click(screen.getByRole('button', { name: 'Проверить весь чек' }));
  fireEvent.click(await screen.findByLabelText('Деньги поступили'));
  fireEvent.click(screen.getByRole('button', { name: 'Принять весь чек' }));
  await waitFor(() => expect(posts).toEqual(['/api/v1/retail/installment-payments/group-1/accept']));
});

it('keeps B2B invoice issuance reachable for an eligible order without invoices', async () => {
  let issued = false;
  const order = { id: 'b2b-issue', party: 'supplier', buyer: { id: 'buyer', name: 'Покупатель' }, supplier: { id: 'seller', name: 'Продавец' }, source: 'direct', terms: { lines: [], route: 'local', deliveryTerms: '', warrantyTerms: '', serviceTerms: '', paymentSchedule: [] }, total: money, status: 'accepted', statusReason: '', revision: 'order-r1', updatedAt: '', allocations: [], shipments: [], addenda: [], allowedActions: [], receiptBatches: [], receivingWarehouseId: null, hasReceivingWarehouse: false, history: [] };
	const posts: { url: string; revision: string | null }[] = [];
	vi.stubGlobal('fetch', vi.fn(async (input: RequestInfo | URL, init: RequestInit = {}) => {
    const url = String(input);
    if (url.endsWith('/identity/session')) return response(session(accountantPermissions));
	if (init.method === 'POST') { posts.push({ url, revision: new Headers(init.headers).get('If-Match') }); issued = true; return response({ data: { ...order, revision: 'order-r2' }, revision: 'order-r2' }); }
    if (url.endsWith('/commerce/orders?limit=100')) return response({ items: [order] });
    if (url.endsWith('/commerce/orders/b2b-issue')) return response({ data: order });
    if (url.endsWith('/commerce/orders/b2b-issue/invoices')) return response({ items: issued ? [{ id: 'issued-invoice', orderId: order.id, total: money, paid: { ...money, amountMinor: '0' }, pending: { ...money, amountMinor: '0' }, outstanding: money, status: 'issued', schedule: [], paymentEvidence: [], allowedActions: [], revision: 'invoice-r1' }] : [] });
    return response({ items: [] });
  }));
  const client = new QueryClient({ defaultOptions: { queries: { retry: false, staleTime: Infinity } } });
  render(<QueryClientProvider client={client}><MemoryRouter><SessionGate title="Тест"><FinanceWorkspace><BillingPage /></FinanceWorkspace></SessionGate></MemoryRouter></QueryClientProvider>);
  expect(await screen.findByRole('button', { name: 'Выставить счёт' })).toBeTruthy();
  fireEvent.click(screen.getByRole('button', { name: 'Выставить счёт' }));
	const dialog = await screen.findByRole('dialog', { name: /Заказ/ });
	fireEvent.click(await within(dialog).findByRole('button', { name: 'Выставить счёт' }));
	const form = await screen.findByRole('dialog', { name: 'Выставить счёт' });
	fireEvent.change(within(form).getByLabelText('Срок оплаты'), { target: { value: '2026-10-30' } });
	fireEvent.click(within(form).getByRole('button', { name: 'Выставить счёт' }));
	await waitFor(() => expect(posts).toEqual([{ url: '/api/v1/commerce/orders/b2b-issue/invoices', revision: '"order-r1"' }]));
	await waitFor(() => expect(screen.queryByText('B2B-счета к выставлению')).toBeNull());
});

it('keeps commerce evidence decisions read-only outside Accounting despite allowed actions', () => {
  const invoice = { id: 'invoice-1', orderId: 'order-1', total: money, paid: { ...money, amountMinor: '0' }, pending: { ...money, amountMinor: '0' }, outstanding: money, status: 'issued', schedule: [], revision: 'invoice-r1', allowedActions: ['void'], paymentEvidence: [{ id: 'evidence-1', amount: money, paidOn: '2026-10-01', externalReference: 'BANK-1', attachmentIds: [], status: 'submitted', revision: 'evidence-r1', allowedActions: ['accept', 'reject'] }] };
  render(<InvoicePanel invoice={invoice} party="supplier" refresh={[]} />);
  expect(screen.queryByRole('button', { name: 'Аннулировать' })).toBeNull();
  expect(screen.queryByRole('button', { name: 'Принять' })).toBeNull();
  expect(screen.queryByRole('button', { name: 'Отклонить' })).toBeNull();
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
