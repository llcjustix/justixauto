// @vitest-environment jsdom
import { QueryClient, QueryClientProvider } from '@tanstack/react-query';
import { MemoryRouter } from 'react-router-dom';
import { SessionGate } from '@justixauto/kit';
import { act, cleanup, fireEvent, render, screen, waitFor, within } from '@testing-library/react';
import { afterEach, describe, expect, it, vi } from 'vitest';
import { DealDialog } from './pages/retail';
import { SalesPage } from './pages/sales';
import { BillingPage } from './pages/billing';
import { calculateInstallment } from './installment-calculator';
import type { Deal, InstallmentPlan, Money, RetailInvoice } from './data';

const m = (amountMinor: string): Money => ({ amountMinor, currency: 'USD' });
function invoice(id: string, amount: string, purpose = 'monthly-installment', number: number | null = 1): RetailInvoice {
  return { id, dealId: 'd-1', purpose, installmentNumber: number, amount: m(amount), recipientSnapshot: 'Клиент',
    dueDate: '2024-02-29', status: 'issued', paid: m('0'), pending: m('0'), outstanding: m(amount), available: m(amount),
    allowedActions: ['submit-payment'], paymentEvidence: [], revision: 'invoice-rev' };
}
function deal(saved = true): Deal {
  const initial = invoice('initial', '1000', 'first-installment', null);
  initial.paid = m('1000'); initial.outstanding = m('0'); initial.available = m('0'); initial.allowedActions = [];
  initial.paymentEvidence = [{ id: 'deposit-evidence', amount: m('1000'), paidOn: '2024-01-01', externalReference: 'INITIAL-PAID', attachmentIds: [], status: 'accepted', decisionReason: '', revision: 'deposit-rev', allowedActions: [] }];
  const registration = invoice('registration', '500', 'registration', null);
  registration.allowedActions = [];
  const invoices = saved ? [initial, registration, invoice('monthly-1', '10000'), invoice('monthly-2', '19000', 'monthly-installment', 2)] : [initial, registration];
  const plan: InstallmentPlan = { id: 'plan-1', state: 'active', contractReference: 'SIGNED-2024', contractSignedOn: '2024-01-01',
    contractFileIds: ['saved-contract'], contractTotal: m('30000'), downPaymentInvoiceId: initial.id, downPayment: initial.amount,
    scheduledTotal: m('29000'), paid: m('0'), pending: m('0'), outstanding: m('29000'), available: m('29000'), settlementState: 'outstanding', allowedActions: ['submit-installment-payment'], payments: [],
    rows: invoices.filter(i => i.purpose === 'monthly-installment').map(i => ({ number: i.installmentNumber!, invoiceId: i.id, dueDate: i.dueDate!, amount: i.amount, paid: i.paid, pending: i.pending, outstanding: i.outstanding, available: i.available, allowedActions: i.allowedActions })),
  };
  return { id: 'd-1', branchId: 'branch', customer: { id: 'customer', displayName: 'Тестовый покупатель', phone: '+998', revision: '1' },
    leadId: null, vehicleId: 'car', paymentScheme: 'own-installment', price: m('30000'), status: 'delivered', statusReason: '',
    contractSignedOn: '2026-09-30', contractReference: 'CURRENT-2026', contractFileIds: ['current-contract'], registeredOn: null,
    plateNumber: '', registrationReference: '', deliveredAt: '2024-01-02T00:00:00Z', invoices, installmentPlan: saved ? plan : null,
    installmentDraft: null, allowedActions: saved ? [] : ['set-installment-terms'], history: [], revision: 'deal-rev-7', updatedAt: '2026-10-01T00:00:00Z' };
}
function evidence(status = 'submitted', allowedActions = ['accept', 'reject']) {
  return { id: 'e-2', amount: m('1025'), paidOn: '2024-02-28', externalReference: 'PAY-2', attachmentIds: ['proof'], status,
    decisionReason: '', revision: 'evidence-rev-3', allowedActions };
}
const response = (body: unknown, status = 200) => new Response(JSON.stringify(body), { status, headers: { 'Content-Type': 'application/json' } });
const failure = (status = 412, message = 'stale', fields = {}) => response({ error: { code: 'fixture_error', message, fields } }, status);
type Recorded = { url: string; body: Record<string, any>; revision: string | null };
function harness(initial = deal(), page: 'deal' | 'sales' | 'billing' = 'deal', initialInvoiceId?: string) {
  const state = { deal: initial, failGet: false, failBilling: false, billingReads: 0, reads: 0, requests: [] as Recorded[], urls: [] as string[],
    onPost: undefined as ((request: Recorded) => Response | Promise<Response>) | undefined,
    uploadGate: undefined as Promise<void> | undefined,
    readGate: undefined as Promise<void> | undefined };
  const fetch = vi.fn(async (input: RequestInfo | URL, init: RequestInit = {}) => {
    const url = String(input); state.urls.push(url);
    if (init.method === 'POST') {
      if (url.endsWith('/documents/files')) { if (state.uploadGate) await state.uploadGate; return response({ data: { id: 'uploaded-proof' }, revision: '1' }, 201); }
      const request = { url, body: JSON.parse(String(init.body)), revision: new Headers(init.headers).get('If-Match') };
      state.requests.push(request);
      return state.onPost ? state.onPost(request) : response({ data: state.deal, revision: state.deal.revision }, 201);
    }
    if (url.endsWith('/identity/session')) return response({ data: {
      user: { id: 'u', displayName: 'User', status: 'active', passwordChangeRequired: false }, roles: [], permissions: [],
      context: { revision: '1', companyId: 'company', branchScope: { mode: 'ALL', branchIds: [] } },
      accessibleCompanies: [{ id: 'company', name: 'Company', kind: 'seller', access: 'full' }], setup: { next: 'none' },
    } });
    if (url.endsWith('/retail/deals/d-1')) {
      state.reads++;
      if (state.readGate) await state.readGate;
      return state.failGet ? failure(503, 'Сервер временно недоступен') : response({ data: state.deal, revision: state.deal.revision });
    }
    if (url.includes('/retail/deals?')) { state.billingReads++; if (state.failBilling) return failure(503, 'Billing read failed'); }
    return response({ items: url.includes('/retail/deals?') ? [state.deal] : [] });
  });
  vi.stubGlobal('fetch', fetch);
  const client = new QueryClient({ defaultOptions: { queries: { retry: false, staleTime: Infinity }, mutations: { retry: false } } });
  const onClose = vi.fn();
  const scene = page === 'sales' ? <SalesPage /> : page === 'billing' ? <BillingPage /> : <DealDialog id="d-1" {...(initialInvoiceId ? { initialInvoiceId } : {})} onClose={onClose} />;
  const view = render(<QueryClientProvider client={client}><MemoryRouter><SessionGate title="Test">{scene}</SessionGate></MemoryRouter></QueryClientProvider>);
  return { state, client, view, fetch, onClose };
}
const change = (label: string | RegExp, value: string) => fireEvent.change(screen.getByLabelText(label), { target: { value } });
const click = (name: string | RegExp) => fireEvent.click(screen.getByRole('button', { name }));
const closeFromFooter = () => fireEvent.click(within(screen.getByRole('dialog').querySelector('.modal-footer')!).getByRole('button', { name: 'Закрыть' }));
const oneDialog = () => expect(screen.getAllByRole('dialog')).toHaveLength(1);
const selected = () => within(screen.getByRole('region', { name: 'Выбранный платёж рассрочки' }));
async function showPayments() {
  fireEvent.click(await screen.findByRole('tab', { name: 'Оплаты', exact: true }));
}
async function editSchedule() {
  await showPayments();
  fireEvent.click(await screen.findByRole('button', { name: 'Рассчитать рассрочку' }));
  oneDialog();
  change('Срок, месяцев', '3'); change('Дата первого платежа', '2024-01-31');
}
async function openPayment(number = 2) {
  await showPayments();
  fireEvent.click(await screen.findByRole('button', { name: `Открыть платёж ${number}` }));
  fireEvent.click(selected().getByRole('button', { name: 'Внести оплату' }));
  oneDialog();
  change(`Сумма месяца ${number}`, '10.25'); change('Сумма по чеку', '10.25'); change(/Дата оплаты/, '2024-02-28'); change(/Номер платёжки/, 'PAY-2');
}
function updateInvoice(d: Deal, id: string, patch: Partial<RetailInvoice>) {
  d.invoices = d.invoices!.map(i => i.id === id ? { ...i, ...patch } : i);
  if (d.installmentPlan) d.installmentPlan.rows = d.installmentPlan.rows.map(row => {
    const i = d.invoices!.find(i => i.id === row.invoiceId)!;
    return { ...row, paid: i.paid, pending: i.pending, outstanding: i.outstanding, available: i.available, allowedActions: i.allowedActions };
  });
}
afterEach(() => { cleanup(); vi.unstubAllGlobals(); });

