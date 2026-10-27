// @vitest-environment jsdom
import { QueryClient, QueryClientProvider } from '@tanstack/react-query';
import { cleanup, fireEvent, render, screen, waitFor } from '@testing-library/react';
import { MemoryRouter } from 'react-router-dom';
import { afterEach, describe, expect, it, vi } from 'vitest';
import { SessionGate } from '@justixauto/kit';
import { ReservationWorkspace } from './reservation-workspace';
import { BillingPage } from './pages/billing';
import { FinanceWorkspace } from '../accounting/finance-workspace';

afterEach(() => {
  cleanup();
  vi.unstubAllGlobals();
});
const m = (amountMinor: string) => ({ amountMinor, currency: 'USD' });
const response = (body: unknown, status = 200) =>
  new Response(JSON.stringify(body), { status, headers: { 'Content-Type': 'application/json' } });
const reservation = {
  id: 'res-1',
  status: 'active',
  buyer: { id: 'dealer', name: 'Dealer' },
  supplier: { id: 'distributor', name: 'Distributor' },
  requested: { terms: {}, lines: [] },
  current: { terms: { unitPrice: m('10000'), advance: m('5000'), advanceDueAt: '2030-07-01T00:00:00Z' }, lines: [] },
  requestedDigest: 'requested',
  currentDigest: 'agreed',
  revision: '9007199254740993',
  advanceDueAt: '2030-07-01T00:00:00Z',
  allowedActions: [],
};
const evidence = (status: 'submitted' | 'accepted' | 'rejected' = 'submitted') => ({
  id: 'e-1',
  amount: m('5000'),
  paidOn: '2026-10-01',
  externalReference: 'BANK-1',
  attachmentIds: ['file-1'],
  status,
  decisionReason: status === 'rejected' ? 'Не поступили' : '',
  submittedBy: 'buyer-user',
  decidedBy: status === 'submitted' ? undefined : 'accountant-user',
  revision: '9007199254740994',
  createdAt: '2026-10-01T10:00:00Z',
  decidedAt: status === 'submitted' ? undefined : '2026-10-02T10:00:00Z',
});
const accounting = (extra: Record<string, unknown> = {}) => ({
  reservationId: 'res-1',
  agreedAdvance: m('5000'),
  accepted: m('0'),
  pending: m('0'),
  refunded: m('0'),
  refundObligation: m('0'),
  refundStatus: 'none',
  evidence: [],
  allowedActions: ['submit-payment'],
  revision: '9007199254740993',
  ...extra,
});
const session = (company: 'dealer' | 'distributor', permissions: string[]) => ({
  data: {
    user: { id: 'user', displayName: 'User', status: 'active', passwordChangeRequired: false },
    roles: [],
    permissions,
    context: { revision: '1', companyId: company, branchScope: { mode: 'ALL', branchIds: [] } },
    accessibleCompanies: [
      {
        id: company,
        name: company,
        kind: 'seller',
        userType: company,
        verification: 'verified',
        verificationReason: '',
        access: 'full',
      },
    ],
    setup: { next: 'none' },
  },
});
type Call = { url: string; method: string; headers: Headers; body: unknown };
function harness(
  company: 'dealer' | 'distributor',
  permissions: string[],
  initial: ReturnType<typeof accounting>,
  host: 'workspace' | 'billing' = 'workspace',
  onPost?: (call: Call, state: { value: ReturnType<typeof accounting>; stale: boolean }) => Response | void,
  withInvoice = false,
  accountingScope = false,
) {
  const state = { value: initial, stale: false, calls: [] as Call[], reads: 0 };
  vi.stubGlobal(
    'fetch',
    vi.fn(async (input: RequestInfo | URL, init: RequestInit = {}) => {
      const url = String(input);
      const method = init.method ?? 'GET';
      const headers = new Headers(init.headers);
      const body = init.body && typeof init.body === 'string' ? JSON.parse(init.body) : init.body;
      const call = { url, method, headers, body };
      state.calls.push(call);
      if (url.endsWith('/identity/session')) return response(session(company, permissions));
      if (url.endsWith('/documents/files') && method === 'POST')
        return response(
          { data: { id: 'file-1', fileName: 'proof.pdf', mime: 'application/pdf', byteLength: 1, sensitive: true } },
          201,
        );
      if (method === 'POST') {
        const result = onPost?.(call, state);
        return result ?? response({ data: state.value, revision: state.value.revision });
      }
      if (url.endsWith('/commerce/reservations/res-1/accounting')) {
        state.reads++;
        return response({ data: state.value, revision: state.value.revision });
      }
      if (url.endsWith('/commerce/reservations/res-1'))
        return response({ data: reservation, revision: reservation.revision });
      if (url.includes('/commerce/reservations?')) return response({ items: [reservation] });
      if (url.endsWith('/commerce/orders/order-1/invoices') && withInvoice)
        return response({
          items: [
            {
              id: 'invoice-1',
              orderId: 'order-1',
              total: m('10000'),
              paid: m('2000'),
              pending: m('0'),
              outstanding: m('8000'),
              status: 'issued',
              paymentEvidence: [],
              allowedActions: [],
              revision: '2',
              schedule: [],
            },
          ],
        });
      if (url.includes('/commerce/orders?'))
        return response({
          items: withInvoice
            ? [
                {
                  id: 'order-1',
                  party: 'buyer',
                  buyer: { name: 'Dealer' },
                  supplier: { name: 'Distributor' },
                  source: 'reservation',
                  terms: { lines: [], paymentSchedule: [] },
                  total: m('10000'),
                  status: 'accepted',
                  revision: '2',
                },
              ]
            : [],
        });
      if (url.includes('/commerce/supply-lots') || url.includes('/inventory/vehicle-models'))
        return response({ items: [] });
      return response({ items: [] });
    }),
  );
  const client = new QueryClient({ defaultOptions: { queries: { retry: false, staleTime: Infinity } } });
  const page = host === 'billing' ? <BillingPage /> : <ReservationWorkspace onClose={() => {}} />;
  render(
    <QueryClientProvider client={client}>
      <MemoryRouter>
        <SessionGate title="Test">
          {host === 'billing' || accountingScope ? <FinanceWorkspace>{page}</FinanceWorkspace> : page}
        </SessionGate>
      </MemoryRouter>
    </QueryClientProvider>,
  );
  return state;
}
async function workspaceEntry() {
  fireEvent.click(await screen.findByRole('button', { name: 'Открыть' }));
  await screen.findByRole('heading', { name: 'Аванс и возврат' });
}
const postCall = (calls: Call[], suffix: string) => calls.find((c) => c.method === 'POST' && c.url.endsWith(suffix));

