import { useEffect, useRef, useState } from 'react';
import type { RefObject } from 'react';
import { useQueryClient } from '@tanstack/react-query';
import {
  ActionButton,
  ApiError,
  Badge,
  Button,
  Details,
  Modal,
  Notice,
  Panel,
  Table,
  Tabs,
  date,
  dateTime,
  errorText,
  fileUrl,
  get,
  money,
  post,
  useData,
  useSession,
} from '@justixauto/kit';
import type { FieldSpec } from '@justixauto/kit';
import {
  useVehicleLabel,
  stageOrder,
  stageLabel,
  channelLabel,
  sourceLabel,
  purposeLabel,
  schemeLabel,
  dealLabel,
} from '../data';
import type { Deal, InstallmentPlan, InstallmentPaymentGroup, RetailInvoice, Lead } from '../data';
import { InstallmentEditor, InstallmentSection } from '../installment-servicing';
import { InstallmentPaymentForm, eligibleInstallmentInvoices } from '../installment-payment-form';
import { InstallmentPaymentReview } from '../installment-payment-review';
import { InstallmentPreview } from '../installment-terms-form';
import { RetailActionButton, RetailActionContext, RetailActionForm } from '../retail-actions';
import type { RetailAction } from '../retail-actions';
import { vehicleColorsLabel } from '../vehicle-colors';

const reason: FieldSpec[] = [{ name: 'reason', label: 'Причина', type: 'textarea', required: true }];

// ---------------- listings ----------------

// ---------------- CRM ----------------

const eventLabel: Record<string, string> = {
  'lead.created': 'Лид создан',
  'lead.assigned': 'Назначен ответственный',
  'lead.stage_changed': 'Смена этапа',
  'lead.won': 'Продажа',
  'deal.reserved': 'Сделка создана, автомобиль зарезервирован',
  'deal.contract_recorded': 'Договор подписан',
  'deal.invoice_issued': 'Выставлен счёт',
  'deal.payment_submitted': 'Внесена оплата',
  'deal.payment_accepted': 'Оплата принята',
  'deal.payment_rejected': 'Оплата отклонена',
  'deal.registered': 'Регистрация',
  'deal.delivered': 'Автомобиль выдан',
  'deal.cancelled': 'Сделка отменена',
};
function LeadDialogFooter({
  l,
  id,
  refresh,
  currentUserId,
}: {
  l: Lead;
  id: string;
  refresh: unknown[][];
  currentUserId: string;
}) {
  const next = stageOrder[stageOrder.indexOf(l.stage) + 1];
  return (
    <>
      <ActionButton
        label="Контакт"
        refresh={refresh}
        fields={[
          {
            name: 'channel',
            label: 'Канал',
            type: 'select',
            required: true,
            options: Object.entries(channelLabel),
          },
          { name: 'note', label: 'Итог', type: 'textarea', required: true },
        ]}
        onSubmit={(v) => post(`/retail/leads/${id}/contacts`, v)}
      />
      {l.assignedUserId !== currentUserId && (
        <ActionButton
          label="Взять себе"
          refresh={refresh}
          onSubmit={() =>
            post(`/retail/leads/${id}/assign`, { assignedUserId: currentUserId }, { ifMatch: l.revision })
          }
        />
      )}
      {next && (
        <ActionButton
          label={`→ ${stageLabel[next]}`}
          variant="primary"
          refresh={refresh}
          onSubmit={() => post(`/retail/leads/${id}/stage`, { stage: next }, { ifMatch: l.revision })}
        />
      )}
      <ActionButton
        label="Потерян"
        variant="danger"
        fields={reason}
        refresh={refresh}
        onSubmit={(v) =>
          post(`/retail/leads/${id}/stage`, { stage: 'lost', reason: v.reason }, { ifMatch: l.revision })
        }
      />
    </>
  );
}

function LeadDialogBody({ l, currentUserId }: { l: Lead; currentUserId: string }) {
  return (
    <>
      <Details
        items={[
          ['Этап', stageLabel[l.stage]],
          ['Источник', sourceLabel[l.source] ?? l.source],
          ['Телефон', l.customer?.phone || '—'],
          ['Ответственный', l.assignedUserId === currentUserId ? 'Я' : l.assignedUserId ? 'Назначен' : 'Не назначен'],
          ['Причина потери', l.lostReason || '—'],
          ['Сделка', l.dealId ? 'Создана' : '—'],
        ]}
      />
      <Panel title="Контакты" padded>
        <ul className="kit-timeline">
          {(l.contacts ?? []).map((c, i) => (
            <li key={i}>
              {dateTime(c.occurredAt)} — {channelLabel[c.channel] ?? c.channel}: {c.note}
            </li>
          ))}
        </ul>
      </Panel>
      <Panel title="История" padded>
        <ul className="kit-timeline">
          {(l.history ?? []).map((h, i) => (
            <li key={i}>
              {dateTime(h.occurredAt)} — {eventLabel[h.type] ?? h.type}
              {h.reason ? ` · ${h.reason}` : ''}
            </li>
          ))}
        </ul>
      </Panel>
    </>
  );
}