describe('sale dialog tabs', () => {
  it('defaults to deal readiness and exposes only the selected section with common actions and close', async () => {
    const d = deal(); d.status = 'reserved'; d.allowedActions = ['cancel'];
    d.checklist = { contract: true, firstInstallment: true, registrationPaid: false, registered: false, policyResolved: true };
    d.history = [{ type: 'deal.reserved', occurredAt: '2024-01-01T00:00:00Z', reason: 'История тестовой продажи' }];
    const { onClose } = harness(d);
    await screen.findByRole('tabpanel', { name: 'Сделка и выдача' });
    expect(screen.getByRole('tab', { name: 'Сделка и выдача' }).getAttribute('aria-selected')).toBe('true');
    expect(screen.getByLabelText('Готовность к выдаче')).toBeTruthy();
    expect(screen.getByRole('link', { name: 'скан 1' }).getAttribute('href')).toBe('/api/v1/documents/files/current-contract/content');
    expect(screen.queryByRole('button', { name: 'Открыть платёж 1' })).toBeNull();
    expect(screen.queryByText(/История тестовой продажи/)).toBeNull();
    for (const name of ['Оплаты', 'История', 'Сделка и выдача']) {
      fireEvent.click(screen.getByRole('tab', { name, exact: true }));
      expect(screen.getAllByRole('tabpanel')).toHaveLength(1);
      expect(screen.getByRole('tabpanel', { name })).toBeTruthy();
      expect(screen.getByRole('button', { name: 'Отменить сделку' }).closest('.modal-footer')).toBeTruthy();
      expect(screen.getAllByRole('button', { name: 'Отменить сделку' })).toHaveLength(1);
      expect(screen.getByRole('button', { name: 'Отменить сделку' }).closest('[role="tabpanel"]')).toBeNull();
      expect(!!screen.queryByRole('button', { name: 'Открыть платёж 1' })).toBe(name === 'Оплаты');
      expect(!!screen.queryByText('INITIAL-PAID')).toBe(name === 'Оплаты');
      expect(!!screen.queryByText(/История тестовой продажи/)).toBe(name === 'История');
      expect(!!screen.queryByLabelText('Готовность к выдаче')).toBe(name === 'Сделка и выдача');
      closeFromFooter(); oneDialog();
    }
    expect(onClose).toHaveBeenCalledTimes(3);
  });

  it('keeps history selected through refresh and sale action cancellation with honest empty states', async () => {
    const d = deal(false); d.paymentScheme = 'cash'; d.invoices = []; d.allowedActions = ['cancel'];
    const { state } = harness(d);
    await showPayments(); expect(screen.getByText('Счета пока не выставлены.')).toBeTruthy();
    expect(screen.queryByText('Условия беспроцентной рассрочки')).toBeNull();
    fireEvent.click(screen.getByRole('tab', { name: 'История', exact: true }));
    expect(screen.getByText('Событий пока нет.')).toBeTruthy();
    state.deal = { ...d, revision: 'refreshed' }; click('Обновить данные сделки');
    await screen.findByRole('tabpanel', { name: 'История' });
    click('Отменить сделку'); oneDialog(); change('Причина', 'Черновик'); click('Назад к сделке');
    expect(screen.getByRole('tab', { name: 'История', exact: true }).getAttribute('aria-selected')).toBe('true');
    expect(screen.getByText('Событий пока нет.')).toBeTruthy();
    expect(state.requests).toHaveLength(0); oneDialog();
  });

  it('retains payments after terms cancellation and protects the editor footer while saving', async () => {
    const { state, onClose } = harness(deal(false));
    await editSchedule(); click('Отмена');
    expect(screen.getByRole('tabpanel', { name: 'Оплаты' })).toBeTruthy();
    click('Рассчитать рассрочку'); closeFromFooter();
    expect(screen.getByRole('tabpanel', { name: 'Оплаты' })).toBeTruthy();
    await editSchedule(); let finish!: () => void;
    state.onPost = () => new Promise(resolve => { finish = () => { state.deal = deal(); resolve(response({ data: state.deal })); }; });
    click('Сохранить условия'); closeFromFooter();
    expect(screen.getByLabelText('Срок, месяцев')).toBeTruthy(); expect(onClose).not.toHaveBeenCalled();
    await act(async () => { finish(); }); await screen.findByText('Условия сохранены.');
    expect(screen.getByRole('tabpanel', { name: 'Оплаты' })).toBeTruthy(); oneDialog();
  });

  it('opens Billing payment immediately and explicit back selects payments before closing to Billing', async () => {
    harness(deal(), 'billing'); fireEvent.click(await screen.findByRole('tab', { name: 'От клиентов' }));
    fireEvent.click(within((await screen.findByText('Платёж рассрочки 2')).closest('tr')!).getByRole('button', { name: 'Открыть' }));
    await screen.findByRole('button', { name: 'Внести оплату' });
    expect(screen.queryByRole('tabpanel', { name: 'Сделка и выдача' })).toBeNull(); oneDialog();
    fireEvent.click(screen.getByRole('dialog').querySelector<HTMLButtonElement>('.modal-close')!);
    expect(screen.queryByRole('dialog')).toBeNull();
    fireEvent.click(within(screen.getByText('Платёж рассрочки 2').closest('tr')!).getByRole('button', { name: 'Открыть' }));
    await screen.findByRole('button', { name: 'Внести оплату' }); click('Назад к графику');
    expect(screen.getByRole('tabpanel', { name: 'Оплаты' })).toBeTruthy(); oneDialog();
    closeFromFooter(); expect(screen.queryByRole('dialog')).toBeNull();
    expect(screen.getByRole('tab', { name: 'От клиентов' }).getAttribute('aria-selected')).toBe('true');
  });

  it('selects readiness before focus and clears intent when returning to payments', async () => {
    const d = deal(); d.status = 'reserved'; d.installmentPlan!.state = 'planned';
    d.checklist = { contract: true, firstInstallment: false, registrationPaid: false, registered: false, policyResolved: true };
    harness(d); await showPayments(); click('Открыть платёж 2'); click('К готовности к выдаче');
    const readiness = await screen.findByLabelText('Готовность к выдаче');
    await waitFor(() => expect(document.activeElement).toBe(readiness));
    expect(screen.getByRole('tabpanel', { name: 'Сделка и выдача' })).toBeTruthy();
    await showPayments();
    const trigger = screen.getByRole('button', { name: 'Открыть платёж 1' });
    trigger.closest<HTMLElement>('.modal-body')!.scrollTop = 175;
    fireEvent.click(trigger); closeFromFooter();
    await waitFor(() => expect(document.activeElement).toBe(screen.getByRole('button', { name: 'Открыть платёж 1' })));
    expect(screen.getByRole('tabpanel', { name: 'Оплаты' })).toBeTruthy();
    expect(screen.getByRole('button', { name: 'Открыть платёж 1' }).closest<HTMLElement>('.modal-body')!.scrollTop).toBe(175);
  });

  it('restores payment tab scroll and plan focus only after failed-refresh retry remounts safe content', async () => {
    const d = deal(); d.allowedActions = ['cancel'];
    const { state } = harness(d); await showPayments();
    const trigger = screen.getByRole('button', { name: 'Внести оплату' });
    trigger.closest<HTMLElement>('.modal-body')!.scrollTop = 320; fireEvent.click(trigger);
    fireEvent.click(screen.getByLabelText('Выбрать месяц 1')); change('Сумма месяца 1', '10'); change('Сумма по чеку', '10');
    change('Дата оплаты', '2024-02-28'); change('Номер платёжки / чека', 'REFRESH-RETURN');
    state.onPost = () => { state.failGet = true; return response({ data: {} }, 201); };
    click('Отправить на проверку'); await screen.findByText(/Операция сохранена. Не удалось обновить данные/);
    closeFromFooter(); await screen.findByText(/Не удалось загрузить данные сделки/);
    expect(screen.queryByRole('button', { name: 'Отменить сделку' })).toBeNull();
    expect(screen.queryByRole('tabpanel')).toBeNull();
    state.failGet = false; click('Повторить загрузку сделки');
    const returned = await screen.findByRole('button', { name: 'Внести оплату' });
    await waitFor(() => expect(document.activeElement).toBe(returned));
    expect(returned.closest<HTMLElement>('.modal-body')!.scrollTop).toBe(320);
    expect(screen.getByRole('tabpanel', { name: 'Оплаты' })).toBeTruthy();
    expect(state.requests).toHaveLength(1); oneDialog();
  });
});

