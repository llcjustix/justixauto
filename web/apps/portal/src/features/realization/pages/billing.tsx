import { useState } from 'react';
import {
  Badge,
  Button,
  Cell,
  FilterSelect,
  Modal,
  Page,
  Panel,
  ResultMeta,
  Stat,
  Stats,
  Table,
  Tabs,
  Toolbar,
  date,
  get,
  list,
  matches,
  money,
  plural,
  useData,
  useSearchQuery,
  useSession,
} from '@justixauto/kit';
import { purposeLabel, sumMoney, useLinesLabel } from '../data';
import type { Deal, InstallmentPaymentGroup, Invoice, Money, Order, Reservation, ReservationAccounting, RetailInvoice } from '../data';
import { DealDialog, RetailInvoicePanel } from './retail';
import { InvoicePanel, OrderDialog } from './trade';
import { InstallmentPaymentReview } from '../installment-payment-review';
import { ReservationPayment } from '../reservation-payment';
import { useFinanceWorkspace } from '../../accounting/finance-workspace';

type State = 'due' | 'review' | 'paid' | 'void';
const stateLabel: Record<State, [string, 'warning' | 'info' | 'success' | undefined]> = {
  due: ['К оплате', 'warning'],
  review: ['На сверке', 'info'],
  paid: ['Оплачен', 'success'],
  void: ['Аннулирован', undefined],
};
const stateOf = (i: { status: string; outstanding: Money; pending: Money }): State =>
  i.status === 'void'
    ? 'void'
    : i.outstanding.amountMinor === '0'
      ? 'paid'
      : i.pending.amountMinor !== '0'
        ? 'review'
        : 'due';
const moneyList = (ms: Money[]) => (ms.length ? ms.map(money).join(' + ') : '0');
const today = new Date().toISOString().slice(0, 10);

export function BillingPage() {
  const session = useSession();
  const accounting = useFinanceWorkspace();
  const canCommerce = session.can('commerce.read');
  const canRetail = session.can('retail.read');
  const canEnterAccounting = (canRetail && (session.can('retail.invoices.manage') || session.can('retail.payments.accept')))
    || (canCommerce && (session.can('commerce.invoices.manage') || session.can('commerce.payments.accept')));
  const [selected, setSelected] = useState<'suppliers' | 'clients'>('suppliers');
  const tabs: ['suppliers' | 'clients', string][] = [
    ...(canCommerce ? [['suppliers', 'B2B'] as ['suppliers', string]] : []),
    ...(canRetail ? [['clients', 'От клиентов'] as ['clients', string]] : []),
  ];
  const channel = tabs.some(([key]) => key === selected) ? selected : tabs[0]![0];
  return (
    <Page
      title="Счета и оплаты"
      subtitle="Расчёты по закупкам и розничным продажам. Деньги через платформу не проходят — фиксируются подтверждения."
    >
      {!accounting && <Panel title="Решения по счетам и оплатам" padded><p>Счета выставляет и оплаты подтверждает бухгалтер. {canEnterAccounting && <a href="/accounting/billing">Открыть кабинет бухгалтерии</a>}</p></Panel>}
      <Tabs
        channel
        value={channel}
        onChange={setSelected}
        tabs={tabs}
      />
      {channel === 'suppliers' ? <SupplierInvoices enabled={canCommerce} /> : <ClientInvoices enabled={canRetail} />}
    </Page>
  );
}

function Summary({
  rows,
  overdue,
}: {
  rows: { status: string; total: Money; paid: Money; pending: Money; outstanding: Money }[];
  overdue: Money[];
}) {
  const live = rows.filter((r) => r.status !== 'void');
  return (
    <Stats>
      <Stat
        label="К оплате"
        value={moneyList(sumMoney(live.map((r) => r.outstanding)))}
        note="остаток по выставленным счетам"
      />
      <Stat label="На сверке" value={moneyList(sumMoney(live.map((r) => r.pending)))} note="подтверждение отправлено" />
      <Stat label="Оплачено" value={moneyList(sumMoney(live.map((r) => r.paid)))} note="принятые оплаты" />
      <Stat
        label="Просрочено"
        value={moneyList(sumMoney(overdue))}
        note={overdue.length ? 'срок оплаты прошёл' : 'нет просрочки'}
      />
    </Stats>
  );
}

type SupplierRow = Invoice & { order: Order };
type SupplierData = { orders: Order[]; rows: SupplierRow[] };