export function LeadDialog({ id, onClose }: { id: string; onClose: () => void }) {
  const q = useData(['lead', id], () => get<Lead>(`/retail/leads/${id}`));
  const s = useSession();
  const l = q.data?.data;
  const refresh = [['lead', id], ['leads']];
  const open = l && l.stage !== 'won' && l.stage !== 'lost';
  return (
    <Modal
      title={l ? `Лид: ${l.customer?.displayName ?? ''}` : 'Лид'}
      onClose={onClose}
      size="wide"
      footer={l && open && <LeadDialogFooter l={l} id={id} refresh={refresh} currentUserId={s.view.user.id} />}
    >
      {l && <LeadDialogBody l={l} currentUserId={s.view.user.id} />}
    </Modal>
  );
}

// ---------------- deals ----------------

const purposes: Record<string, string[]> = {
  cash: ['vehicle-payment', 'registration'],
  'own-installment': ['first-installment', 'registration'],
  'partner-finance': ['registration'],
};

const check = (ok: boolean | undefined) => (ok === undefined ? '—' : ok ? '✓' : '✗');

function DeliveryAction({ d, refresh }: { d: Deal; refresh: unknown[][] }) {
  if (!d.allowedActions.includes('deliver')) return null;
  return <RetailActionButton
    label="Выдать автомобиль"
    variant="primary"
    refresh={refresh}
    fields={[{ name: 'occurredAt', label: 'Когда выдан', type: 'datetime', required: true }]}
    intro={<p>Автомобиль будет списан со склада. Действие необратимо.</p>}
    onSubmit={(v) => post(`/retail/deals/${d.id}/deliveries`, v, { ifMatch: d.revision })}
  />;
}

function DeliveryReadiness({ d, targetRef }: { d: Deal; targetRef?: RefObject<HTMLDivElement | null> }) {
  const c = d.checklist;
  if (!c || d.status !== 'reserved') return null;
  const registrationOptional = c.registrationOptional ?? (d.paymentScheme === 'cash');
  return <div ref={targetRef} tabIndex={-1} aria-label="Готовность к выдаче">
    <Panel title="Готовность к выдаче" padded>
      <ul className="kit-timeline">
        <li>{check(c.contract)} Договор</li>
        {c.vehiclePayment !== undefined && <li>{check(c.vehiclePayment)} Оплата автомобиля</li>}
        {c.firstInstallment !== undefined && <li>{check(c.firstInstallment)} Первый взнос</li>}
        {c.insuranceApproved !== undefined && <li>{check(c.insuranceApproved)} Одобрение страховой</li>}
        <li>{check(c.registrationPaid)} Оплата регистрации{registrationOptional ? ' (необязательно)' : ''}</li>
        <li>{check(c.registered)} Регистрация{registrationOptional ? ' (необязательно)' : ''}</li>
        {!c.policyResolved && <li>✗ Правила выдачи при финансировании банком/МФО ещё не утверждены (OD-01)</li>}
      </ul>
    </Panel>
  </div>;
}