describe('scheduled remaining balance', () => {
  function savedSchedule(d: Deal, total: string, entries: [number, string][]) {
    const monthly = entries.map(([number, amount]) => invoice(`monthly-${number}`, amount, 'monthly-installment', number));
    d.invoices = [...d.invoices!.filter(i => i.purpose !== 'monthly-installment'), ...monthly];
    d.installmentPlan!.scheduledTotal = m(total);
    d.installmentPlan!.rows = monthly.map(i => ({ number: i.installmentNumber!, invoiceId: i.id, dueDate: i.dueDate!, amount: i.amount,
      paid: i.paid, pending: i.pending, outstanding: i.outstanding, available: i.available, allowedActions: i.allowedActions }));
  }
  const scheduleRow = (number: number) => screen.getByRole('button', { name: `Открыть платёж ${number}` }).closest('tr')!;

  it('shows the planned balance after each full saved payment including the final rounding remainder', async () => {
    const d = deal();
    savedSchedule(d, '2200000', Array.from({ length: 36 }, (_, index): [number, string] => [index + 1, index === 35 ? '61115' : '61111']));
    harness(d); await showPayments();
    await screen.findByText('Остаток после платежа');
    expect(screen.getByText(/плановый остаток после платежей по сохранённому графику; фактические оплаты показаны отдельно/)).toBeTruthy();
    expect(scheduleRow(1).textContent).toContain('21 388.89 USD');
    expect(scheduleRow(35).textContent).toContain('611.15 USD');
    expect(scheduleRow(36).textContent).toContain('0.00 USD');
  });

  it('uses immutable saved rows in schedule-number order instead of current deal values', async () => {
    const d = deal();
    savedSchedule(d, '1000', [[3, '150'], [1, '400'], [2, '450']]);
    d.price = m('999999');
    d.installmentDraft = calculateInstallment(d.price, { downPayment: m('1'), termMonths: 2, firstDueDate: '2024-01-31' });
    harness(d); await showPayments();
    await screen.findByRole('button', { name: 'Открыть платёж 1' });
    expect(scheduleRow(1).textContent).toContain('6.00 USD');
    expect(scheduleRow(2).textContent).toContain('1.50 USD');
    expect(scheduleRow(3).textContent).toContain('0.00 USD');
    expect(d.installmentPlan!.rows.map(row => row.number)).toEqual([3, 1, 2]);
  });

  it('keeps planned projection separate from actual facts and opens the actual invoice balance', async () => {
    const d = deal();
    savedSchedule(d, '30000', [[1, '10000'], [2, '20000']]);
    updateInvoice(d, 'monthly-1', { paid: m('10000'), outstanding: m('0'), available: m('0') });
    updateInvoice(d, 'monthly-2', { pending: m('20000'), available: m('0') });
    Object.assign(d.installmentPlan!, { paid: m('10000'), pending: m('20000'), outstanding: m('0') });
    harness(d); await showPayments();
    const trigger = await screen.findByRole('button', { name: 'Открыть платёж 2' });
    expect(scheduleRow(1).textContent).toContain('200.00 USD');
    expect(scheduleRow(2).textContent).toContain('0.00 USD');
    expect(screen.getByText('Подтверждено по графику')).toBeTruthy();
    expect(screen.getByText('На проверке по графику')).toBeTruthy();
    expect(trigger.getAttribute('data-installment-invoice')).toBe('monthly-2');
    expect(trigger.getAttribute('type')).toBe('button');
    expect(trigger.classList.contains('btn')).toBe(true);
    expect(trigger.classList.contains('btn-secondary')).toBe(true);
    expect(trigger.classList.contains('btn-sm')).toBe(true);
    fireEvent.click(trigger);
    await screen.findByRole('dialog', { name: /Платёж рассрочки 2/ });
    const region = screen.getByRole('region', { name: 'Выбранный платёж рассрочки' }).textContent ?? '';
    expect(region).toContain('Остаток');
    expect(region).toContain('200.00 USD');
  });

  it('formats balances above Number.MAX_SAFE_INTEGER without precision loss', async () => {
    const d = deal();
    savedSchedule(d, '9007199254740993', [[1, '1'], [9007199254740992, '9007199254740992']]);
    harness(d); await showPayments();
    await screen.findByRole('button', { name: 'Открыть платёж 1' });
    expect(scheduleRow(1).textContent).toContain('90 071 992 547 409.92 USD');
    expect(scheduleRow(9007199254740992).textContent).toContain('0.00 USD');
  });

  it('shows exact confirmed, pending, partial, and unpaid facts for every saved row', async () => {
    const d = deal();
    updateInvoice(d, 'monthly-1', { paid: m('9007199254740993'), pending: m('0'), outstanding: m('0'), available: m('0') });
    updateInvoice(d, 'monthly-2', { paid: m('1'), pending: m('2'), outstanding: m('9007199254740993'), available: m('0') });
    harness(d); await showPayments();
    await screen.findByRole('button', { name: 'Открыть платёж 1' });
    expect(scheduleRow(1).textContent).toContain('Оплачен');
    expect(scheduleRow(2).textContent).toContain('Ожидает подтверждения · Частично оплачен');
  });
});

describe('sale servicing UI', () => {
  it('suppresses completed contract and registration actions independently while retaining other permissions', async () => {
    const d = deal(false);
    d.allowedActions = ['record-contract', 'record-registration', 'cancel'];
    d.contractSignedOn = '2024-01-01'; d.contractReference = 'SIGNED';
    const { state } = harness(d);
    await screen.findByText(/SIGNED от/);
    expect(screen.queryByRole('button', { name: 'Договор подписан' })).toBeNull();
    expect(screen.getByRole('button', { name: 'Регистрация' })).toBeTruthy();
    expect(screen.getByRole('button', { name: 'Отменить сделку' })).toBeTruthy();
    d.registeredOn = '2024-01-02'; d.plateNumber = '01A001AA'; d.registrationReference = 'REG';
    state.deal = d; click('Обновить данные сделки');
    await screen.findByText(/01A001AA/);
    expect(screen.queryByRole('button', { name: 'Регистрация' })).toBeNull();
    expect(screen.getByRole('button', { name: 'Отменить сделку' })).toBeTruthy();
  });

  it('uses the same bottom close handlers for details and selected payments', async () => {
    const { onClose } = harness();
    const dialog = await screen.findByRole('dialog', { name: /Сделка:/ });
    fireEvent.click(within(dialog.querySelector('.modal-footer')!).getByRole('button', { name: 'Закрыть' }));
    expect(onClose).toHaveBeenCalledTimes(1);
    cleanup();
    harness(); await showPayments(); fireEvent.click(await screen.findByRole('button', { name: 'Открыть платёж 2' }));
    const payment = screen.getByRole('dialog', { name: /Платёж рассрочки 2/ });
    fireEvent.click(within(payment.querySelector('.modal-footer')!).getByRole('button', { name: 'Закрыть' }));
    await screen.findByRole('button', { name: 'Открыть платёж 2' });
  });

  it('falls back to stable details focus when preparation has no supplied checklist', async () => {
    const d = deal(); d.status = 'reserved'; d.deliveredAt = null; d.installmentPlan!.state = 'planned'; d.allowedActions = [];
    delete d.checklist;
    harness(d); await showPayments(); fireEvent.click(await screen.findByRole('button', { name: 'Открыть платёж 2' }));
    click('К готовности к выдаче');
    const details = await screen.findByLabelText('Детали сделки');
    await waitFor(() => expect(document.activeElement).toBe(details));
  });
});

