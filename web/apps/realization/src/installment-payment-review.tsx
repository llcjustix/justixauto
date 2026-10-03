import { useEffect, useRef, useState } from 'react';
import { ApiError, Button, Notice, fileUrl, money, post } from '@justixauto/kit';
import type { InstallmentPaymentGroup } from './data';

export interface InstallmentPaymentReviewProps {
  payment: InstallmentPaymentGroup;
  onBack: () => void;
  /** Mirrors decision POST and host-refresh busy state for the enclosing Modal. */
  onBusyChange?: (busy: boolean) => void;
  /** Host refresh follows a successful mutation and may fail without reopening it. */
  onDecided: () => Promise<void>;
  blocked?: boolean;
}

export function InstallmentPaymentReview({ payment, onBack, onBusyChange, onDecided, blocked = false }: InstallmentPaymentReviewProps) {
  const [reason, setReason] = useState('');
  const [error, setError] = useState('');
  const [busy, setBusy] = useState(false);
  const [completed, setCompleted] = useState<'accept' | 'reject' | null>(null);
  const [confirmed, setConfirmed] = useState(false);
  const sending = useRef(false);
  useEffect(() => { if (completed && !blocked) setError(''); }, [completed, blocked]);
  const close = () => { if (!busy && !blocked) onBack(); };
  async function decide(action: 'accept' | 'reject') {
    if (sending.current || completed || blocked || payment.status !== 'submitted' || !payment.allowedActions?.includes(action) || (action === 'reject' ? !reason.trim() : !confirmed)) return;
    sending.current = true; setBusy(true); onBusyChange?.(true); setError('');
    try {
      await post(`/retail/installment-payments/${payment.id}/${action}`, action === 'accept' ? { confirmation: true } : { reason: reason.trim() }, { ifMatch: payment.revision });
      setCompleted(action);
      try { await onDecided(); }
      catch { setError('Не удалось обновить данные; повторите загрузку перед следующим действием.'); }
    } catch (value) { setError(value instanceof ApiError ? [value.message, ...Object.values(value.fields)].filter(Boolean).join(' · ') : String(value)); }
    finally { sending.current = false; setBusy(false); onBusyChange?.(false); }
  }
  return <section className="kit-stack" aria-label="Проверка оплаты рассрочки">
    <h3>Чек {payment.externalReference}</h3>
    <p>{payment.paidOn} · заявлено {money(payment.claimedAmount)}</p>
    <p>Статус: {completed === 'accept' ? 'Принято' : completed === 'reject' ? `Отклонено: ${reason.trim()}` : payment.status === 'submitted' ? 'На проверке' : payment.status === 'accepted' ? 'Принято' : `Отклонено: ${payment.decisionReason}`}</p>
    {completed && <Notice kind="success">Решение сохранено для всего чека.</Notice>}
    {error && <Notice kind="danger">{error}</Notice>}
    <table className="kit-lines"><thead><tr><th>Месяц</th><th>Сумма</th></tr></thead><tbody>{payment.allocations.map(allocation => <tr key={allocation.invoiceId}><td>{allocation.number}</td><td>{money(allocation.amount)}</td></tr>)}</tbody></table>
    {payment.attachmentIds.length > 0 && <p>Файлы: {payment.attachmentIds.map((id, index) => <span key={id}><a href={fileUrl(id)} target="_blank" rel="noreferrer">файл {index + 1}</a>{' '}</span>)}</p>}
    {payment.decisionReason && <p>Причина: {payment.decisionReason}</p>}
    {!completed && !blocked && payment.status === 'submitted' && payment.allowedActions?.includes('reject') && <label className="kit-field">Причина отклонения<input aria-label="Причина отклонения" value={reason} disabled={busy} onChange={event => setReason(event.target.value)} /></label>}
    {!completed && !blocked && payment.status === 'submitted' && payment.allowedActions?.includes('accept') && <label><input type="checkbox" checked={confirmed} disabled={busy} onChange={event => setConfirmed(event.target.checked)} /> Деньги поступили</label>}
    <div className="kit-row"><Button icon="back" disabled={busy || blocked} onClick={close}>Назад</Button>
      {!completed && !blocked && payment.status === 'submitted' && payment.allowedActions?.includes('accept') && <Button variant="primary" busy={busy} disabled={!confirmed} onClick={() => void decide('accept')}>Принять весь чек</Button>}
      {!completed && !blocked && payment.status === 'submitted' && payment.allowedActions?.includes('reject') && <Button variant="danger" busy={busy} disabled={!reason.trim()} onClick={() => void decide('reject')}>Отклонить весь чек</Button>}
    </div>
  </section>;
}