function SupplierInvoices({ enabled }: { enabled: boolean }) {
  const accountingWorkspace = useFinanceWorkspace();
  const session = useSession();
  const companyId = session.company?.id ?? '';
  const canIssue = accountingWorkspace && session.can('commerce.invoices.manage');
  const advances = useData(['billing', 'reservations', companyId], async () => {
    const reservations: Reservation[] = [];
    for (let offset = 0; ; offset += 100) {
      const page = await list<Reservation>(`/commerce/reservations?limit=100&offset=${offset}`);
      reservations.push(...page);
      if (page.length < 100) break;
    }
    return Promise.all(reservations.filter(r => !!r.current?.terms.advance).map(async reservation => ({
      reservation,
      accounting: (await get<ReservationAccounting>(`/commerce/reservations/${encodeURIComponent(reservation.id)}/accounting`)).data,
    })));
  }, enabled && !!companyId);
  const q = useData(['billing', 'suppliers'], async () => {
    const orders = await list<Order>('/commerce/orders?limit=100');
    const per = await Promise.all(
      orders.map((o) =>
        list<Invoice>(`/commerce/orders/${o.id}/invoices`).then((is) =>
          is.map((i): SupplierRow => ({ ...i, order: o })),
        ),
      ),
    );
    return { orders, rows: per.flat() } satisfies SupplierData;
  }, enabled);
  const lines = useLinesLabel();
  const [query, setQuery] = useSearchQuery();
  const [state, setState] = useState('');
  const [open, setOpen] = useState<SupplierRow | null>(null);
  const [openAdvance, setOpenAdvance] = useState<Reservation | null>(null);
  const [openOrder, setOpenOrder] = useState<Order | null>(null);
  const all = q.data?.rows ?? [];
  const eligibleOrders = canIssue ? (q.data?.orders ?? []).filter((order) =>
    order.party === 'supplier' && ['accepted', 'fulfilling'].includes(order.status) &&
    !all.some((invoice) => invoice.order.id === order.id && invoice.status === 'issued'),
  ) : [];
  const late = (i: SupplierRow) => stateOf(i) === 'due' && i.schedule.some((p) => p.dueDate < today);
  const rows = all.filter(
    (i) => (!state || stateOf(i) === state) && matches(query, i.order.supplier.name, i.order.buyer.name, lines(i.order.terms.lines)),
  );
  return (
    <>
      <Panel title="Авансы и возвраты по резервам">
        {advances.isLoading && <p>Загрузка авансов…</p>}
        {advances.error && <p>Не удалось загрузить резервы. <Button onClick={() => void advances.refetch()}>Повторить</Button></p>}
        {!advances.isLoading && !advances.error && advances.data?.length === 0 && <p>Авансов по резервам пока нет.</p>}
        {(advances.data ?? []).map(({ reservation, accounting }) => {
          const pending = accounting.evidence.some(e => e.status === 'submitted');
          const label = accounting.refundStatus === 'confirmed' ? 'Возврат подтверждён'
            : accounting.refundStatus === 'requested' ? 'Возврат ожидает подтверждения'
              : pending ? 'Аванс на проверке' : accounting.creditOrderId ? 'Аванс зачтён в счёт'
                : accounting.accepted.amountMinor !== '0' ? 'Аванс принят' : 'Ожидается аванс';
          return <p key={reservation.id}>{reservation.buyer.name} → {reservation.supplier.name} · {money(accounting.agreedAdvance)} · {label}
            {accounting.refundStatus === 'requested' && ` · обязательство ${money(accounting.refundObligation)}`}
            {' '}<Button size="sm" onClick={() => setOpenAdvance(reservation)}>{accountingWorkspace && pending && accounting.allowedActions.includes('accept-payment') ? 'Проверить аванс' : 'Открыть расчёты'}</Button>
          </p>;
        })}
      </Panel>
      <Summary rows={all} overdue={all.filter(late).map((i) => i.outstanding)} />
      {eligibleOrders.length > 0 && <Panel title="B2B-счета к выставлению">
        {eligibleOrders.map((order) => <p key={order.id}>Покупатель {order.buyer.name} · {lines(order.terms.lines)} <Button size="sm" onClick={() => setOpenOrder(order)}>Выставить счёт</Button></p>)}
      </Panel>}
      <Panel>
        <Toolbar query={query} onQuery={setQuery} placeholder="Контрагент или автомобиль" onReset={() => setState('')}>
          <FilterSelect
            value={state}
            onChange={setState}
            all="Все статусы"
            options={Object.entries(stateLabel).map(([k, [l]]) => [k, l])}
          />
        </Toolbar>
        <ResultMeta>
          {plural(rows.length, ['счёт', 'счёта', 'счетов'])} · счета и оплаты B2B
        </ResultMeta>
        <Table
          rows={rows}
          loading={q.isLoading}
          error={q.error}
          rowKey={(i) => i.id}
          onRowClick={setOpen}
          empty="B2B-счетов пока нет"
          columns={[
            {
              title: 'Счёт',
              render: (i) => (
                <Cell
                  main={`Счёт · ${i.order.party === 'buyer' ? i.order.supplier.name : i.order.buyer.name}`}
                  sub={i.schedule.length ? `до ${date(i.schedule[i.schedule.length - 1]!.dueDate)}` : undefined}
                />
              ),
            },
            { title: 'Заказ', render: (i) => lines(i.order.terms.lines) },
            { title: 'Контрагент', render: (i) => i.order.party === 'buyer' ? i.order.supplier.name : i.order.buyer.name },
            { title: 'Сумма', render: (i) => <Cell main={money(i.total)} sub={`остаток ${money(i.outstanding)}`} /> },
            {
              title: 'Статус',
              render: (i) => {
                const [l, t] = stateLabel[stateOf(i)];
                return <Badge tone={late(i) ? 'danger' : t}>{late(i) ? 'Просрочен' : l}</Badge>;
              },
            },
            {
              title: 'Действие',
              render: (i) => (
                <Button size="sm" variant={stateOf(i) === 'due' ? 'primary' : 'secondary'} onClick={() => setOpen(i)}>
                  {stateOf(i) === 'due' ? 'Добавить оплату' : 'Открыть'}
                </Button>
              ),
            },
          ]}
        />
      </Panel>
      {open && (
        <Modal
          title={`Счёт · ${open.order.party === 'buyer' ? open.order.supplier.name : open.order.buyer.name}`}
          icon="wallet"
          size="wide"
          onClose={() => setOpen(null)}
          help={lines(open.order.terms.lines)}
        >
          <InvoiceById id={open.id} orderId={open.order.id} party={open.order.party} />
        </Modal>
      )}
      {openAdvance && <Modal title="Аванс резерва" icon="wallet" size="wide" onClose={() => setOpenAdvance(null)}>
        <ReservationPayment reservation={(advances.data ?? []).find(row => row.reservation.id === openAdvance.id)?.reservation ?? openAdvance} />
      </Modal>}
      {openOrder && <OrderDialog id={openOrder.id} onClose={() => setOpenOrder(null)} />}
    </>
  );
}