describe('generated installment terms and immutable plans', () => {
  it('opens historical installment from Sales and preserves its tab and original amounts', async () => {
    const { state } = harness(deal(), 'sales');
    fireEvent.click(await screen.findByRole('tab', { name: /Рассрочки/ }));
    expect(state.reads).toBe(0);
    fireEvent.click(await screen.findByRole('button', { name: 'Открыть график' }));
    await showPayments();
    await screen.findByRole('button', { name: 'Открыть платёж 2' }); oneDialog();
    expect(screen.queryByRole('button', { name: 'Рассчитать рассрочку' })).toBeNull();
    expect(screen.queryByRole('button', { name: 'Изменить условия рассрочки' })).toBeNull();
    expect(screen.getByText(/Договор SIGNED-2024/)).toBeTruthy();
    expect(screen.getAllByText('190.00 USD').length).toBeGreaterThan(0);
    closeFromFooter(); expect(screen.getByRole('button', { name: 'Открыть график' })).toBeTruthy();
  });

  it.each(['reserved', 'delivered'])('allows legacy %s terms without inventing contract, invoice, term or date', async status => {
    const d = { ...deal(false), status, invoices: [], contractSignedOn: null, contractReference: '' };
    const { state } = harness(d); await showPayments();
    fireEvent.click(await screen.findByRole('button', { name: 'Рассчитать рассрочку' })); oneDialog();
    for (const label of ['Первый взнос', 'Срок, месяцев', 'Дата первого платежа']) expect((screen.getByLabelText(label) as HTMLInputElement).value).toBe('');
    expect((screen.getByRole('button', { name: 'Сохранить условия' }) as HTMLButtonElement).disabled).toBe(true);
    change('Первый взнос', '10'); change('Срок, месяцев', '3'); change('Дата первого платежа', '2024-01-31');
    expect(within(screen.getByRole('region', { name: 'Предварительный график' })).getByText('2024-03-31')).toBeTruthy();
    state.onPost = request => {
      state.deal = { ...d, installmentDraft: calculateInstallment(d.price, request.body as any), revision: 'draft-rev' };
      return response({ data: state.deal });
    };
    click('Сохранить условия'); await screen.findByText('Условия сохранены.');
    expect(screen.getByText('Предварительный график — не подлежит оплате')).toBeTruthy();
    expect(screen.getByText('Нужен подписанный договор с датой и номером.')).toBeTruthy();
    expect(screen.getByText('Нужен один выставленный счёт первого взноса.')).toBeTruthy();
    expect(screen.queryByRole('button', { name: /Открыть платёж/ })).toBeNull();
    expect(state.requests[0]).toEqual({ url: '/api/v1/retail/deals/d-1/installment-terms', revision: '"deal-rev-7"', body: { downPayment: m('1000'), termMonths: 3, firstDueDate: '2024-01-31' } });
  });

  it('locks the issued down payment, saves only terms, and reloads immediate final server snapshot', async () => {
    const { state } = harness(deal(false));
    state.onPost = () => { state.deal = deal(); state.deal.installmentPlan!.contractReference = 'SERVER-FINAL'; return response({ data: state.deal }); };
    await editSchedule();
    expect((screen.getByLabelText('Первый взнос') as HTMLInputElement).readOnly).toBe(true);
    expect((screen.getByLabelText('Первый взнос') as HTMLInputElement).value).toBe('10.00');
    click('Сохранить условия'); await screen.findByText('Условия сохранены.'); oneDialog();
    expect(state.requests[0]!.body).toEqual({ downPayment: m('1000'), termMonths: 3, firstDueDate: '2024-01-31' });
    expect(screen.getByText(/Договор SERVER-FINAL/)).toBeTruthy();
    expect(screen.getByText('INITIAL-PAID')).toBeTruthy();
  });

  it('requires a committed preset or custom term before saving fixed-down installment terms', async () => {
    const { state } = harness(deal(false)); await editSchedule();
    const pickPreset = (name: string) => {
      fireEvent.focus(screen.getByRole('combobox', { name: 'Вариант срока' }));
      fireEvent.click(screen.getByRole('option', { name, exact: true }));
    };
    pickPreset('24 месяцев');
    fireEvent.change(screen.getByRole('combobox', { name: 'Вариант срока' }), { target: { value: 'unmatched' } });
    expect((screen.getByLabelText('Срок, месяцев') as HTMLInputElement).value).toBe('24');
    expect((screen.getByRole('button', { name: 'Сохранить условия' }) as HTMLButtonElement).disabled).toBe(true);
    click('Сохранить условия'); expect(state.requests).toHaveLength(0);
    fireEvent.change(screen.getByRole('combobox', { name: 'Вариант срока' }), { target: { value: '' } }); pickPreset('Другой срок');
    expect((screen.getByLabelText('Срок, месяцев') as HTMLInputElement).value).toBe('');
    change('Срок, месяцев', '7');
    expect((screen.getByLabelText('Первый взнос') as HTMLInputElement).readOnly).toBe(true);
    click('Сохранить условия'); await screen.findByText('Условия сохранены.');
    expect(state.requests[0]!.body).toEqual({ downPayment: m('1000'), termMonths: 7, firstDueDate: '2024-01-31' });
  });

  it('prefills a saved draft and invalid partial input removes the previously valid preview', async () => {
    const d = deal(false); d.installmentDraft = calculateInstallment(d.price, { downPayment: m('1000'), termMonths: 3, firstDueDate: '2024-01-31' });
    harness(d); await showPayments(); fireEvent.click(await screen.findByRole('button', { name: 'Изменить условия рассрочки' }));
    expect((screen.getByLabelText('Срок, месяцев') as HTMLInputElement).value).toBe('3');
    expect((screen.getByLabelText('Дата первого платежа') as HTMLInputElement).value).toBe('2024-01-31');
    change('Срок, месяцев', '');
    expect(screen.queryByRole('region', { name: 'Предварительный график' })).toBeNull();
    expect((screen.getByRole('button', { name: 'Сохранить условия' }) as HTMLButtonElement).disabled).toBe(true);
    change('Срок, месяцев', '3'); expect(screen.getByRole('region', { name: 'Предварительный график' })).toBeTruthy();
  });

  it.each([409, 412, 422])('retains terms on %s and never retries against a background revision', async status => {
    const { state, client } = harness(deal(false));
    state.onPost = () => failure(status, 'Конфликт условий', { termMonths: 'Проверьте срок' });
    await editSchedule();
    client.setQueryData(['deal', 'd-1'], { data: { ...deal(false), revision: 'new-unreviewed' } });
    click('Сохранить условия'); await screen.findByText(/Конфликт условий/);
    expect(state.requests[0]!.revision).toBe('"deal-rev-7"');
    expect((screen.getByLabelText('Срок, месяцев') as HTMLInputElement).value).toBe('3');
    expect((screen.getByLabelText('Дата первого платежа') as HTMLInputElement).value).toBe('2024-01-31');
    if (status !== 422) expect((screen.getByRole('button', { name: 'Сохранить условия' }) as HTMLButtonElement).disabled).toBe(true);
    click('Отмена'); expect(state.requests).toHaveLength(1); oneDialog();
  });

  it('prevents duplicate save and dismiss while busy', async () => {
    const { state } = harness(deal(false)); let finish!: () => void;
    state.onPost = () => new Promise(resolve => { finish = () => { state.deal = deal(); resolve(response({ data: state.deal })); }; });
    await editSchedule(); const button = screen.getByRole('button', { name: 'Сохранить условия' });
    fireEvent.click(button); fireEvent.click(button); closeFromFooter();
    expect(screen.getByText('Условия беспроцентной рассрочки')).toBeTruthy();
    expect((screen.getByRole('button', { name: 'Отмена' }) as HTMLButtonElement).disabled).toBe(true);
    expect(state.requests).toHaveLength(1); finish(); await screen.findByText('Условия сохранены.');
  });

  it('reports saved success separately from failed refresh, retries only GET', async () => {
    const { state } = harness(deal(false));
    state.onPost = () => { state.deal = deal(); state.failGet = true; return response({ data: state.deal }); };
    await editSchedule(); click('Сохранить условия'); await screen.findByText(/Условия сохранены.*Не удалось обновить данные/);
    state.failGet = false; click('Повторить загрузку сделки'); await screen.findByRole('button', { name: 'Открыть платёж 1' });
    expect(state.requests).toHaveLength(1);
  });

  it.each(['installmentPlan', 'installmentDraft'] as const)('distinguishes missing %s from loaded null and supports failed read retry', async field => {
    const d = deal(false); delete d[field]; const { state } = harness(d); await showPayments();
    await screen.findByText('Сервер не вернул данные графика.'); expect(screen.queryByRole('button', { name: 'Рассчитать рассрочку' })).toBeNull();
    state.failGet = true; click('Повторить загрузку графика'); await screen.findByText(/Не удалось загрузить данные сделки/);
    state.failGet = false; state.deal = deal(false); click('Повторить загрузку сделки'); await screen.findByRole('button', { name: 'Рассчитать рассрочку' });
    expect(screen.getByRole('tabpanel', { name: 'Оплаты' })).toBeTruthy();
    expect(state.requests).toHaveLength(0);
  });

  it.each(['reserved', 'cancelled'])('honors server permission for %s sales', async status => {
    harness({ ...deal(false), status, allowedActions: [] }); await showPayments();
    await screen.findByText(status === 'cancelled' ? 'Сделка отменена.' : 'Для изменения условий нет разрешения.');
    expect(screen.queryByRole('button', { name: 'Рассчитать рассрочку' })).toBeNull();
  });

  it('shows draft terms in contract action and locks only the dedicated first invoice, preserving registration editing', async () => {
    const d = { ...deal(false), status: 'reserved', contractSignedOn: null, invoices: [], allowedActions: ['record-contract', 'issue-invoice', 'set-installment-terms'] };
    d.installmentDraft = calculateInstallment(d.price, { downPayment: m('1234'), termMonths: 3, firstDueDate: '2024-01-31' });
    const { state } = harness(d); state.onPost = () => failure(422, 'Проверка счёта');
    fireEvent.click(await screen.findByRole('button', { name: 'Договор подписан' })); oneDialog();
    expect(screen.getByRole('region', { name: 'Предварительный график' })).toBeTruthy();
    expect(screen.getByText('12.34 USD')).toBeTruthy(); change(/Дата подписания/, '2024-01-01'); change(/Номер договора/, 'CONTRACT');
    click('Договор подписан'); await screen.findByText('Проверка счёта');
    expect(state.requests[0]!.body).toEqual({ signedOn: '2024-01-01', reference: 'CONTRACT', bindingIds: [] });
    click('Назад к сделке'); click('Выставить счёт первого взноса'); oneDialog();
    expect(screen.queryByLabelText(/Сумма/)).toBeNull(); expect(screen.queryByLabelText('Валюта')).toBeNull();
    expect(screen.getByText(/12.34 USD.*зафиксированы/)).toBeTruthy(); change(/Оплатить до/, '2024-01-30'); click('Выставить счёт первого взноса'); await screen.findByText('Проверка счёта');
    expect(state.requests[1]!.body).toEqual({ recipientSnapshot: 'Тестовый покупатель', dueDate: '2024-01-30', purpose: 'first-installment', amount: m('1234') });
    expect(state.requests[1]!.revision).toBe('"deal-rev-7"');
    click('Назад к сделке'); click('Выставить счёт'); oneDialog();
    expect(screen.queryByRole('option', { name: 'Первый взнос' })).toBeNull();
    fireEvent.click(screen.getByRole('combobox', { name: 'Назначение' })); fireEvent.click(await screen.findByRole('option', { name: 'Регистрация' }));
    change(/Сумма/, '5.25'); change(/Оплатить до/, '2024-01-30'); click('Выставить счёт'); await screen.findByText('Проверка счёта');
    expect(state.requests[2]!.body.amount).toEqual(m('525'));
    expect(state.requests[2]!.body.purpose).toBe('registration');
  });
});

