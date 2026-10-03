// @vitest-environment jsdom
import { cleanup, fireEvent, render, screen, waitFor } from '@testing-library/react';
import { afterEach, expect, it, vi } from 'vitest';
import { InstallmentPaymentReview } from './installment-payment-review';
import type { InstallmentPaymentGroup } from './data';

const payment: InstallmentPaymentGroup = { id: 'receipt-1', dealId: 'deal-1', installmentPlanId: 'plan-1', claimedAmount: { amountMinor: '3000', currency: 'USD' }, paidOn: '2026-10-01', externalReference: 'CHECK-1', attachmentIds: ['file-1', 'file-2'], status: 'submitted', decisionReason: '', revision: 'parent-revision-9', allowedActions: ['accept', 'reject'], allocations: [
  { invoiceId: 'month-1', evidenceId: 'e1', number: 1, amount: { amountMinor: '1000', currency: 'USD' } },
  { invoiceId: 'month-2', evidenceId: 'e2', number: 2, amount: { amountMinor: '2000', currency: 'USD' } },
] };
const response = (body: unknown, status = 200) => new Response(JSON.stringify(body), { status, headers: { 'Content-Type': 'application/json' } });
afterEach(() => { cleanup(); vi.unstubAllGlobals(); });

it('reviews every allocation and accepts the parent with its revision', async () => {
  const requests: { url: string; body: unknown; revision: string | null }[] = [];
  vi.stubGlobal('fetch', vi.fn(async (input: RequestInfo | URL, init: RequestInit = {}) => {
    requests.push({ url: String(input), body: JSON.parse(String(init.body)), revision: new Headers(init.headers).get('If-Match') }); return response({ data: payment, revision: payment.revision });
  }));
  const onDecided = vi.fn().mockResolvedValue(undefined);
  const onBusyChange = vi.fn();
  render(<InstallmentPaymentReview payment={payment} onBack={vi.fn()} onBusyChange={onBusyChange} onDecided={onDecided} />);
  expect(screen.getByText('1')).toBeTruthy(); expect(screen.getByText('2')).toBeTruthy();
  expect(screen.queryByText('month-1')).toBeNull(); expect(screen.queryByText('month-2')).toBeNull();
  expect(screen.getAllByRole('link', { name: /файл/ })).toHaveLength(2);
  fireEvent.click(screen.getByRole('button', { name: 'Принять весь чек' }));
  expect(requests).toHaveLength(0);
  fireEvent.click(screen.getByLabelText('Деньги поступили'));
  fireEvent.click(screen.getByRole('button', { name: 'Принять весь чек' }));
  await waitFor(() => expect(onDecided).toHaveBeenCalledOnce());
  expect(onBusyChange.mock.calls).toEqual([[true], [false]]);
  expect(requests).toEqual([{ url: '/api/v1/retail/installment-payments/receipt-1/accept', body: { confirmation: true }, revision: '"parent-revision-9"' }]);
  expect(screen.getByText('Статус: Принято')).toBeTruthy();
  expect(screen.queryByText('Статус: На проверке')).toBeNull();
});

it('hides missing actions and retains successful parent decision after refresh failure', async () => {
  const requests: unknown[] = [];
  vi.stubGlobal('fetch', vi.fn(async (input: RequestInfo | URL, init: RequestInit = {}) => { requests.push(JSON.parse(String(init.body))); return response({ data: payment, revision: payment.revision }); }));
  const { rerender } = render(<InstallmentPaymentReview payment={{ ...payment, allowedActions: [] }} onBack={vi.fn()} onDecided={vi.fn()} />);
  expect(screen.queryByRole('button', { name: 'Принять весь чек' })).toBeNull();
  rerender(<InstallmentPaymentReview payment={payment} onBack={vi.fn()} onDecided={vi.fn().mockRejectedValue(new Error('refresh'))} />);
  fireEvent.click(screen.getByLabelText('Деньги поступили'));
  fireEvent.click(screen.getByRole('button', { name: 'Принять весь чек' }));
  await screen.findByText(/Решение сохранено/);
  expect(screen.queryByRole('button', { name: 'Принять весь чек' })).toBeNull();
  expect(requests).toEqual([{ confirmation: true }]);
});

it('requires a rejection reason and submits one complete parent decision', async () => {
  const fetch = vi.fn(async () => response({ data: { ...payment, status: 'rejected' }, revision: 'new' })); vi.stubGlobal('fetch', fetch);
  render(<InstallmentPaymentReview payment={payment} onBack={vi.fn()} onDecided={vi.fn().mockResolvedValue(undefined)} />);
  fireEvent.click(screen.getByRole('button', { name: 'Отклонить весь чек' })); expect(fetch).not.toHaveBeenCalled();
  fireEvent.change(screen.getByLabelText('Причина отклонения'), { target: { value: ' Нет зачисления ' } });
  fireEvent.click(screen.getByRole('button', { name: 'Отклонить весь чек' })); await screen.findByText('Статус: Отклонено: Нет зачисления');
  expect(fetch).toHaveBeenCalledOnce();
  const [url, init] = fetch.mock.calls[0] as unknown as [string, RequestInit];
  expect(url).toBe('/api/v1/retail/installment-payments/receipt-1/reject'); expect(JSON.parse(String(init.body))).toEqual({ reason: 'Нет зачисления' });
  expect(new Headers(init.headers).get('If-Match')).toBe('"parent-revision-9"');
});