function DealDialogActions({ d, id, refresh }: { d: Deal; id: string; refresh: unknown[][] }) {
  const can = (a: string) => d.allowedActions.includes(a);
  return (
    <>
      {can('record-contract') && !d.contractSignedOn && (
        <RetailActionButton
          label="Договор подписан"
          refresh={refresh}
          intro={d.installmentDraft && !d.installmentPlan ? <><p>Условия для подписания договора:</p><InstallmentPreview draft={d.installmentDraft} /></> : undefined}
          fields={[
            { name: 'signedOn', label: 'Дата подписания', type: 'date', required: true },
            { name: 'reference', label: 'Номер договора', type: 'text', required: true },
            { name: 'file', label: 'Скан договора', type: 'file', purpose: 'deal-document' },
          ]}
          onSubmit={(v) =>
            post(
              `/retail/deals/${id}/contract-records`,
              { signedOn: v.signedOn, reference: v.reference, bindingIds: v.file ? [v.file] : [] },
              { ifMatch: d.revision },
            )
          }
        />
      )}
      {can('issue-invoice') && d.installmentDraft && !d.installmentPlan && !(d.invoices ?? []).some(i => i.purpose === 'first-installment' && i.status === 'issued') && (
        <RetailActionButton label="Выставить счёт первого взноса" refresh={refresh}
          intro={<p>Первый взнос по условиям: {money(d.installmentDraft.downPayment)}. Сумма и валюта зафиксированы.</p>}
          fields={[
            { name: 'recipientSnapshot', label: 'Плательщик (как в счёте)', type: 'text', required: true, initial: d.customer.displayName },
            { name: 'dueDate', label: 'Оплатить до', type: 'date', required: true },
          ]}
          onSubmit={v => post(`/retail/deals/${id}/invoices`, { ...v, purpose: 'first-installment', amount: d.installmentDraft!.downPayment }, { ifMatch: d.revision })} />
      )}
      {can('issue-invoice') && d.paymentScheme === 'cash' && !(d.invoices ?? []).some(i => i.purpose === 'vehicle-payment' && i.status === 'issued') && (
        <RetailActionButton label="Выставить счёт за автомобиль" refresh={refresh}
          intro={<p>Стоимость автомобиля: {money(d.price)}. Сумма и валюта зафиксированы.</p>}
          fields={[
            { name: 'recipientSnapshot', label: 'Плательщик (как в счёте)', type: 'text', required: true, initial: d.customer.displayName },
            { name: 'dueDate', label: 'Оплатить до', type: 'date', required: true },
          ]}
          onSubmit={v => post(`/retail/deals/${id}/invoices`, { recipientSnapshot: v.recipientSnapshot, dueDate: v.dueDate, purpose: 'vehicle-payment', amount: d.price }, { ifMatch: d.revision })} />
      )}
      {can('issue-invoice') && (
        <RetailActionButton
          label="Выставить счёт"
          refresh={refresh}
          fields={[
            {
              name: 'purpose',
              label: 'Назначение',
              type: 'select',
              required: true,
              options: (purposes[d.paymentScheme] ?? []).filter(p =>
                (p !== 'first-installment' || (!d.installmentDraft && !d.installmentPlan)) &&
                (d.paymentScheme !== 'cash' || p !== 'vehicle-payment'),
              ).map((p) => [p, purposeLabel[p] ?? p]),
            },
            { name: 'amount', label: 'Сумма', type: 'money', required: true, currency: d.price.currency },
            {
              name: 'recipientSnapshot',
              label: 'Плательщик (как в счёте)',
              type: 'text',
              required: true,
              initial: d.customer.displayName,
            },
            { name: 'dueDate', label: 'Оплатить до', type: 'date', required: true },
          ]}
          onSubmit={(v) => post(`/retail/deals/${id}/invoices`, v, { ifMatch: d.revision })}
        />
      )}
      {can('record-registration') && !d.registeredOn && (
        <RetailActionButton
          label="Регистрация"
          refresh={refresh}
          fields={[
            { name: 'registeredOn', label: 'Дата регистрации', type: 'date', required: true },
            { name: 'plateNumber', label: 'Госномер', type: 'text', required: true },
            { name: 'reference', label: 'Номер свидетельства', type: 'text', required: true },
          ]}
          onSubmit={(v) => post(`/retail/deals/${id}/registration`, v, { ifMatch: d.revision })}
        />
      )}
      {can('deliver') && <DeliveryAction d={d} refresh={refresh} />}
      {can('cancel') && (
        <RetailActionButton
          label="Отменить сделку"
          variant="danger"
          fields={reason}
          refresh={refresh}
          onSubmit={(v) => post(`/retail/deals/${id}/cancel`, v, { ifMatch: d.revision })}
        />
      )}
    </>
  );
}