describe('planned payment availability', () => {
  function plannedDeliveryDeal(deliver = false) {
    const d = deal();
    d.status = 'reserved'; d.deliveredAt = null; d.installmentPlan!.state = 'planned';
    d.allowedActions = deliver ? ['deliver'] : [];
    d.checklist = { contract: true, firstInstallment: false, insuranceApproved: undefined, registrationPaid: false, registered: false, policyResolved: false };
    updateInvoice(d, 'monthly-2', { allowedActions: [] });
    return d;
  }

  it('shows supplied readiness facts without a synthetic planned payment control', async () => {
    const { state } = harness(plannedDeliveryDeal()); await showPayments();
    fireEvent.click(await screen.findByRole('button', { name: 'Открыть платёж 2' }));
    expect(selected().queryByRole('button', { name: 'Внести оплату' })).toBeNull();
    expect(selected().getByText(/✓ Договор/)).toBeTruthy();
    expect(selected().getByText(/✗ Первый взнос/)).toBeTruthy();
    expect(selected().getByText(/✗ Оплата регистрации/)).toBeTruthy();
    expect(selected().getByText(/OD-01/)).toBeTruthy();
    expect(state.requests).toHaveLength(0);
  });

  it.each([true, false, undefined])('uses only planned invoice permission for payment entry: %s', async permission => {
    const d = plannedDeliveryDeal();
    updateInvoice(d, 'monthly-2', { allowedActions: permission === undefined ? undefined as unknown as string[] : permission ? ['submit-payment'] : [] });
    harness(d); await showPayments(); fireEvent.click(await screen.findByRole('button', { name: 'Открыть платёж 2' }));
    expect(selected().queryByRole('button', { name: 'Внести оплату' })).toEqual(permission ? expect.any(HTMLButtonElement) : null);
  });

  it('gates delivery on server permission and complete loaded selection', async () => {
    const { state } = harness(plannedDeliveryDeal(true)); await showPayments();
    fireEvent.click(await screen.findByRole('button', { name: 'Открыть платёж 2' }));
    fireEvent.click(selected().getByRole('button', { name: 'Выдать автомобиль' })); oneDialog();
    click('Отмена');
    await screen.findByRole('button', { name: 'Выдать автомобиль' });
    expect(state.requests).toHaveLength(0);
  });

  it('returns from cancelled delivery to the selected payment without posting', async () => {
    const { state } = harness(plannedDeliveryDeal(true)); await showPayments();
    fireEvent.click(await screen.findByRole('button', { name: 'Открыть платёж 2' }));
    click('Выдать автомобиль'); click('Назад к платежу');
    expect(screen.getByRole('dialog', { name: /Платёж рассрочки 2/ })).toBeTruthy();
    expect(selected().getByRole('button', { name: 'Выдать автомобиль' })).toBeTruthy();
    expect(state.requests).toHaveLength(0);
  });

  it('submits existing delivery payload and waits for refreshed invoice permission', async () => {
    const { state } = harness(plannedDeliveryDeal(true)); await showPayments();
    state.onPost = request => {
      state.deal.status = 'delivered'; state.deal.deliveredAt = '2024-02-28T10:30:00Z'; state.deal.allowedActions = [];
      state.deal.installmentPlan!.state = 'active'; updateInvoice(state.deal, 'monthly-2', { allowedActions: ['submit-payment'] });
      return response({ data: state.deal }, 201);
    };
    fireEvent.click(await screen.findByRole('button', { name: 'Открыть платёж 2' })); click('Выдать автомобиль');
    change('Когда выдан', '2024-02-28T10:30'); click('Выдать автомобиль');
    await screen.findByText('Выдать автомобиль: выполнено.');
    expect(selected().getByRole('button', { name: 'Внести оплату' })).toBeTruthy();
    expect(state.requests).toEqual([{ url: '/api/v1/retail/deals/d-1/deliveries', revision: '"deal-rev-7"', body: { occurredAt: '2024-02-28T05:30:00.000Z' } }]); oneDialog();
  });

  it('withholds payment after a refreshed delivery response denies permission', async () => {
    const { state } = harness(plannedDeliveryDeal(true)); await showPayments();
    state.onPost = () => {
      state.deal.status = 'delivered'; state.deal.deliveredAt = '2024-02-28T10:30:00Z'; state.deal.allowedActions = [];
      state.deal.installmentPlan!.state = 'active'; updateInvoice(state.deal, 'monthly-2', { allowedActions: [] });
      return response({ data: state.deal }, 201);
    };
    fireEvent.click(await screen.findByRole('button', { name: 'Открыть платёж 2' })); click('Выдать автомобиль');
    change('Когда выдан', '2024-02-28T10:30'); click('Выдать автомобиль');
    await screen.findByText('Выдать автомобиль: выполнено.');
    expect(selected().queryByRole('button', { name: 'Внести оплату' })).toBeNull();
    expect(state.requests).toHaveLength(1);
  });

  it('keeps successful delivery distinct from a failed refresh and retries only the GET', async () => {
    const { state } = harness(plannedDeliveryDeal(true)); await showPayments();
    state.onPost = () => { state.failGet = true; return response({ data: state.deal }, 201); };
    fireEvent.click(await screen.findByRole('button', { name: 'Открыть платёж 2' })); click('Выдать автомобиль');
    change('Когда выдан', '2024-02-28T10:30'); click('Выдать автомобиль');
    await screen.findByText(/Выдать автомобиль: выполнено.*Не удалось обновить данные/);
    expect(selected().queryByRole('button', { name: 'Внести оплату' })).toBeNull();
    expect(state.requests).toHaveLength(1);
    state.failGet = false; click('Повторить загрузку сделки');
    await waitFor(() => expect(state.reads).toBe(3));
    expect(state.requests).toHaveLength(1);
  });

  it('retains delivery draft on stale error and blocks duplicate busy submission', async () => {
    const { state } = harness(plannedDeliveryDeal(true)); await showPayments(); let finish!: () => void;
    state.onPost = () => new Promise(resolve => { finish = () => resolve(failure()); });
    fireEvent.click(await screen.findByRole('button', { name: 'Открыть платёж 2' })); click('Выдать автомобиль');
    change('Когда выдан', '2024-02-28T10:30'); const submit = screen.getByRole('button', { name: 'Выдать автомобиль' });
    fireEvent.click(submit); fireEvent.click(submit); await waitFor(() => expect(state.requests).toHaveLength(1)); finish();
    await screen.findByText(/Вернитесь к сделке и обновите данные/);
    expect((screen.getByLabelText('Когда выдан') as HTMLInputElement).value).toBe('2024-02-28T10:30');
  });

  it('targets preparation readiness beyond the long schedule', async () => {
    const d = plannedDeliveryDeal();
    for (let number = 3; number <= 36; number++) d.invoices!.push(invoice(`monthly-${number}`, '100', 'monthly-installment', number));
    d.installmentPlan!.rows = d.invoices!.filter(i => i.purpose === 'monthly-installment').map(i => ({ number: i.installmentNumber!, invoiceId: i.id, dueDate: i.dueDate!, amount: i.amount, paid: i.paid, pending: i.pending, outstanding: i.outstanding, available: i.available, allowedActions: i.allowedActions }));
    harness(d); await showPayments(); fireEvent.click(await screen.findByRole('button', { name: 'Открыть платёж 36' }));
    click('К готовности к выдаче');
    const readiness = await screen.findByLabelText('Готовность к выдаче');
    await waitFor(() => expect(document.activeElement).toBe(readiness)); oneDialog();
  });
});