/** Re-reads the invoice so the dialog shows fresh payments after each action. */
function InvoiceById({ id, orderId, party }: { id: string; orderId: string; party: Order['party'] }) {
  const q = useData(['order-invoices', orderId], () => list<Invoice>(`/commerce/orders/${orderId}/invoices`));
  const i = q.data?.find((x) => x.id === id);
  return i ? (
    <InvoicePanel invoice={i} party={party} refresh={[['order-invoices', orderId], ['billing']]} />
  ) : (
    <p className="cell-sub">Загрузка…</p>
  );
}

type ClientRow = RetailInvoice & { deal: Deal };
type ClientData = { deals: Deal[]; rows: ClientRow[] };

function ClientInvoices({ enabled }: { enabled: boolean }) {
  const accountingWorkspace = useFinanceWorkspace();
  const q = useData(['billing', 'clients'], async () => {
    const deals = await list<Deal>('/retail/deals?limit=100');
    const full = await Promise.all(deals.map((d) => get<Deal>(`/retail/deals/${d.id}`).then((r) => r.data)));
    return { deals: full, rows: full.flatMap((d) => (d.invoices ?? []).map((i): ClientRow => ({ ...i, deal: d }))) } satisfies ClientData;
  }, enabled);
  const [query, setQuery] = useSearchQuery();
  const [state, setState] = useState('');
  const [open, setOpen] = useState<ClientRow | null>(null);
  const [openDeal, setOpenDeal] = useState<Deal | null>(null);
  const [review, setReview] = useState<{ payment: InstallmentPaymentGroup; deal: Deal } | null>(null);
  const all = (q.data?.rows ?? []).map((i) => ({ ...i, total: i.amount }));
  const waiting = (q.data?.deals ?? []).filter((d) => d.allowedActions.includes('issue-invoice') || d.allowedActions.includes('issue-installment-invoices'));
  const submitted = new Map<string, { payment: InstallmentPaymentGroup; deal: Deal }>();
  for (const deal of q.data?.deals ?? []) for (const invoice of deal.invoices ?? []) for (const payment of invoice.paymentGroups ?? []) {
    if (payment.status === 'submitted' && (payment.allowedActions.includes('accept') || payment.allowedActions.includes('reject'))) submitted.set(payment.id, { payment, deal });
  }
  const late = (i: ClientRow) => i.purpose !== 'monthly-installment' && stateOf(i) === 'due' && !!i.dueDate && i.dueDate < today;
  const rows = all.filter(
    (i) =>
      (!state || stateOf(i) === state) &&
      matches(query, i.deal.customer.displayName, i.recipientSnapshot, purposeLabel[i.purpose]),
  );
  return (
    <>
      <Summary rows={all} overdue={all.filter(late).map((i) => i.outstanding)} />
      {(waiting.length > 0 || submitted.size > 0) && <Panel title={accountingWorkspace ? 'Очередь бухгалтера' : 'История счетов и оплат'}>
        {waiting.map((deal) => <p key={deal.id}>{accountingWorkspace ? `Сделка ${deal.customer.displayName} ждёт выставления счёта` : `Сделка ${deal.customer.displayName}: Выставление счетов — в кабинете бухгалтера`} <Button size="sm" onClick={() => setOpenDeal(deal)}>{accountingWorkspace ? 'Открыть сделку' : 'Просмотреть сделку'}</Button></p>)}
        {[...submitted.values()].map(({ payment, deal }) => <p key={payment.id}>Чек {payment.externalReference} · {deal.customer.displayName} <Button size="sm" onClick={() => setReview({ payment, deal })}>{accountingWorkspace ? 'Проверить весь чек' : 'Просмотреть чек'}</Button></p>)}
      </Panel>}
      <Panel>
        <Toolbar query={query} onQuery={setQuery} placeholder="Клиент или назначение" onReset={() => setState('')}>
          <FilterSelect
            value={state}
            onChange={setState}
            all="Все статусы"
            options={Object.entries(stateLabel).map(([k, [l]]) => [k, l])}
          />
        </Toolbar>
        <ResultMeta>
          {plural(rows.length, ['счёт', 'счёта', 'счетов'])} · платежи клиента по Исламской линии сюда не попадают
        </ResultMeta>
        <Table
          rows={rows}
          loading={q.isLoading}
          error={q.error}
          rowKey={(i) => i.id}
          onRowClick={setOpen}
          empty="Счетов клиентам пока нет"
          columns={[
            {
              title: 'Счёт',
              render: (i) => (
                <Cell
                  main={`${purposeLabel[i.purpose] ?? i.purpose}${i.installmentNumber ? ` ${i.installmentNumber}` : ''}`}
                  sub={i.dueDate ? `до ${date(i.dueDate)}` : undefined}
                />
              ),
            },
            { title: 'Клиент', render: (i) => <Cell main={i.deal.customer.displayName} sub={i.recipientSnapshot} /> },
            { title: 'Сумма', render: (i) => <Cell main={money(i.amount)} sub={`остаток ${money(i.outstanding)}`} /> },
            {
              title: 'Статус',
              render: (i) => {
                const [l, t] = stateLabel[stateOf(i)];
                return <Badge tone={late(i) ? 'danger' : t}>{late(i) ? 'Просрочен' : l}</Badge>;
              },
            },
            {
              title: 'Действие',
              render: (i) => (
                <Button size="sm" onClick={() => setOpen(i)}>
                  Открыть
                </Button>
              ),
            },
          ]}
        />
      </Panel>
      {open && (open.purpose === 'monthly-installment' ? <DealDialog id={open.deal.id} initialInvoiceId={open.id} onClose={() => setOpen(null)} /> : (
        <Modal
          title={`${purposeLabel[open.purpose] ?? 'Счёт'} · ${open.deal.customer.displayName}`}
          icon="wallet"
          size="wide"
          onClose={() => setOpen(null)}
        >
          <RetailInvoiceById dealId={open.deal.id} id={open.id} />
        </Modal>
      ))}
      {openDeal && <DealDialog id={openDeal.id} onClose={() => setOpenDeal(null)} />}
      {review && <Modal title={`Чек · ${review.deal.customer.displayName}`} icon="wallet" size="wide" onClose={() => setReview(null)}>
        <InstallmentPaymentReview payment={review.payment} onBack={() => setReview(null)} onDecided={async () => { await q.refetch(); }} />
      </Modal>}
    </>
  );
}

function RetailInvoiceById({ dealId, id }: { dealId: string; id: string }) {
  const q = useData(['deal', dealId], () => get<Deal>(`/retail/deals/${dealId}`));
  const i = q.data?.data.invoices?.find((x) => x.id === id);
  return i ? (
    <RetailInvoicePanel invoice={i} refresh={[['deal', dealId], ['billing']]} />
  ) : (
    <p className="cell-sub">Загрузка…</p>
  );
}