function DealDialogBody({
  d,
  label,
  readinessTarget,
}: {
  d: Deal;
  label: (vehicleId: string) => string;
  readinessTarget?: RefObject<HTMLDivElement | null>;
}) {
  return (
    <>
      <Details
        items={[
          ['Автомобиль', d.vehicleSnapshot ? d.vehicleSnapshot.vin : label(d.vehicleId)],
          ['Фактические цвета', vehicleColorsLabel(d.vehicleSnapshot ?? {})],
          ['Схема', schemeLabel[d.paymentScheme]],
          ['Цена', money(d.price)],
          ['Статус', dealLabel[d.status]],
          [
            'Договор',
            d.contractSignedOn ? (
              <>
                {d.contractReference} от {date(d.contractSignedOn)}
                {d.contractFileIds.map((f, i) => (
                  <span key={f}>
                    {' '}
                    ·{' '}
                    <a href={fileUrl(f)} target="_blank" rel="noreferrer">
                      скан {i + 1}
                    </a>
                  </span>
                ))}
              </>
            ) : (
              '—'
            ),
          ],
          [
            'Регистрация',
            d.registeredOn ? `${d.plateNumber} (${d.registrationReference}) от ${date(d.registeredOn)}` : '—',
          ],
          ['Выдан', dateTime(d.deliveredAt)],
          ['Причина отмены', d.statusReason || '—'],
        ]}
      />
      <DeliveryReadiness d={d} targetRef={readinessTarget} />
    </>
  );
}