describe('selected installment payment and finance decisions', () => {
  it.each([
    [1, 'close'], [2, 'back'], [36, 'escape'],
  ] as const)('opens a long-plan payment %i and restores schedule scroll and trigger via %s', async (number, navigation) => {
    const d = deal();
    for (let number = 3; number <= 36; number++) d.invoices!.push(invoice(`monthly-${number}`, '100', 'monthly-installment', number));
    d.installmentPlan!.rows = d.invoices!.filter(i => i.purpose === 'monthly-installment').map(i => ({ number: i.installmentNumber!, invoiceId: i.id, dueDate: i.dueDate!, amount: i.amount, paid: i.paid, pending: i.pending, outstanding: i.outstanding, available: i.available, allowedActions: i.allowedActions }));
    harness(d); await showPayments();
    const trigger = await screen.findByRole('button', { name: `Открыть платёж ${number}` });
    const body = trigger.closest<HTMLElement>('.modal-body')!;
    Object.defineProperty(body, 'scrollTop', { value: 240, writable: true });
    expect(document.activeElement).not.toBe(trigger);
    fireEvent.click(trigger);
    expect(screen.getByRole('dialog', { name: new RegExp(`Платёж рассрочки ${number} ·`) })).toBeTruthy();
    expect(selected().getByText(/Тестовый покупатель.*дата платежа/)).toBeTruthy();
    expect(selected().getByRole('button', { name: 'Внести оплату' })).toBeTruthy();
    expect(screen.queryByText('INITIAL-PAID')).toBeNull();
    expect(screen.queryByRole('button', { name: 'Открыть платёж 1' })).toBeNull(); oneDialog();
    if (navigation === 'escape') fireEvent.keyDown(document, { key: 'Escape' });
    else if (navigation === 'close') closeFromFooter(); else click('Назад к графику');
    const returnedTrigger = await screen.findByRole('button', { name: `Открыть платёж ${number}` });
    await waitFor(() => { expect(returnedTrigger.closest<HTMLElement>('.modal-body')!.scrollTop).toBe(240); expect(document.activeElement).toBe(returnedTrigger); });
    oneDialog();
    const next = number === 2 ? 1 : 2;
    fireEvent.click(await screen.findByRole('button', { name: `Открыть платёж ${next}` }));
    expect(screen.getByRole('dialog', { name: new RegExp(`Платёж рассрочки ${next} ·`) })).toBeTruthy();
    fireEvent.keyDown(document, { key: 'Escape' });
    await screen.findByRole('button', { name: `Открыть платёж ${next}` }); oneDialog();
  });

  it.each(['row', 'invoice'] as const)('retries a missing selected payment %s and recovers in the same dialog', async missing => {
    const d = deal();
    if (missing === 'row') d.installmentPlan!.rows = d.installmentPlan!.rows.filter(row => row.invoiceId !== 'monthly-2');
    else d.invoices = d.invoices!.filter(invoice => invoice.id !== 'monthly-2');
    const { state } = harness(d, 'deal', 'monthly-2');
    await screen.findByText(/Выбранный платёж не найден в графике/); oneDialog();
    expect(screen.queryByRole('button', { name: 'Внести оплату' })).toBeNull();
    expect(screen.queryByRole('button', { name: 'Выдать автомобиль' })).toBeNull();
    state.failGet = true; click('Повторить загрузку сделки');
    await screen.findByText(/Не удалось загрузить сделку.*Сервер временно недоступен/);
    state.failGet = false; state.deal = deal(); click('Повторить загрузку сделки');
    await screen.findByRole('button', { name: 'Внести оплату' });
    expect(screen.getByRole('dialog', { name: /Платёж рассрочки 2 ·/ })).toBeTruthy();
    expect(screen.queryByRole('button', { name: 'Открыть платёж 1' })).toBeNull();
    expect(screen.queryByText(/Выбранный платёж не найден/)).toBeNull();
    expect(state.reads).toBe(3); expect(state.requests).toHaveLength(0); oneDialog();
  });

  it('preserves active payment control focus during a same-selection query refresh', async () => {
    const { state, client } = harness(); await showPayments();
    fireEvent.click(await screen.findByRole('button', { name: 'Открыть платёж 2' }));
    const control = selected().getByRole('button', { name: 'Внести оплату' });
    control.focus();
    state.deal = deal();
    updateInvoice(state.deal, 'monthly-2', { paymentEvidence: [{ ...evidence('accepted', []), externalReference: 'REFRESHED-PAYMENT' }] });
    let finish!: () => void;
    state.readGate = new Promise<void>(resolve => { finish = resolve; });
    let refresh!: Promise<void>;
    act(() => { refresh = client.refetchQueries({ queryKey: ['deal', 'd-1'] }); });
    await waitFor(() => expect(state.reads).toBe(2));
    expect(document.activeElement).toBe(control);
    await act(async () => { finish(); await refresh; });
    await screen.findByText('REFRESHED-PAYMENT');
    expect(selected().getByRole('button', { name: 'Внести оплату' })).toBe(control);
    expect(document.activeElement).toBe(control); oneDialog();
    expect(state.requests).toHaveLength(0);
  });

  it('posts 10.25 as a selected monthly allocation with file evidence, reloads pending history, keeps one dialog', async () => {
    const { state } = harness();
    state.onPost = () => {
      updateInvoice(state.deal, 'monthly-2', { pending: m('1025'), available: m('17975'), paymentEvidence: [evidence('submitted', [])] });
      return response({ data: state.deal.invoices![3] }, 201);
    };
    await openPayment();
    fireEvent.change(screen.getByLabelText(/Подтверждение \(файл\)/), { target: { files: [new File(['proof'], 'proof.pdf', { type: 'application/pdf' })] } });
    click('Отправить на проверку'); await screen.findByText(/Платёж отправлен на проверку/); oneDialog();
    await waitFor(() => expect((screen.getByRole('button', { name: 'Назад' }) as HTMLButtonElement).disabled).toBe(false)); click('Назад');
    expect(state.requests).toEqual([{ url: '/api/v1/retail/deals/d-1/installment-payments', revision: null, body: {
      claimedAmount: m('1025'), paidOn: '2024-02-28', externalReference: 'PAY-2', attachmentBindingIds: ['uploaded-proof'],
      allocations: [{ invoiceId: 'monthly-2', amount: m('1025') }],
    } }]);
    expect(selected().getByText('PAY-2')).toBeTruthy();
    expect(selected().getByRole('button', { name: 'Внести оплату' })).toBeTruthy();
    click('Назад к графику');
    expect(screen.getByText('INITIAL-PAID')).toBeTruthy();
    expect(state.deal.invoices![2]!.pending.amountMinor).toBe('0');
    expect(screen.getByRole('link', { name: 'Скан договора 1' }).getAttribute('href')).toBe('/api/v1/documents/files/saved-contract/content');
  });

  it('retains money/date/reference/file draft after error, and prevents close or duplicate submission while busy', async () => {
    const { state } = harness(); let finish!: () => void;
    let finishUpload!: () => void; state.uploadGate = new Promise(resolve => { finishUpload = resolve; });
    state.onPost = () => new Promise(resolve => { finish = () => resolve(failure(422, 'Документ ещё проверяется', { claimedAmount: 'Проверьте сумму' })); });
    await openPayment(); const file = new File(['x'], 'proof.pdf', { type: 'application/pdf' });
    const upload = screen.getByLabelText(/Подтверждение \(файл\)/) as HTMLInputElement;
    fireEvent.change(upload, { target: { files: [file] } });
    const submit = screen.getByRole('button', { name: 'Отправить на проверку' }); fireEvent.click(submit); fireEvent.click(submit);
    closeFromFooter(); fireEvent.click(screen.getByRole('dialog').querySelector<HTMLButtonElement>('.modal-close')!); fireEvent.keyDown(document, { key: 'Escape' }); click('Назад');
    expect(screen.getByRole('region', { name: 'Ручное распределение платежа' })).toBeTruthy(); expect(state.requests).toHaveLength(0);
    await act(async () => { finishUpload(); });
    await waitFor(() => expect(state.requests).toHaveLength(1)); closeFromFooter(); fireEvent.keyDown(document, { key: 'Escape' }); click('Назад'); oneDialog();
    expect(screen.getByLabelText('Сумма месяца 2')).toBeTruthy(); finish();
    await screen.findByText(/Документ ещё проверяется/);
    expect((screen.getByLabelText('Сумма месяца 2') as HTMLInputElement).value).toBe('10.25');
    expect((screen.getByLabelText(/Дата оплаты/) as HTMLInputElement).value).toBe('2024-02-28');
    expect((screen.getByLabelText(/Номер платёжки/) as HTMLInputElement).value).toBe('PAY-2');
    expect(upload.files?.[0]).toBe(file); click('Назад');
    expect(screen.getByRole('dialog', { name: /Платёж рассрочки 2/ })).toBeTruthy();
  });

  it('validates amount/currency and required evidence details without posting', async () => {
    const { state } = harness(); await openPayment();
    expect(screen.queryByLabelText('Валюта')).toBeNull();
    const submit = screen.getByRole('button', { name: 'Отправить на проверку' }) as HTMLButtonElement;
    for (const amount of ['0', '191']) { change('Сумма месяца 2', amount); change('Сумма по чеку', amount); expect(submit.disabled).toBe(true); fireEvent.click(submit); }
    change('Сумма месяца 2', '10.25'); change('Сумма по чеку', '10'); expect(submit.disabled).toBe(true);
    change('Сумма по чеку', '10.25'); change(/Дата оплаты/, ''); expect(submit.disabled).toBe(true);
    expect(state.requests).toHaveLength(0); click('Назад'); oneDialog();
  });

  it.each(['planned', 'cancelled', 'readonly', 'missing-actions', 'pending', 'paid'])('shows history with no unauthorized payment or decision: %s', async mode => {
    const d = deal(); const patch: Partial<RetailInvoice> = { allowedActions: [], paymentEvidence: [evidence('submitted', [])] };
    if (mode === 'missing-actions') patch.allowedActions = undefined as unknown as string[];
    if (mode === 'pending') { patch.pending = m('19000'); patch.available = m('0'); }
    if (mode === 'paid') { patch.outstanding = m('0'); patch.available = m('0'); patch.paid = m('19000'); patch.paymentEvidence = [evidence('accepted', [])]; }
    if (mode === 'planned' || mode === 'cancelled') { d.installmentPlan!.state = mode; d.status = mode === 'planned' ? 'reserved' : 'cancelled'; }
    updateInvoice(d, 'monthly-2', patch); harness(d); await showPayments();
    fireEvent.click(await screen.findByRole('button', { name: 'Открыть платёж 2' }));
    expect(selected().getByText('PAY-2')).toBeTruthy();
    const payment = selected().queryByRole('button', { name: 'Внести оплату' });
    expect(payment).toBeNull();
    expect(screen.queryByRole('button', { name: 'Принять' })).toBeNull(); expect(screen.queryByRole('button', { name: 'Отклонить' })).toBeNull();
    if (mode === 'pending') expect(screen.getByText(/Вся оставшаяся сумма на проверке/)).toBeTruthy();
    if (mode === 'paid') expect(selected().getByText('Платёж полностью оплачен.')).toBeTruthy();
  });

  it.each(['accept', 'reject'])('sends finance %s with evidence revision and refreshes selected-row capacity', async action => {
    const d = deal(); updateInvoice(d, 'monthly-2', { paymentEvidence: [evidence()], pending: m('1025'), available: m('17975'), allowedActions: [] });
    const { state } = harness(d); await showPayments(); const label = action === 'accept' ? 'Принять' : 'Отклонить';
    state.onPost = () => {
      updateInvoice(state.deal, 'monthly-2', { paymentEvidence: [{ ...evidence(action === 'accept' ? 'accepted' : 'rejected', []), decisionReason: action === 'reject' ? 'Нет зачисления' : '' }],
        pending: m('0'), paid: m(action === 'accept' ? '1025' : '0'), outstanding: m(action === 'accept' ? '17975' : '19000'), available: m(action === 'accept' ? '17975' : '19000'), allowedActions: ['submit-payment'] });
      return response({ data: state.deal.invoices![3] });
    };
    fireEvent.click(await screen.findByRole('button', { name: 'Открыть платёж 2' })); click(label); oneDialog();
    click(label); await screen.findByText(action === 'accept' ? 'Подтвердите поступление денег.' : 'Укажите причину отклонения.');
    expect(state.requests).toHaveLength(0);
    if (action === 'accept') fireEvent.click(screen.getByLabelText('Деньги поступили')); else change(/Причина/, 'Нет зачисления');
    click(label); await screen.findByText(`${label}: выполнено.`); oneDialog();
    expect(state.requests).toEqual([{ url: `/api/v1/retail/evidence/e-2/${action}`, revision: '"evidence-rev-3"', body: action === 'accept' ? { confirmation: true } : { reason: 'Нет зачисления' } }]);
    expect(selected().getByRole('button', { name: 'Внести оплату' })).toBeTruthy();
    expect(selected().getByText(action === 'accept' ? 'Принято' : 'Отклонено: Нет зачисления')).toBeTruthy();
  });

  it('retains finance rejection reason after stale decision and returns to the same row', async () => {
    const d = deal(); updateInvoice(d, 'monthly-2', { paymentEvidence: [evidence()] });
    const { state } = harness(d); await showPayments(); state.onPost = () => failure();
    fireEvent.click(await screen.findByRole('button', { name: 'Открыть платёж 2' })); click('Отклонить'); change(/Причина/, 'Проверить банк'); click('Отклонить');
    await screen.findByText(/Вернитесь к сделке и обновите данные/);
    expect((screen.getByLabelText(/Причина/) as HTMLTextAreaElement).value).toBe('Проверить банк');
    expect(state.requests).toHaveLength(1); click('Назад к платежу'); oneDialog(); expect(selected().getByText('PAY-2')).toBeTruthy();
  });

  it('opens the same monthly invoice from Billing without stacking, and preserves the clients tab', async () => {
    harness(deal(), 'billing'); fireEvent.click(await screen.findByRole('tab', { name: 'От клиентов' }));
    const title = await screen.findByText('Платёж рассрочки 2');
    fireEvent.click(within(title.closest('tr')!).getByRole('button', { name: 'Открыть' }));
    await screen.findByRole('button', { name: 'Внести оплату' }); oneDialog();
    fireEvent.click(selected().getByRole('button', { name: 'Внести оплату' })); oneDialog(); click('Назад');
    expect(screen.getByRole('dialog', { name: /Платёж рассрочки 2/ })).toBeTruthy(); closeFromFooter();
    expect(screen.getByText('Платёж рассрочки 2')).toBeTruthy();
  });

  it('keeps first-installment billing payment usable with server permissions and exact payload', async () => {
    const d = deal(); updateInvoice(d, 'initial', { paid: m('0'), outstanding: m('1000'), available: m('1000'), allowedActions: ['submit-payment'], paymentEvidence: [] });
    const { state } = harness(d, 'billing'); fireEvent.click(await screen.findByRole('tab', { name: 'От клиентов' }));
    const title = await screen.findByText('Первый взнос'); fireEvent.click(within(title.closest('tr')!).getByRole('button', { name: 'Открыть' }));
    fireEvent.click(await screen.findByRole('button', { name: 'Внести оплату' }));
    change(/Сумма/, '5.25'); change(/Дата оплаты/, '2024-01-01'); change(/Номер платёжки/, 'INITIAL-NEW');
    fireEvent.click(within(screen.getByRole('dialog', { name: 'Внести оплату' })).getByRole('button', { name: 'Внести оплату' }));
    await waitFor(() => expect(state.requests).toHaveLength(1));
    expect(state.requests[0]).toEqual({ url: '/api/v1/retail/invoices/initial/evidence', revision: null, body: { claimedAmount: m('525'), paidOn: '2024-01-01', externalReference: 'INITIAL-NEW', attachmentBindingIds: [] } });
    await waitFor(() => expect(screen.queryByRole('dialog', { name: 'Внести оплату' })).toBeNull());
  });
});

