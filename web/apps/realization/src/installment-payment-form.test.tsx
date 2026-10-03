// @vitest-environment jsdom
import { cleanup, fireEvent, render, screen, waitFor } from '@testing-library/react';
import { afterEach, expect, it, vi } from 'vitest';
import { InstallmentPaymentForm } from './installment-payment-form';
import type { InstallmentPlan, Money, RetailInvoice } from './data';

const m = (amountMinor: string): Money => ({ amountMinor, currency: 'USD' });
function invoice(id: string, number: number, amount: string, pending = '0'): RetailInvoice {
  return { id, dealId: 'deal-1', purpose: 'monthly-installment', installmentNumber: number, amount: m(amount), recipientSnapshot: '', dueDate: `2026-0${number}-01`, status: 'issued', paid: m('0'), pending: m(pending), outstanding: m(amount), available: m((BigInt(amount) - BigInt(pending)).toString()), allowedActions: ['submit-payment'], paymentEvidence: [], revision: 'invoice-rev' };
}
const invoices = [invoice('month-1', 1, '1000'), invoice('month-2', 2, '9007199254740993123', '23')];
const plan: InstallmentPlan = { id: 'plan-1', state: 'planned', allowedActions: ['submit-installment-payment'], contractReference: 'R', contractSignedOn: '2026-01-01', contractFileIds: [], contractTotal: m('9007199254740994123'), downPaymentInvoiceId: 'first', downPayment: m('1000'), scheduledTotal: m('9007199254740994123'), paid: m('0'), pending: m('23'), outstanding: m('9007199254740994123'), rows: invoices.map(i => ({ number: i.installmentNumber!, invoiceId: i.id, dueDate: i.dueDate!, amount: i.amount, paid: i.paid, pending: i.pending, outstanding: i.outstanding, available: i.available, allowedActions: i.allowedActions })) };
const response = (body: unknown, status = 201) => new Response(JSON.stringify(body), { status, headers: { 'Content-Type': 'application/json' } });
afterEach(() => { cleanup(); vi.unstubAllGlobals(); });

function setup(onSubmitted = vi.fn().mockResolvedValue(undefined), onBusyChange = vi.fn()) {
  const requests: unknown[] = [];
  vi.stubGlobal('fetch', vi.fn(async (input: RequestInfo | URL, init: RequestInit = {}) => {
    if (init.method === 'POST') { requests.push({ url: String(input), body: JSON.parse(String(init.body)) }); return response({ data: { id: 'payment' }, revision: 'payment-rev' }); }
    return response({ items: [] });
  }));
  render(<InstallmentPaymentForm dealId="deal-1" plan={plan} invoices={invoices} selectedInvoiceId="month-1" onBack={vi.fn()} onBusyChange={onBusyChange} onSubmitted={onSubmitted} />);
  return { requests, onSubmitted, onBusyChange };
}
const change = (label: string, value: string) => fireEvent.change(screen.getByLabelText(label), { target: { value } });

it('submits manually selected exact allocations in the contract camelCase body', async () => {
  const run = setup();
  fireEvent.click(screen.getByLabelText('Выбрать месяц 2'));
  change('Сумма месяца 1', '10'); change('Сумма месяца 2', '90071992547409931');
  change('Сумма по чеку', '90071992547409941'); change('Дата оплаты', '2026-10-01'); change('Номер платёжки / чека', ' RECEIPT-7 ');
  fireEvent.click(screen.getByRole('button', { name: 'Отправить на проверку' }));
  await waitFor(() => expect(run.onSubmitted).toHaveBeenCalledOnce());
  expect(run.onBusyChange.mock.calls).toEqual([[true], [false]]);
  expect(screen.getByText(/Подтверждение финансового сотрудника ещё не получено/)).toBeTruthy();
  expect(run.requests).toEqual([{ url: '/api/v1/retail/deals/deal-1/installment-payments', body: {
    claimedAmount: m('9007199254740994100'), paidOn: '2026-10-01', externalReference: 'RECEIPT-7', attachmentBindingIds: [],
    allocations: [{ invoiceId: 'month-1', amount: m('1000') }, { invoiceId: 'month-2', amount: m('9007199254740993100') }],
  } }]);
});