export function DealDialog({ id, onClose, initialInvoiceId }: { id: string; onClose: () => void; initialInvoiceId?: string }) {
  const q = useData(['deal', id], () => get<Deal>(`/retail/deals/${id}`));
  const client = useQueryClient();
  const label = useVehicleLabel();
  const d = q.data?.data;
  const refresh = [['deal', id], ['deals'], ['billing'], ['vehicles'], ['leads']];
  const [operation, setOperation] = useState<RetailAction | null>(null);
  const [editing, setEditing] = useState(false);
  const [tab, setTab] = useState<'deal' | 'payments' | 'history'>('deal');
  const editorBusy = useRef(false);
  const [selected, setSelected] = useState(initialInvoiceId ?? null);
  const [paymentView, setPaymentView] = useState<'details' | 'form' | 'review'>(initialInvoiceId ? 'details' : 'details');
  const [selectedPayment, setSelectedPayment] = useState<InstallmentPaymentGroup | null>(null);
  const [paymentPayoff, setPaymentPayoff] = useState(false);
  const [paymentBusy, setPaymentBusy] = useState(false);
  const [readBusy, setReadBusy] = useState(false);
  const readRunning = useRef(false);
  const [paymentReadFailed, setPaymentReadFailed] = useState(false);
  const [paymentSaved, setPaymentSaved] = useState(false);
  const [paymentOrigin, setPaymentOrigin] = useState<'schedule' | 'billing' | null>(initialInvoiceId ? 'billing' : null);
  const [notice, setNotice] = useState('');
  const [readinessIntent, setReadinessIntent] = useState(false);
  const scheduleScroll = useRef(0);
  const scheduleInvoiceId = useRef<string | null>(null);
  const schedulePlanAction = useRef<string | null>(null);
  const restoreSchedule = useRef(false);
  const readinessTarget = useRef<HTMLDivElement>(null);
  const detailsTarget = useRef<HTMLDivElement>(null);
  const row = d?.installmentPlan?.rows.find(r => r.invoiceId === selected);
  const invoice = d?.invoices?.find(i => i.id === selected);
  useEffect(() => {
    if (!restoreSchedule.current || selected || paymentView !== 'details' || paymentOrigin !== 'schedule' || readinessIntent || tab !== 'payments' || operation || editing || q.error || paymentReadFailed || readBusy) return;
    const trigger = Array.from(document.querySelectorAll<HTMLButtonElement>('[data-installment-invoice], [data-installment-plan-payment]'))
      .find(button => scheduleInvoiceId.current ? button.dataset.installmentInvoice === scheduleInvoiceId.current : button.dataset.installmentPlanPayment === schedulePlanAction.current);
    if (!trigger) return;
    const body = trigger.closest<HTMLElement>('.modal-body');
    const restore = window.setTimeout(() => {
      if (body) body.scrollTop = scheduleScroll.current;
      trigger.focus({ preventScroll: true });
      restoreSchedule.current = false;
    });
    return () => window.clearTimeout(restore);
  }, [selected, paymentOrigin, paymentView, readinessIntent, tab, operation, editing, q.error, paymentReadFailed, readBusy, d]);
  useEffect(() => {
    if (selected || !readinessIntent || tab !== 'deal') return;
    const target = readinessTarget.current ?? detailsTarget.current;
    if (!target) return;
    target.scrollIntoView?.({ block: 'start' });
    target.focus({ preventScroll: true });
    setReadinessIntent(false);
  }, [selected, readinessIntent, tab, d, readBusy]);
  function openPayment(invoiceId: string, trigger: HTMLButtonElement) {
    if (paymentReadFailed || readRunning.current) return;
    setPaymentSaved(false);
    restoreSchedule.current = true;
    scheduleInvoiceId.current = invoiceId;
    scheduleScroll.current = trigger.closest<HTMLElement>('.modal-body')?.scrollTop ?? 0;
    setPaymentOrigin('schedule');
    setPaymentView('details');
    setPaymentPayoff(false);
    setSelectedPayment(null);
    setSelected(invoiceId);
  }
  function openPlanPayment(payoff: boolean, trigger: HTMLButtonElement) {
    if (paymentReadFailed || readRunning.current || !d?.installmentPlan || !eligibleInstallmentInvoices(d.id, d.installmentPlan, d.invoices ?? []).length) return;
    setPaymentSaved(false);
    restoreSchedule.current = true;
    scheduleInvoiceId.current = null;
    schedulePlanAction.current = payoff ? 'payoff' : 'manual';
    scheduleScroll.current = trigger.closest<HTMLElement>('.modal-body')?.scrollTop ?? 0;
    setPaymentOrigin('schedule'); setPaymentView('form'); setPaymentPayoff(payoff); setSelectedPayment(null); setSelected(null);
  }
  function closePayment() {
    if (paymentBusy || readRunning.current) return;
    if (paymentOrigin === 'billing') onClose();
    else { setTab('payments'); setSelected(null); setPaymentView('details'); setSelectedPayment(null); setPaymentPayoff(false); }
  }
  function closeDeal() {
    if (editorBusy.current || paymentBusy || readRunning.current) return;
    if (editing) setEditing(false);
    else onClose();
  }
  function openReadiness() {
    restoreSchedule.current = false;
    setTab('deal');
    setReadinessIntent(true);
    setPaymentOrigin(null);
    setSelected(null);
  }
  async function refreshPaymentAfter() {
    readRunning.current = true; setReadBusy(true);
    try {
      const results = await Promise.allSettled(refresh.map(queryKey => client.invalidateQueries({ queryKey, refetchType: 'all' }, { throwOnError: true })));
      const failed = results.some(result => result.status === 'rejected');
      setPaymentReadFailed(failed);
      if (failed) throw new Error('refresh failed');
    } finally { readRunning.current = false; setReadBusy(false); }
  }
  async function retryPaymentRead() {
    if (readRunning.current || paymentBusy) return;
    try { await refreshPaymentAfter(); } catch { /* The failed read stays visible; no mutation is repeated. */ }
  }
  async function refreshAfter(message: string, keys = refresh) {
    const results = await Promise.allSettled(keys.map(queryKey => client.invalidateQueries({ queryKey }, { throwOnError: true })));
    setNotice(message + (results.some(result => result.status === 'rejected') ? ' Не удалось обновить данные. Повторите загрузку сделки; операция уже выполнена.' : ''));
  }
  const retry = () => void retryPaymentRead();
  const currentPayment = selectedPayment && (d?.installmentPlan?.payments?.find(payment => payment.id === selectedPayment.id)
    ?? d?.invoices?.flatMap(invoice => invoice.paymentGroups ?? []).find(payment => payment.id === selectedPayment.id));
  if (operation) return <RetailActionForm action={operation}
    context={`${d?.customer.displayName ?? 'Сделка'}${row ? ` · Платёж рассрочки ${row.number} от ${date(row.dueDate)}` : ''}`}
    backLabel={selected ? 'Назад к платежу' : undefined}
    onClose={() => setOperation(null)} onDone={() => refreshAfter(`${operation.label}: выполнено.`, operation.refresh ?? refresh)} />;
  if (selected || paymentView !== 'details') return <RetailActionContext.Provider value={action => { setNotice(''); setOperation(action); }}>
    <PaymentDialog deal={d} row={row} invoice={invoice} refresh={refresh} notice={notice} planState={d?.installmentPlan?.state} error={q.error ? errorText(q.error) : ''}
      loading={!d && !q.error} view={paymentView} payment={currentPayment ?? (paymentReadFailed ? selectedPayment : null)} readFailed={paymentReadFailed} saved={paymentSaved} busy={paymentBusy || readBusy}
      onBusyChange={setPaymentBusy} onClose={closePayment} onBack={() => { if (!paymentBusy && !readRunning.current) { if (paymentView === 'details') { setTab('payments'); setPaymentOrigin('schedule'); setSelected(null); } setPaymentView('details'); setSelectedPayment(null); setPaymentPayoff(false); } }}
      onOpenForm={() => { setPaymentSaved(false); setPaymentPayoff(false); setPaymentView('form'); }} onOpenReview={payment => { setPaymentSaved(false); setPaymentPayoff(false); setSelectedPayment(payment); setPaymentView('review'); }}
      initialFullRemaining={paymentPayoff} onSubmitted={async () => { setPaymentSaved(true); await refreshPaymentAfter(); }} onPreparation={openReadiness} onRetry={retryPaymentRead} />
  </RetailActionContext.Provider>;
  return <RetailActionContext.Provider value={action => { setNotice(''); setOperation(action); }}><div className="sale-dialog">
    <Modal
      title={d ? `Сделка: ${d.customer.displayName}` : 'Сделка'}
      onClose={closeDeal}
      size="wide"
      footer={<>
        <Button disabled={readBusy} onClick={closeDeal}>Закрыть</Button>
        {d && !editing && !q.error && !paymentReadFailed && !readBusy && <DealDialogActions d={d} id={id} refresh={refresh} />}
      </>}
    >
      {notice && <Notice>{notice}</Notice>}
      {editing && d ? <InstallmentEditor deal={d} onBusy={busy => { editorBusy.current = busy; }}
        onCancel={() => setEditing(false)} onSaved={async () => { await refreshAfter('Условия сохранены.'); setEditing(false); }} /> : <>
        {(q.error || paymentReadFailed) && <Notice kind="danger">Не удалось загрузить данные сделки. {q.error ? errorText(q.error) : ''} <Button disabled={readBusy} onClick={retry}>Повторить загрузку сделки</Button></Notice>}
        {!d && !q.error && <p role="status">Загрузка сделки…</p>}
        {d && !q.error && !paymentReadFailed && !readBusy && <>
          <Button disabled={q.isFetching} onClick={retry}>Обновить данные сделки</Button>
          <Tabs value={tab} onChange={next => { setReadinessIntent(false); restoreSchedule.current = false; setTab(next); }} tabs={[
            ['deal', 'Сделка и выдача'], ['payments', 'Оплаты'], ['history', 'История'],
          ]} />
          <div className="sale-tab-content" role="tabpanel" aria-label={{ deal: 'Сделка и выдача', payments: 'Оплаты', history: 'История' }[tab]}>
            {tab === 'deal' && <div ref={detailsTarget} tabIndex={-1} aria-label="Детали сделки"><DealDialogBody d={d} label={label} readinessTarget={readinessTarget} /></div>}
            {tab === 'payments' && <>
              {d.paymentScheme === 'own-installment' && <InstallmentSection deal={d} onCreate={() => { setNotice(''); setEditing(true); }} onSelect={openPayment} onPayment={openPlanPayment} onRetry={retry} />}
              {!(d.invoices ?? []).length && <p>Счета пока не выставлены.</p>}
              {(d.invoices ?? []).filter(i => i.purpose !== 'monthly-installment').map(i => <RetailInvoicePanel key={i.id} invoice={i} refresh={refresh} />)}
            </>}
            {tab === 'history' && <Panel title="История" padded>
              {!d.history?.length && <p>Событий пока нет.</p>}
              <ul className="kit-timeline">{(d.history ?? []).map((h, i) => <li key={i}>
                {dateTime(h.occurredAt)} — {eventLabel[h.type] ?? h.type}{h.reason ? ` · ${h.reason}` : ''}
              </li>)}</ul>
            </Panel>}
          </div>
        </>}
      </>}
    </Modal>
  </div></RetailActionContext.Provider>;
}