describe('grouped installment payment integration', () => {
  function grouped(d: Deal, actions: string[] = ['accept', 'reject']) {
    const payment = { id: 'group-1', dealId: d.id, installmentPlanId: d.installmentPlan!.id, claimedAmount: m('1525'), paidOn: '2024-02-28', externalReference: 'GROUP-2', attachmentIds: ['proof'], status: 'submitted' as const, decisionReason: '', revision: 'group-rev-1', allowedActions: actions,
      allocations: [{ invoiceId: 'monthly-1', evidenceId: 'e-1', number: 1, amount: m('500') }, { invoiceId: 'monthly-2', evidenceId: 'e-2', number: 2, amount: m('1025') }] };
    d.installmentPlan!.payments = [payment];
    updateInvoice(d, 'monthly-1', { pending: m('500'), available: m('9500'), paymentGroups: [payment], paymentEvidence: [{ ...evidence('submitted', []), id: 'e-1', amount: m('500'), externalReference: payment.externalReference, paymentGroupId: payment.id }] });
    updateInvoice(d, 'monthly-2', { pending: m('1025'), available: m('17975'), paymentGroups: [payment], paymentEvidence: [{ ...evidence('submitted', []), externalReference: payment.externalReference, paymentGroupId: payment.id }] });
    d.installmentPlan!.pending = m('1525'); d.installmentPlan!.available = m('27475');
    return payment;
  }

  it('uses only projected plan and row actions for reserved manual entry and selected prefill', async () => {
    const d = deal(); d.status = 'reserved'; d.deliveredAt = null; d.installmentPlan!.state = 'planned'; d.installmentPlan!.allowedActions = ['submit-installment-payment'];
    harness(d); await showPayments();
    fireEvent.click(await screen.findByRole('button', { name: 'Внести оплату' }));
    await screen.findByRole('region', { name: 'Ручное распределение платежа' }); oneDialog();
    click('Назад');
    await waitFor(() => expect(document.activeElement).toBe(screen.getByRole('button', { name: 'Внести оплату' })));
    fireEvent.click(await screen.findByRole('button', { name: 'Открыть платёж 2' })); click('Внести оплату');
    await screen.findByRole('region', { name: 'Ручное распределение платежа' });
    expect((screen.getByLabelText('Выбрать месяц 2') as HTMLInputElement).checked).toBe(true);
    expect(screen.getByText(/график считается закрытым только после подтверждения/)).toBeTruthy();
  });

  it('does not infer plan or row payment permission and keeps cancelled plans inactive', async () => {
    const d = deal(); d.status = 'cancelled'; d.installmentPlan!.state = 'cancelled'; d.installmentPlan!.allowedActions = ['submit-installment-payment'];
    updateInvoice(d, 'monthly-2', { allowedActions: [] }); harness(d); await showPayments();
    await screen.findByText('Сделка отменена. Платежи по графику недоступны.');
    expect(screen.queryByRole('button', { name: 'Погасить досрочно' })).toBeNull();
    fireEvent.click(screen.getByRole('button', { name: 'Открыть платёж 2' }));
    expect(selected().queryByRole('button', { name: 'Внести оплату' })).toBeNull();
  });

  it('reviews a complete parent from Billing and suppresses grouped child decisions', async () => {
    const d = deal(); grouped(d); harness(d, 'billing'); fireEvent.click(await screen.findByRole('tab', { name: 'От клиентов' }));
    fireEvent.click(within((await screen.findByText('Платёж рассрочки 2')).closest('tr')!).getByRole('button', { name: 'Открыть' }));
    await waitFor(() => expect(selected().getByRole('button', { name: 'Открыть весь чек' })).toBeTruthy());
    expect(selected().queryByRole('button', { name: 'Принять' })).toBeNull();
    click('Открыть весь чек'); await screen.findByRole('region', { name: 'Проверка оплаты рассрочки' }); oneDialog();
    expect(screen.getByRole('region', { name: 'Проверка оплаты рассрочки' }).textContent).toContain('GROUP-2');
    expect(screen.getByRole('region', { name: 'Проверка оплаты рассрочки' }).textContent).toContain('10.25 USD');
    expect(within(screen.getByRole('region', { name: 'Проверка оплаты рассрочки' })).getAllByRole('row')).toHaveLength(3);
    expect(within(screen.getByRole('region', { name: 'Проверка оплаты рассрочки' })).getByText('5.00 USD')).toBeTruthy();
    expect(screen.getByLabelText('Деньги поступили')).toBeTruthy();
    click('Назад'); closeFromFooter(); expect(screen.queryByRole('dialog')).toBeNull();
  });

  it('shows settlement only from confirmed facts and retains post success when refresh fails', async () => {
    const d = deal(); d.status = 'reserved'; d.deliveredAt = null; d.installmentPlan!.state = 'planned'; d.installmentPlan!.allowedActions = ['submit-installment-payment'];
    const { state } = harness(d); await showPayments(); fireEvent.click(await screen.findByRole('button', { name: 'Погасить досрочно' }));
    expect(screen.getByText(/график считается закрытым только после подтверждения/)).toBeTruthy();
    fireEvent.click(screen.getByLabelText('Выбрать месяц 2')); change('Сумма месяца 1', '100'); change('Сумма по чеку', '100'); change('Дата оплаты', '2024-02-28'); change('Номер платёжки / чека', 'GROUP-POST');
    state.onPost = () => { state.failGet = true; return response({ data: {} }, 201); };
    click('Отправить на проверку'); await screen.findByText(/Операция сохранена. Не удалось обновить данные/);
    expect(screen.getByText(/Платёж отправлен на проверку/)).toBeTruthy();
    expect(state.requests).toHaveLength(1); expect(screen.queryByRole('button', { name: 'Отправить на проверку' })).toBeNull();
    state.failGet = false; click('Повторить загрузку сделки');
    await waitFor(() => expect((screen.getByRole('button', { name: 'Назад' }) as HTMLButtonElement).disabled).toBe(false)); click('Назад');
    expect(screen.queryByText('Расчёты по ежемесячному графику подтверждены.')).toBeNull();
    expect(state.requests).toHaveLength(1);
  });

  it.each(['plan', 'row', 'invoice'] as const)('withholds selected entry when projected permission is missing: %s', async missing => {
    const d = deal();
    if (missing === 'plan') d.installmentPlan!.allowedActions = undefined;
    if (missing === 'row') d.installmentPlan!.rows[1]!.allowedActions = undefined as unknown as string[];
    if (missing === 'invoice') d.invoices![3]!.allowedActions = undefined as unknown as string[];
    const { state } = harness(d); await showPayments();
    fireEvent.click(await screen.findByRole('button', { name: 'Открыть платёж 2' }));
    expect(selected().queryByRole('button', { name: 'Внести оплату' })).toBeNull();
    expect(screen.queryByRole('region', { name: 'Ручное распределение платежа' })).toBeNull();
    expect(state.requests).toHaveLength(0);
  });

  it('retries missing complete parent details without child decisions', async () => {
    const d = deal(); const parent = grouped(d); d.invoices![3]!.paymentGroups = [];
    const { state } = harness(d, 'deal', 'monthly-2');
    await screen.findByText(/Не удалось загрузить весь чек/);
    expect(screen.queryByRole('button', { name: 'Принять' })).toBeNull();
    state.deal.invoices![3]!.paymentGroups = [parent]; click('Повторить загрузку сделки');
    await screen.findByRole('button', { name: 'Открыть весь чек' }); click('Открыть весь чек');
    expect(within(screen.getByRole('region', { name: 'Проверка оплаты рассрочки' })).getAllByRole('row')).toHaveLength(3);
    expect(state.requests).toHaveLength(0);
  });

  it('retains a parent decision and retries failed Billing projection with GET only while blocking dismiss', async () => {
    const d = deal(); grouped(d); const { state, onClose } = harness(d, 'billing');
    fireEvent.click(await screen.findByRole('tab', { name: 'От клиентов' }));
    fireEvent.click(within((await screen.findByText('Платёж рассрочки 2')).closest('tr')!).getByRole('button', { name: 'Открыть' }));
    await screen.findByRole('button', { name: 'Открыть весь чек' }); click('Открыть весь чек');
    state.onPost = () => {
      state.failBilling = true;
      const parent = state.deal.installmentPlan!.payments![0]!; parent.status = 'accepted'; parent.allowedActions = []; parent.revision = 'group-rev-2';
      for (const allocation of parent.allocations) updateInvoice(state.deal, allocation.invoiceId, { paid: allocation.amount, pending: m('0'), outstanding: m(allocation.number === 1 ? '9500' : '17975'), available: m(allocation.number === 1 ? '9500' : '17975'), paymentEvidence: [{ ...evidence('accepted', []), id: allocation.evidenceId, amount: allocation.amount, paymentGroupId: parent.id }], paymentGroups: [parent] });
      state.deal.installmentPlan!.paid = m('1525'); state.deal.installmentPlan!.pending = m('0'); state.deal.installmentPlan!.outstanding = m('27475');
      return response({ data: parent });
    };
    fireEvent.click(screen.getByLabelText('Деньги поступили')); click('Принять весь чек');
    await screen.findByText(/Операция сохранена. Не удалось обновить данные/);
    expect(screen.getByText('Статус: Принято')).toBeTruthy();
    expect(screen.queryByRole('button', { name: 'Принять весь чек' })).toBeNull();
    const billingReads = state.billingReads;
    let finish!: () => void; state.readGate = new Promise(resolve => { finish = resolve; }); state.failBilling = false;
    click('Повторить загрузку сделки');
    await waitFor(() => expect(state.billingReads).toBeGreaterThan(billingReads));
    closeFromFooter(); fireEvent.click(screen.getByRole('dialog').querySelector<HTMLButtonElement>('.modal-close')!); fireEvent.keyDown(document, { key: 'Escape' }); click('Назад');
    expect(screen.getByRole('region', { name: 'Проверка оплаты рассрочки' })).toBeTruthy(); expect(onClose).not.toHaveBeenCalled();
    await act(async () => { finish(); });
    await waitFor(() => expect(screen.queryByText(/Операция сохранена. Не удалось обновить данные/)).toBeNull());
    expect(state.requests).toEqual([{ url: '/api/v1/retail/installment-payments/group-1/accept', body: { confirmation: true }, revision: '"group-rev-1"' }]);
    click('Назад'); click('Открыть весь чек');
    expect(screen.getByText('Статус: Принято')).toBeTruthy(); expect(screen.queryByRole('button', { name: 'Принять весь чек' })).toBeNull();
  });

  it.each(['pending', 'settled'] as const)('uses consistent confirmed settlement facts for %s schedule', async status => {
    const d = deal(); const paid = status === 'settled';
    for (const i of d.invoices!.filter(i => i.purpose === 'monthly-installment')) updateInvoice(d, i.id, { paid: paid ? i.amount : m('0'), pending: paid ? m('0') : i.amount, outstanding: paid ? m('0') : i.amount, available: m('0'), allowedActions: [] });
    Object.assign(d.installmentPlan!, { paid: m(paid ? '29000' : '0'), pending: m(paid ? '0' : '29000'), outstanding: m(paid ? '0' : '29000'), available: m('0'), settlementState: paid ? 'settled' : 'outstanding', allowedActions: [] });
    harness(d); await showPayments(); await screen.findByRole('button', { name: 'Открыть платёж 2' });
    expect(!!screen.queryByText('Расчёты по ежемесячному графику подтверждены.')).toBe(paid);
    expect(screen.queryByRole('button', { name: 'Погасить досрочно' })).toBeNull();
  });

  it('provides bottom close during initial loading and read error and guards explicit retry', async () => {
    const { state, onClose } = harness();
    let finish!: () => void; state.readGate = new Promise(resolve => { finish = resolve; });
    await screen.findByText('Загрузка сделки…'); closeFromFooter(); expect(onClose).toHaveBeenCalledOnce();
    state.failGet = true; await act(async () => { finish(); });
    await screen.findByText(/Не удалось загрузить данные сделки/); closeFromFooter(); expect(onClose).toHaveBeenCalledTimes(2);
    state.readGate = new Promise(resolve => { finish = resolve; }); state.failGet = false;
    click('Повторить загрузку сделки'); await waitFor(() => expect(state.reads).toBe(2));
    closeFromFooter(); fireEvent.keyDown(document, { key: 'Escape' }); fireEvent.click(screen.getByRole('dialog').querySelector<HTMLButtonElement>('.modal-close')!);
    expect(onClose).toHaveBeenCalledTimes(2);
    await act(async () => { finish(); }); await showPayments(); await screen.findByRole('button', { name: 'Открыть платёж 2' });
  });
});