it.each(['plan', 'row', 'invoice', 'foreign', 'purpose', 'currency', 'cancelled'] as const)('excludes unauthorized or ineligible allocations: %s', mode => {
  const p = structuredClone(plan), list = structuredClone(invoices);
  if (mode === 'plan') p.allowedActions = undefined;
  if (mode === 'row') p.rows.forEach(row => { row.allowedActions = []; });
  if (mode === 'invoice') list.forEach(invoice => { invoice.allowedActions = []; });
  if (mode === 'foreign') list.forEach(invoice => { invoice.dealId = 'other'; });
  if (mode === 'purpose') list.forEach(invoice => { invoice.purpose = 'first-installment'; });
  if (mode === 'currency') list.forEach(invoice => { invoice.available.currency = 'EUR'; });
  if (mode === 'cancelled') p.state = 'cancelled';
  const fetch = vi.fn(); vi.stubGlobal('fetch', fetch);
  render(<InstallmentPaymentForm dealId="deal-1" plan={p} invoices={list} selectedInvoiceId="month-1" initialFullRemaining onBack={vi.fn()} onSubmitted={vi.fn()} />);
  expect(screen.queryByRole('button', { name: 'Отправить на проверку' })).toBeNull();
  expect(screen.queryByLabelText('Выбрать месяц 1')).toBeNull();
  expect(fetch).not.toHaveBeenCalled();
});

it('retains an uploaded binding after receipt failure and replaces it only for a new file', async () => {
  const calls: { url: string; body?: any }[] = [];
  vi.stubGlobal('fetch', vi.fn(async (input: RequestInfo | URL, init: RequestInit = {}) => {
    const url = String(input);
    if (url.endsWith('/documents/files')) { calls.push({ url }); return response({ data: { id: `binding-${calls.length}` } }); }
    calls.push({ url, body: JSON.parse(String(init.body)) });
    return response({ error: { code: 'bad', message: 'Receipt rejected', fields: {} } }, 422);
  }));
  render(<InstallmentPaymentForm dealId="deal-1" plan={plan} invoices={invoices} selectedInvoiceId="month-1" onBack={vi.fn()} onSubmitted={vi.fn()} />);
  change('Сумма месяца 1', '10'); change('Сумма по чеку', '10'); change('Дата оплаты', '2026-10-01'); change('Номер платёжки / чека', 'R');
  const fileInput = screen.getByLabelText('Подтверждение (файл)');
  fireEvent.change(fileInput, { target: { files: [new File(['proof'], 'proof.pdf')] } });
  fireEvent.click(screen.getByRole('button', { name: 'Отправить на проверку' })); await screen.findByText('Receipt rejected');
  fireEvent.click(screen.getByRole('button', { name: 'Отправить на проверку' })); await screen.findByText('Receipt rejected');
  expect(calls).toHaveLength(3);
  expect(calls[1]!.body.attachmentBindingIds).toEqual(['binding-1']); expect(calls[2]!.body.attachmentBindingIds).toEqual(['binding-1']);
  fireEvent.change(fileInput, { target: { files: [new File(['new'], 'new.pdf')] } });
  fireEvent.click(screen.getByRole('button', { name: 'Отправить на проверку' })); await screen.findByText('Receipt rejected');
  expect(calls).toHaveLength(5); expect(calls[4]!.body.attachmentBindingIds).toEqual(['binding-4']);
});

it('starts the explicit full-remaining preset with each available amount', () => {
  render(<InstallmentPaymentForm dealId="deal-1" plan={plan} invoices={invoices} initialFullRemaining onBack={vi.fn()} onSubmitted={vi.fn()} />);
  expect((screen.getByLabelText('Сумма месяца 1') as HTMLInputElement).value).toBe('10.00');
  expect((screen.getByLabelText('Сумма месяца 2') as HTMLInputElement).value).toBe('90071992547409931.00');
});

it('rejects mismatched or over-available drafts without a request and explains pending payoff capacity', () => {
  const run = setup();
  expect(screen.getByText(/исключают платежи на проверке/)).toBeTruthy();
  change('Сумма месяца 1', '10.01'); change('Сумма по чеку', '10.01'); change('Дата оплаты', '2026-10-01'); change('Номер платёжки / чека', 'R');
  expect((screen.getByRole('button', { name: 'Отправить на проверку' }) as HTMLButtonElement).disabled).toBe(true);
  fireEvent.click(screen.getByRole('button', { name: 'Оплатить остаток графика' }));
  expect((screen.getByLabelText('Сумма месяца 2') as HTMLInputElement).value).toBe('90071992547409931.00');
  expect(run.requests).toHaveLength(0);
});

it('keeps the completed receipt non-resubmittable when host refresh fails', async () => {
  const onSubmitted = vi.fn().mockRejectedValue(new Error('read failed'));
  const run = setup(onSubmitted);
  change('Сумма месяца 1', '10'); change('Сумма по чеку', '10'); change('Дата оплаты', '2026-10-01'); change('Номер платёжки / чека', 'R');
  fireEvent.click(screen.getByRole('button', { name: 'Отправить на проверку' }));
  await screen.findByText(/Платёж отправлен на проверку/);
  expect(screen.queryByRole('button', { name: 'Отправить на проверку' })).toBeNull();
  expect(run.requests).toHaveLength(1);
});