function PaymentDialog({ deal, row, invoice, refresh, notice, planState, error, loading, view, payment, readFailed, saved, busy, initialFullRemaining, onBusyChange, onClose, onBack, onOpenForm, onOpenReview, onSubmitted, onPreparation, onRetry }: {
  deal: Deal | undefined; row: InstallmentPlan['rows'][number] | undefined;
  invoice: RetailInvoice | undefined; refresh: unknown[][]; notice: string; planState: InstallmentPlan['state'] | undefined; error: string; loading: boolean;
  view: 'details' | 'form' | 'review'; payment: InstallmentPaymentGroup | null; readFailed: boolean; saved: boolean; busy: boolean; initialFullRemaining: boolean;
  onBusyChange: (busy: boolean) => void; onClose: () => void; onBack: () => void; onOpenForm: () => void; onOpenReview: (payment: InstallmentPaymentGroup) => void;
  onSubmitted: () => Promise<void>; onPreparation: () => void; onRetry: () => void;
}) {
  const title = view === 'form' ? 'Внести оплату по графику' : view === 'review' ? 'Проверка оплаты рассрочки' : row ? `Платёж рассрочки ${row.number} · ${date(row.dueDate)}` : 'Платёж рассрочки';
  const close = () => { if (!busy) onClose(); };
  const canSubmit = !!deal?.installmentPlan && !!invoice && eligibleInstallmentInvoices(deal.id, deal.installmentPlan, deal.invoices ?? []).some(value => value.invoice.id === invoice.id);
  const missingParent = invoice?.paymentEvidence.some(evidence => evidence.paymentGroupId && !invoice.paymentGroups?.some(group => group.id === evidence.paymentGroupId));
  return <div className="sale-dialog"><Modal title={title} onClose={close} size="wide" footer={<Button disabled={busy} onClick={close}>Закрыть</Button>}>
    {readFailed && <Notice kind="danger">{saved ? 'Операция сохранена. Не удалось обновить данные; повторите загрузку перед следующим действием.' : 'Не удалось загрузить сделку.'} {error} <Button disabled={busy} onClick={onRetry}>Повторить загрузку сделки</Button></Notice>}
    {view === 'form' && deal?.installmentPlan && <InstallmentPaymentForm dealId={deal.id} plan={deal.installmentPlan} invoices={deal.invoices ?? []}
      {...(invoice ? { selectedInvoiceId: invoice.id } : {})} initialFullRemaining={initialFullRemaining} blocked={busy || readFailed} onBack={onBack} onBusyChange={onBusyChange} onSubmitted={onSubmitted} />}
    {view === 'review' && payment && <InstallmentPaymentReview key={payment.id} payment={payment} blocked={busy || readFailed} onBack={onBack} onBusyChange={onBusyChange} onDecided={onSubmitted} />}
    {view === 'review' && !payment && <Notice kind="danger">Не удалось загрузить весь чек. <Button disabled={busy} onClick={onRetry}>Повторить загрузку сделки</Button></Notice>}
    {view !== 'details' || <>
    <section aria-label="Выбранный платёж рассрочки">
      <Button variant="back" disabled={busy} onClick={onBack}>Назад к графику</Button>
      {deal && row && <p>{deal.customer.displayName} · дата платежа {date(row.dueDate)}</p>}
      {notice && <Notice>{notice}</Notice>}
      {planState === 'cancelled' && <Notice>Сделка отменена. Платежи по графику недоступны.</Notice>}
      {error && !readFailed ? <Notice kind="danger">Не удалось загрузить сделку. {error} <Button disabled={busy} onClick={onRetry}>Повторить загрузку сделки</Button></Notice> :
        loading ? <p role="status">Загрузка сделки…</p> :
          row && invoice ? <>
            {planState === 'planned' && !readFailed && !busy && <>
              <DeliveryReadiness d={deal!} />
              {deal!.allowedActions.includes('deliver') ? <DeliveryAction d={deal!} refresh={refresh} /> : <Button onClick={onPreparation}>К готовности к выдаче</Button>}
            </>}
            {missingParent && !readFailed && <Notice kind="danger">Не удалось загрузить весь чек. <Button disabled={busy} onClick={onRetry}>Повторить загрузку сделки</Button></Notice>}
            {!readFailed && <RetailInvoicePanel invoice={invoice} refresh={refresh} canInstallmentPayment={canSubmit && !busy} onInstallmentPayment={onOpenForm} onReview={payment => { if (!busy) onOpenReview(payment); }} />}
          </> :
            !readFailed && <Notice kind="danger">Выбранный платёж не найден в графике. Обновите данные сделки. <Button disabled={busy} onClick={onRetry}>Повторить загрузку сделки</Button></Notice>}
    </section>
    </>}
  </Modal></div>;
}