describe('P19b reservation accounting', () => {
  it('submits exact private evidence from the real workspace, shows pending and blocks a duplicate', async () => {
    const state = harness('dealer', ['commerce.read', 'commerce.trade'], accounting(), 'workspace', (call) => {
      if (call.url.endsWith('/payment-evidence'))
        state.value = accounting({
          pending: m('5000'),
          evidence: [evidence()],
          allowedActions: ['request-refund'],
          revision: '9007199254740994',
        });
    });
    await workspaceEntry();
    fireEvent.click(screen.getByRole('button', { name: 'Сообщить об авансе' }));
    fireEvent.change(screen.getByLabelText('Сумма аванса'), { target: { value: '50.00' } });
    fireEvent.change(screen.getByLabelText('Дата внешнего платежа'), { target: { value: '2026-10-01' } });
    fireEvent.change(screen.getByLabelText('Номер платёжного документа'), { target: { value: ' BANK-1 ' } });
    fireEvent.change(screen.getByLabelText('Подтверждение'), {
      target: { files: [new File(['x'], 'proof.pdf', { type: 'application/pdf' })] },
    });
    const button = screen.getByRole('button', { name: 'Подтвердить действие' });
    fireEvent.click(button);
    fireEvent.click(button);
    await screen.findByText(/Отправленный документ ожидает проверки/);
    const call = postCall(state.calls, '/commerce/reservations/res-1/payment-evidence')!;
    expect(call.body).toEqual({
      claimedAmount: m('5000'),
      paidOn: '2026-10-01',
      externalReference: 'BANK-1',
      attachmentBindingIds: ['file-1'],
    });
    expect(call.headers.get('If-Match')).toBe('"9007199254740993"');
    expect(state.calls.filter((c) => c.url.endsWith('/payment-evidence') && c.method === 'POST')).toHaveLength(1);
    expect(screen.getByText(/BANK-1/)).toBeTruthy();
    expect(screen.getByRole('link', { name: 'Приватный файл' }).getAttribute('href')).toBe(
      '/api/v1/documents/files/file-1/content',
    );
    expect(screen.getByText(/Принято: 0.00 USD/)).toBeTruthy();
    expect(state.reads).toBeGreaterThan(1);
  });

  it('lets only the supplier accountant accept or reject with both revisions and a recorded reason', async () => {
    const initial = accounting({
      pending: m('5000'),
      evidence: [evidence()],
      allowedActions: ['accept-payment', 'reject-payment'],
    });
    const state = harness(
      'distributor',
      ['commerce.read', 'commerce.payments.accept'],
      initial,
      'workspace',
      (call) => {
        if (call.url.endsWith('/accept'))
          state.value = accounting({
            accepted: m('5000'),
            evidence: [evidence('accepted')],
            allowedActions: [],
            revision: '9007199254740994',
          });
      },
      false,
      true,
    );
    await workspaceEntry();
    fireEvent.click(screen.getByRole('button', { name: 'Принять аванс' }));
    fireEvent.click(screen.getByLabelText('Деньги поступили на счёт поставщика'));
    fireEvent.click(screen.getByRole('button', { name: 'Подтвердить действие' }));
    await screen.findByText(/Принято: 50.00 USD/);
    const call = postCall(state.calls, '/reservation-payment-evidence/e-1/accept')!;
    expect(call.body).toEqual({ reservationRevision: '9007199254740993', confirmation: true, reason: '' });
    expect(call.headers.get('If-Match')).toBe('"9007199254740994"');
    expect(screen.queryByRole('button', { name: 'Принять аванс' })).toBeNull();
    cleanup();

    const denied = harness('distributor', ['commerce.read'], initial);
    await workspaceEntry();
    expect(screen.queryByRole('button', { name: 'Принять аванс' })).toBeNull();
    expect(denied.calls.some((c) => c.method === 'POST')).toBe(false);
    cleanup();

    const outsideAccounting = harness('distributor', ['commerce.read', 'commerce.payments.accept'], initial);
    await workspaceEntry();
    expect(screen.queryByRole('button', { name: 'Принять аванс' })).toBeNull();
    expect(screen.queryByRole('button', { name: 'Отклонить аванс' })).toBeNull();
    expect(outsideAccounting.calls.some((c) => c.method === 'POST')).toBe(false);
    cleanup();

    const rejected = harness(
      'distributor',
      ['commerce.read', 'commerce.payments.accept'],
      initial,
      'workspace',
      (call, current) => {
        if (call.url.endsWith('/reject'))
          current.value = accounting({
            evidence: [evidence('rejected')],
            allowedActions: [],
            revision: '9007199254740994',
          });
      },
      false,
      true,
    );
    await workspaceEntry();
    fireEvent.click(screen.getByRole('button', { name: 'Отклонить аванс' }));
    fireEvent.change(screen.getByLabelText('Причина'), { target: { value: 'Не поступили' } });
    fireEvent.click(screen.getByRole('button', { name: 'Подтвердить действие' }));
    await waitFor(() => expect(postCall(rejected.calls, '/reservation-payment-evidence/e-1/reject')).toBeTruthy());
    expect(postCall(rejected.calls, '/reservation-payment-evidence/e-1/reject')!.body).toEqual({
      reservationRevision: '9007199254740993',
      confirmation: false,
      reason: 'Не поступили',
    });
  });

  it('distinguishes refund request from accountant confirmed full external refund', async () => {
    const state = harness(
      'dealer',
      ['commerce.read', 'commerce.trade'],
      accounting({ accepted: m('5000'), evidence: [evidence('accepted')], allowedActions: ['request-refund'] }),
      'workspace',
      (call) => {
        if (call.url.endsWith('/refund-requests'))
          state.value = accounting({
            accepted: m('5000'),
            evidence: [evidence('accepted')],
            refundStatus: 'requested',
            refundObligation: m('5000'),
            allowedActions: [],
            revision: '9007199254740994',
          });
      },
    );
    await workspaceEntry();
    fireEvent.click(screen.getByRole('button', { name: 'Запросить возврат' }));
    fireEvent.change(screen.getByLabelText('Причина'), { target: { value: 'Отмена' } });
    fireEvent.click(screen.getByRole('button', { name: 'Подтвердить действие' }));
    await screen.findByText(/Возврат запрошен/);
    expect(screen.getByText(/подтверждённый возврат: 0.00 USD/)).toBeTruthy();
    expect(postCall(state.calls, '/refund-requests')!.body).toEqual({ reason: 'Отмена' });
    cleanup();

    const ready = accounting({
      accepted: m('5000'),
      evidence: [evidence('accepted')],
      refundStatus: 'requested',
      refundObligation: m('5000'),
      allowedActions: ['confirm-refund'],
    });
    const confirmed = harness(
      'distributor',
      ['commerce.read', 'commerce.payments.accept'],
      ready,
      'billing',
      (call) => {
        if (call.url.endsWith('/refund-confirmations'))
          confirmed.value = accounting({
            accepted: m('5000'),
            evidence: [evidence('accepted')],
            refundStatus: 'confirmed',
            refunded: m('5000'),
            refundObligation: m('5000'),
            allowedActions: [],
            revision: '9007199254740994',
          });
      },
    );
    fireEvent.click(await screen.findByRole('button', { name: 'Открыть расчёты' }));
    fireEvent.click(await screen.findByRole('button', { name: 'Подтвердить полный возврат' }));
    fireEvent.change(screen.getByLabelText('Дата внешнего платежа'), { target: { value: '2026-10-01' } });
    fireEvent.change(screen.getByLabelText('Номер платёжного документа'), { target: { value: 'REFUND-1' } });
    fireEvent.click(screen.getByLabelText('Полный внешний возврат выполнен'));
    fireEvent.click(screen.getByRole('button', { name: 'Подтвердить действие' }));
    await screen.findByText(/Полный возврат подтверждён бухгалтером/);
    expect(postCall(confirmed.calls, '/refund-confirmations')!.body).toEqual({
      claimedAmount: m('5000'),
      paidOn: '2026-10-01',
      externalReference: 'REFUND-1',
      attachmentBindingIds: [],
      confirmation: true,
    });
    expect(postCall(confirmed.calls, '/refund-confirmations')!.headers.get('If-Match')).toBe('"9007199254740993"');
  });

  it('renders fetched evidence actors and refund provenance without inventing confirmation', async () => {
    const requested = accounting({
      accepted: m('5000'),
      evidence: [evidence('accepted')],
      refundStatus: 'requested',
      refundObligation: m('5000'),
      refund: {
        status: 'requested',
        requestedBy: 'dealer-user',
        requestedAt: '2026-10-03T09:00:00Z',
        requestReason: 'Отмена заказа',
      },
      allowedActions: [],
    });
    harness('dealer', ['commerce.read', 'commerce.trade'], requested);
    await workspaceEntry();
    expect(await screen.findByText(/отправил: buyer-user/)).toBeTruthy();
    expect(screen.getByText(/решение: accountant-user/)).toBeTruthy();
    expect(screen.getByText(/Запросил: dealer-user/)).toBeTruthy();
    expect(screen.getByText(/причина: Отмена заказа/)).toBeTruthy();
    expect(screen.queryByText(/Фактический возврат:/)).toBeNull();
    cleanup();

    const confirmed = accounting({
      accepted: m('5000'),
      evidence: [evidence('accepted')],
      refunded: m('5000'),
      refundObligation: m('5000'),
      refundStatus: 'confirmed',
      refund: {
        status: 'confirmed',
        requestedBy: 'dealer-user',
        requestedAt: '2026-10-03T09:00:00Z',
        requestReason: 'Отмена заказа',
        amount: m('5000'),
        externalReference: 'REFUND-1',
        attachmentIds: ['refund-file'],
        refundedOn: '2026-10-04',
        confirmedBy: 'accountant-user',
        confirmedAt: '2026-10-04T10:00:00Z',
      },
      allowedActions: [],
    });
    harness('distributor', ['commerce.read', 'commerce.payments.accept'], confirmed);
    await workspaceEntry();
    expect(await screen.findByText(/Фактический возврат: 50.00 USD/)).toBeTruthy();
    expect(screen.getByText(/документ REFUND-1/)).toBeTruthy();
    expect(screen.getByText(/подтвердил: accountant-user/)).toBeTruthy();
    expect(screen.getByLabelText('История возврата').querySelector('a')?.getAttribute('href')).toBe(
      '/api/v1/documents/files/refund-file/content',
    );
  });

  it('opens actual Billing reservation and invoice surfaces without double counting accepted credit', async () => {
    const initial = accounting({
      accepted: m('2000'),
      evidence: [{ ...evidence('accepted'), amount: m('2000') }],
      allowedActions: [],
      creditOrderId: 'order-1',
      creditInvoiceId: 'invoice-1',
    });
    const state = harness(
      'distributor',
      ['commerce.read', 'commerce.payments.accept'],
      initial,
      'billing',
      undefined,
      true,
    );
    expect(await screen.findByText(/Аванс зачтён в счёт/)).toBeTruthy();
    fireEvent.click(screen.getByRole('button', { name: 'Открыть расчёты' }));
    expect(await screen.findByText(/Оплата счёта уже включает этот зачёт/)).toBeTruthy();
    expect(screen.getByText(/Принято: 20.00 USD/)).toBeTruthy();
    fireEvent.click(screen.getByRole('button', { name: 'Закрыть' }));
    fireEvent.click(await screen.findByRole('button', { name: 'Добавить оплату' }));
    expect(await screen.findByText(/остаток 80.00 USD/)).toBeTruthy();
    expect(screen.getAllByText('20.00 USD').length).toBeGreaterThan(0);
    expect(state.calls.some((c) => c.url.endsWith('/commerce/reservations/res-1/accounting'))).toBe(true);
    expect(state.calls.some((c) => c.url.endsWith('/commerce/orders/order-1/invoices'))).toBe(true);
  });

  it('shows a stale review error, reloads authority and allows a safe retry only with the new revision', async () => {
    let attempt = 0;
    const state = harness(
      'distributor',
      ['commerce.read', 'commerce.payments.accept'],
      accounting({ pending: m('5000'), evidence: [evidence()], allowedActions: ['accept-payment', 'reject-payment'] }),
      'workspace',
      (call) => {
        if (call.url.endsWith('/accept') && ++attempt === 1) {
          state.value = accounting({
            pending: m('5000'),
            evidence: [{ ...evidence(), revision: '9007199254740995' }],
            allowedActions: ['accept-payment', 'reject-payment'],
            revision: '9007199254740995',
          });
          return response({ error: { code: 'stale', message: 'Stale' } }, 409);
        }
        if (call.url.endsWith('/accept'))
          state.value = accounting({
            accepted: m('5000'),
            evidence: [evidence('accepted')],
            allowedActions: [],
            revision: '9007199254740996',
          });
      },
      false,
      true,
    );
    await workspaceEntry();
    fireEvent.click(screen.getByRole('button', { name: 'Принять аванс' }));
    fireEvent.click(screen.getByLabelText('Деньги поступили на счёт поставщика'));
    fireEvent.click(screen.getByRole('button', { name: 'Подтвердить действие' }));
    await screen.findByText(/Stale|устарел/i);
    fireEvent.click(screen.getByRole('button', { name: 'Подтвердить действие' }));
    await screen.findByText(/Принято: 50.00 USD/);
    const calls = state.calls.filter((c) => c.url.endsWith('/accept'));
    expect(calls).toHaveLength(2);
    expect(calls[0]!.headers.get('If-Match')).toBe('"9007199254740994"');
    expect(calls[1]!.headers.get('If-Match')).toBe('"9007199254740995"');
    expect(calls[1]!.body).toMatchObject({ reservationRevision: '9007199254740995' });
  });
});
