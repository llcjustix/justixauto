import { useRef, useState } from 'react';
import { Badge, Button, Notice, date, dateTime, errorText, fileUrl, money, post, toMinor, upload, useRefresh, useSession } from '@justixauto/kit';
import { useReservationAccounting } from './data';
import type { Reservation, ReservationPaymentEvidence } from './data';

type Action = 'submit' | 'accept' | 'reject' | 'request-refund' | 'confirm-refund';
type Draft = { amount: string; paidOn: string; reference: string; file: File | null; reason: string; confirmed: boolean };
const emptyDraft = (): Draft => ({ amount: '', paidOn: '', reference: '', file: null, reason: '', confirmed: false });
const evidenceStatus: Record<ReservationPaymentEvidence['status'], string> = {
  submitted: 'На проверке', accepted: 'Принято бухгалтером', rejected: 'Отклонено',
};

/** Shared money surface for the Purchases reservation and the Billing queue. */
export function ReservationPayment({ reservation }: { reservation: Reservation }) {
  const session = useSession();
  const refresh = useRefresh();
  const query = useReservationAccounting(reservation.id);
  const accounting = query.data;
  const [action, setAction] = useState<Action | null>(null);
  const [target, setTarget] = useState<ReservationPaymentEvidence | null>(null);
  const [draft, setDraft] = useState<Draft>(emptyDraft);
  const [error, setError] = useState('');
  const [busy, setBusy] = useState(false);
  const [reconcile, setReconcile] = useState(false);
  const sending = useRef(false);
  const companyId = session.company?.id ?? '';
  const buyer = companyId === reservation.buyer.id && session.can('commerce.trade');
  const supplierTrader = companyId === reservation.supplier.id && session.can('commerce.trade');
  const accountant = companyId === reservation.supplier.id && session.can('commerce.payments.accept');
  const can = (name: string) => !!accounting?.allowedActions.includes(name);
  const open = (next: Action, evidence: ReservationPaymentEvidence | null = null) => {
    setAction(next); setTarget(evidence); setDraft(emptyDraft()); setError('');
  };
  const close = () => { if (!busy && !reconcile) { setAction(null); setTarget(null); setError(''); } };
  const reload = async () => {
    await refresh(
      ['reservations', companyId], ['billing', 'reservations', companyId], ['billing', 'suppliers'],
      ['order-invoices'], ['order'], ['orders'],
    );
    const result = await query.refetch();
    if (result.error) throw result.error;
  };
  const retry = async () => {
    try { await reload(); setReconcile(false); setError(''); }
    catch (e) { setError(errorText(e)); }
  };
  async function submit() {
    if (sending.current || reconcile || !action || !accounting) return;
    const currentTarget = target && accounting.evidence.find(e => e.id === target.id);
    const available = action === 'submit' ? buyer && can('submit-payment')
      : action === 'request-refund' ? (buyer || supplierTrader) && can('request-refund')
        : action === 'confirm-refund' ? accountant && can('confirm-refund')
          : accountant && can(action === 'accept' ? 'accept-payment' : 'reject-payment') && currentTarget?.status === 'submitted';
    if (!available) { setError('Действие больше недоступно. Обновите данные.'); return; }
    if (action === 'submit') {
      const minor = toMinor(draft.amount);
      if (!minor || BigInt(minor) <= 0n || BigInt(minor) + BigInt(accounting.accepted.amountMinor) + BigInt(accounting.pending.amountMinor) > BigInt(accounting.agreedAdvance.amountMinor)) {
        setError('Введите сумму в пределах свободного согласованного аванса.'); return;
      }
    }
    if ((action === 'submit' || action === 'confirm-refund') && (!draft.paidOn || !draft.reference.trim())) {
      setError('Укажите дату и номер внешнего платёжного документа.'); return;
    }
    if ((action === 'reject' || action === 'request-refund') && !draft.reason.trim()) {
      setError('Укажите причину.'); return;
    }
    if ((action === 'accept' || action === 'confirm-refund') && !draft.confirmed) {
      setError(action === 'accept' ? 'Подтвердите поступление денег.' : 'Подтвердите совершённый внешний возврат.'); return;
    }
    sending.current = true; setBusy(true); setError('');
    try {
      const id = encodeURIComponent(reservation.id);
      const fileIds = draft.file ? [(await upload(draft.file, 'payment-evidence')).id] : [];
      const evidence = {
        claimedAmount: action === 'confirm-refund' ? accounting.refundObligation : { amountMinor: toMinor(draft.amount)!, currency: accounting.agreedAdvance.currency },
        paidOn: draft.paidOn, externalReference: draft.reference.trim(), attachmentBindingIds: fileIds,
      };
      if (action === 'submit') await post(`/commerce/reservations/${id}/payment-evidence`, evidence, { ifMatch: accounting.revision });
      if (action === 'request-refund') await post(`/commerce/reservations/${id}/refund-requests`, { reason: draft.reason.trim() }, { ifMatch: accounting.revision });
      if (action === 'confirm-refund') await post(`/commerce/reservations/${id}/refund-confirmations`, { ...evidence, confirmation: true }, { ifMatch: accounting.revision });
      if ((action === 'accept' || action === 'reject') && currentTarget) await post(
        `/commerce/reservation-payment-evidence/${encodeURIComponent(currentTarget.id)}/${action}`,
        { reservationRevision: accounting.revision, confirmation: action === 'accept', reason: draft.reason.trim() },
        { ifMatch: currentTarget.revision },
      );
      setReconcile(true);
      try { await reload(); setReconcile(false); setAction(null); setTarget(null); }
      catch (e) { setError(`Операция отправлена. Обновите данные перед следующим действием: ${errorText(e)}`); }
    } catch (e) {
      setError(errorText(e));
      // A stale or uncertain response must be reconciled before a second write.
      setReconcile(true);
      try { await reload(); setReconcile(false); }
      catch { /* Keep writes locked and expose the explicit read retry. */ }
    } finally { sending.current = false; setBusy(false); }
  }

  if (query.isLoading && !accounting) return <Notice>Загрузка расчётов по резерву…</Notice>;
  if (query.error && !accounting) return <Notice kind="danger">{errorText(query.error)} <Button onClick={() => void query.refetch()}>Повторить</Button></Notice>;
  if (!accounting) return null;
  const controlsDisabled = busy || reconcile;
  return <section aria-label="Учёт аванса резерва">
    <h3>Аванс и возврат</h3>
    <p>Согласованный аванс: {money(accounting.agreedAdvance)}{reservation.advanceDueAt ? ` · срок ${date(reservation.advanceDueAt)}` : ''}</p>
    <p>Принято: {money(accounting.accepted)} · на проверке: {money(accounting.pending)} · подтверждённый возврат: {money(accounting.refunded)}</p>
    {accounting.creditOrderId && <p>Аванс зачтён в заказ {accounting.creditOrderId}{accounting.creditInvoiceId ? ` и счёт ${accounting.creditInvoiceId}` : ''}. Оплата счёта уже включает этот зачёт.</p>}
    {accounting.refundStatus === 'requested' && <Notice kind="warning">Возврат запрошен: обязательство {money(accounting.refundObligation)}. Ожидается решение по чекам и подтверждение фактического возврата бухгалтером.</Notice>}
    {accounting.refundStatus === 'confirmed' && <Notice kind="success">Полный возврат подтверждён бухгалтером: {money(accounting.refunded)}.</Notice>}
    {accounting.refundStatus === 'none' && accounting.pending.amountMinor !== '0' && <Notice>Отправленный документ ожидает проверки. Деньги ещё не подтверждены.</Notice>}
    {accounting.refund && <section aria-label="История возврата">
      <h4>История возврата</h4>
      <p>Запросил: {accounting.refund.requestedBy} · {dateTime(accounting.refund.requestedAt)} · причина: {accounting.refund.requestReason}</p>
      {accounting.refund.status === 'confirmed' && <p>Фактический возврат: {accounting.refund.amount ? money(accounting.refund.amount) : '—'}{accounting.refund.refundedOn ? ` · дата ${date(accounting.refund.refundedOn)}` : ''}{accounting.refund.externalReference ? ` · документ ${accounting.refund.externalReference}` : ''}{accounting.refund.confirmedBy ? ` · подтвердил: ${accounting.refund.confirmedBy}` : ''}{accounting.refund.confirmedAt ? ` · ${dateTime(accounting.refund.confirmedAt)}` : ''}
        {accounting.refund.attachmentIds?.map(id => <a key={id} href={fileUrl(id)} target="_blank" rel="noreferrer"> Приватный файл</a>)}</p>}
    </section>}
    {error && <Notice kind="danger">{error} <Button onClick={() => void retry()}>Обновить данные</Button></Notice>}
    <div className="kit-row">
      {buyer && can('submit-payment') && <Button disabled={controlsDisabled} onClick={() => open('submit')}>Сообщить об авансе</Button>}
      {(buyer || supplierTrader) && can('request-refund') && <Button disabled={controlsDisabled} onClick={() => open('request-refund')}>Запросить возврат</Button>}
      {accountant && can('confirm-refund') && <Button disabled={controlsDisabled} onClick={() => open('confirm-refund')}>Подтвердить полный возврат</Button>}
    </div>
    <h4>История документов</h4>
    {accounting.evidence.length === 0 ? <p>Документов пока нет.</p> : <ul>{accounting.evidence.map(e => <li key={e.id}>
      {money(e.amount)} · {date(e.paidOn)} · {e.externalReference} · <Badge tone={e.status === 'accepted' ? 'success' : e.status === 'rejected' ? 'danger' : 'warning'}>{evidenceStatus[e.status]}</Badge>
      <span> · отправил: {e.submittedBy} · {dateTime(e.createdAt)}{e.decidedBy ? ` · решение: ${e.decidedBy}` : ''}{e.decidedAt ? ` · ${dateTime(e.decidedAt)}` : ''}{e.decisionReason ? ` · причина: ${e.decisionReason}` : ''}</span>
      {e.attachmentIds.map(id => <a key={id} href={fileUrl(id)} target="_blank" rel="noreferrer"> Приватный файл</a>)}
      {accountant && e.status === 'submitted' && can('accept-payment') && can('reject-payment') && <span className="kit-row">
        <Button disabled={controlsDisabled} onClick={() => open('accept', e)}>Принять аванс</Button>
        <Button disabled={controlsDisabled} onClick={() => open('reject', e)}>Отклонить аванс</Button>
      </span>}
    </li>)}</ul>}
    {action && <div className="kit-panel" aria-label="Действие с авансом">
      <h4>{action === 'submit' ? 'Сообщить об авансе' : action === 'accept' ? 'Принять внешний платёж' : action === 'reject' ? 'Отклонить документ' : action === 'request-refund' ? 'Запросить полный возврат' : 'Подтвердить полный возврат'}</h4>
      {action === 'submit' && <label>Сумма аванса ({accounting.agreedAdvance.currency})<input aria-label="Сумма аванса" inputMode="decimal" value={draft.amount} onChange={e => setDraft({ ...draft, amount: e.target.value })} /></label>}
      {action === 'confirm-refund' && <p>Полная сумма возврата: {money(accounting.refundObligation)}. Частичный возврат не предусмотрен.</p>}
      {(action === 'submit' || action === 'confirm-refund') && <>
        <label>Дата внешнего платежа<input aria-label="Дата внешнего платежа" type="date" value={draft.paidOn} onChange={e => setDraft({ ...draft, paidOn: e.target.value })} /></label>
        <label>Номер платёжного документа<input aria-label="Номер платёжного документа" value={draft.reference} onChange={e => setDraft({ ...draft, reference: e.target.value })} /></label>
        <label>Подтверждение (приватный файл)<input aria-label="Подтверждение" type="file" onChange={e => setDraft({ ...draft, file: e.target.files?.[0] ?? null })} /></label>
      </>}
      {(action === 'reject' || action === 'request-refund') && <label>Причина<textarea aria-label="Причина" value={draft.reason} onChange={e => setDraft({ ...draft, reason: e.target.value })} /></label>}
      {(action === 'accept' || action === 'confirm-refund') && <label><input type="checkbox" checked={draft.confirmed} onChange={e => setDraft({ ...draft, confirmed: e.target.checked })} />{action === 'accept' ? 'Деньги поступили на счёт поставщика' : 'Полный внешний возврат выполнен'}</label>}
      <div className="kit-row"><Button variant="primary" disabled={controlsDisabled} onClick={() => void submit()}>{busy ? 'Отправка…' : 'Подтвердить действие'}</Button><Button disabled={controlsDisabled} onClick={close}>Отмена</Button></div>
    </div>}
  </section>;
}