export function RetailInvoicePanel({ invoice: i, refresh, canInstallmentPayment = false, onInstallmentPayment, onReview }: { invoice: RetailInvoice; refresh: unknown[][]; canInstallmentPayment?: boolean; onInstallmentPayment?: () => void; onReview?: (payment: InstallmentPaymentGroup) => void }) {
  const grouped = i.purpose === 'monthly-installment';
  return (
    <Panel
      title={`${purposeLabel[i.purpose] ?? i.purpose}${i.installmentNumber ? ` ${i.installmentNumber}` : ''}: ${money(i.amount)} · оплачено ${money(i.paid)} · остаток ${money(i.outstanding)}`}
      actions={
        i.allowedActions?.includes('submit-payment') && (grouped ? canInstallmentPayment && onInstallmentPayment && <Button onClick={onInstallmentPayment}>Внести оплату</Button> :
          <RetailActionButton
            label="Внести оплату"
            refresh={refresh}
            fields={[
              { name: 'claimedAmount', label: 'Сумма', type: 'money', required: true, currency: i.amount.currency },
              { name: 'paidOn', label: 'Дата оплаты', type: 'date', required: true },
              { name: 'externalReference', label: 'Номер платёжки / чека', type: 'text', required: true },
              { name: 'file', label: 'Подтверждение (файл)', type: 'file', purpose: 'payment-evidence' },
            ]}
            onSubmit={(v) => {
              const claimed = v.claimedAmount as { amountMinor: string; currency: string };
              if (claimed.currency !== i.amount.currency) throw new ApiError(422, 'validation', 'Валюта оплаты должна совпадать с валютой счёта.', { claimedAmount: `Выберите ${i.amount.currency}` });
              if (BigInt(claimed.amountMinor) <= 0n || (i.available && BigInt(claimed.amountMinor) > BigInt(i.available.amountMinor))) throw new ApiError(422, 'validation', 'Проверьте сумму оплаты.', { claimedAmount: 'Сумма должна быть положительной и не превышать доступную к внесению.' });
              if (!v.paidOn || !String(v.externalReference ?? '').trim()) throw new ApiError(422, 'validation', 'Укажите дату оплаты и номер платёжки / чека.');
              return post(`/retail/invoices/${i.id}/evidence`, {
                claimedAmount: v.claimedAmount,
                paidOn: v.paidOn,
                externalReference: v.externalReference,
                attachmentBindingIds: v.file ? [v.file] : [],
              });
            }}
          />)
      }
    >
      <p>На проверке: {money(i.pending)} · Доступно к внесению: {money(i.available)}</p>
      {grouped && i.paymentGroups?.map(payment => <p key={payment.id}>Чек {payment.externalReference}: <Button onClick={() => onReview?.(payment)}>Открыть весь чек</Button></p>)}
      {i.outstanding.amountMinor === '0' ? <p>Платёж полностью оплачен.</p> : i.available?.amountMinor === '0' && i.pending.amountMinor !== '0' ? <p>Вся оставшаяся сумма на проверке. Дождитесь подтверждения финансового сотрудника.</p> : null}
      <Table
        rows={i.paymentEvidence}
        rowKey={(e) => e.id}
        empty="Оплат пока нет"
        columns={[
          { title: 'Сумма', render: (e) => money(e.amount) },
          { title: 'Дата', render: (e) => e.paidOn },
          {
            title: 'Документ',
            render: (e) => (
              <>
                {e.externalReference}{' '}
                {e.attachmentIds.map((f) => (
                  <a key={f} href={fileUrl(f)} target="_blank" rel="noreferrer">
                    файл
                  </a>
                ))}
              </>
            ),
          },
          {
            title: 'Статус',
            render: (e) => (
              <Badge tone={e.status === 'accepted' ? 'success' : e.status === 'rejected' ? 'danger' : 'warning'}>
                {
                  (
                    {
                      submitted: 'На проверке',
                      accepted: 'Принято',
                      rejected: `Отклонено: ${e.decisionReason}`,
                    } as Record<string, string>
                  )[e.status]
                }
              </Badge>
            ),
          },
          {
            title: '',
            render: (e) =>
              !e.paymentGroupId && (e.allowedActions?.includes('accept') || e.allowedActions?.includes('reject')) && (
                <div className="kit-row">
                  {e.allowedActions.includes('accept') && <RetailActionButton
                    label="Принять"
                    variant="primary"
                    refresh={refresh}
                    fields={[{ name: 'confirmation', label: 'Деньги поступили', type: 'checkbox' }]}
                    onSubmit={(v) => {
                      if (v.confirmation !== true) throw new ApiError(422, 'validation', 'Подтвердите поступление денег.', { confirmation: 'Подтвердите поступление денег' });
                      return post(`/retail/evidence/${e.id}/accept`, v, { ifMatch: e.revision });
                    }}
                  />}
                  {e.allowedActions.includes('reject') && <RetailActionButton
                    label="Отклонить"
                    fields={reason}
                    refresh={refresh}
                    onSubmit={(v) => {
                      if (!String(v.reason ?? '').trim()) throw new ApiError(422, 'validation', 'Укажите причину отклонения.', { reason: 'Укажите причину' });
                      return post(`/retail/evidence/${e.id}/reject`, v, { ifMatch: e.revision });
                    }}
                  />}
                </div>
              ),
          },
        ]}
      />
    </Panel>
  );
}
