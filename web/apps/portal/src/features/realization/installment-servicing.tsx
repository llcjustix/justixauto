import { useRef, useState } from 'react';
import { ApiError, Button, Details, Notice, Panel, Table, date, errorText, fileUrl, money, post } from '@justixauto/kit';
import type { Deal, InstallmentPlan, Money } from './data';
import { InstallmentPreview, InstallmentTermsForm, draftTerms, previewTerms, termsValues } from './installment-terms-form';

function Agreement({ reference, signedOn, files }: { reference: string; signedOn: string; files: string[] }) {
  return <p>Договор {reference} от {date(signedOn)}{files.map((id, index) => <span key={id}> · <a href={fileUrl(id)} target="_blank" rel="noreferrer">Скан договора {index + 1}</a></span>)}</p>;
}

export function InstallmentEditor({ deal, onBusy, onCancel, onSaved }: {
  deal: Deal; onBusy: (busy: boolean) => void; onCancel: () => void; onSaved: () => Promise<void>;
}) {
  // Freeze the reviewed values and revision even if a background refetch arrives.
  const [snapshot] = useState(deal);
  const initial = snapshot.invoices?.filter(i => i.purpose === 'first-installment' && i.status === 'issued') ?? [];
  const fixedDown = initial.length === 1 ? initial[0]!.amount : undefined;
  const [values, setValues] = useState(() => termsValues(snapshot.installmentDraft, fixedDown));
  const [error, setError] = useState(''), [busy, setBusy] = useState(false), [stale, setStale] = useState(false), [saved, setSaved] = useState(false);
  const sending = useRef(false), completed = useRef(false);
  const preview = previewTerms(snapshot.price, values, fixedDown);
  const problem = initial.length > 1 ? 'Найдено несколько счетов первого взноса. Обновите сделку.' : preview.error;
  async function save() {
    if (sending.current || completed.current || stale || problem || !preview.draft) return;
    sending.current = true; setBusy(true); onBusy(true); setError('');
    try {
      await post(`/retail/deals/${snapshot.id}/installment-terms`, draftTerms(preview.draft), { ifMatch: snapshot.revision });
      completed.current = true; setSaved(true);
      await onSaved();
    } catch (e) {
      const conflict = e instanceof ApiError && (e.status === 409 || e.status === 412);
      if (conflict) setStale(true);
      setError([completed.current ? 'Условия сохранены. Не удалось обновить данные.' : errorText(e),
        e instanceof ApiError ? Object.entries(e.fields).map(([field, message]) => `${field}: ${message}`).join('; ') : '',
        conflict ? 'Данные изменились. Вернитесь к сделке и обновите данные перед повтором.' : '',
      ].filter(Boolean).join(' '));
    } finally { sending.current = false; setBusy(false); onBusy(false); }
  }
  return <div className="kit-stack installment-editor">
    <h3>Условия беспроцентной рассрочки</h3>
    <p>Цена: {money(snapshot.price)}. График рассчитывается автоматически. После подписания договора и выставления счёта первого взноса он фиксируется без отдельного сохранения.</p>
    {error && <Notice kind="danger">{error}</Notice>}
    {initial.length > 1 && <Notice kind="danger">{problem}</Notice>}
    <InstallmentTermsForm price={snapshot.price} values={values} onChange={setValues} fixedDown={fixedDown} disabled={busy || saved} />
    <div className="kit-row">
      <Button icon={saved ? 'back' : undefined} disabled={busy} onClick={onCancel}>{saved ? 'Назад к сделке' : 'Отмена'}</Button>
      {!saved && <Button variant="primary" busy={busy} disabled={!!problem || stale} onClick={() => void save()}>Сохранить условия</Button>}
    </div>
  </div>;
}

export function InstallmentSection({ deal, onCreate, onSelect, onPayment, onRetry }: {
  deal: Deal; onCreate: () => void; onSelect: (id: string, trigger: HTMLButtonElement) => void;
  onPayment: (payoff: boolean, trigger: HTMLButtonElement) => void; onRetry: () => void;
}) {
  const plan = deal.installmentPlan;
  if (plan) return <SavedInstallmentPlan plan={plan} onSelect={onSelect} onPayment={onPayment} />;
  if (plan === undefined || deal.installmentDraft === undefined) return <Panel title="График рассрочки" padded><Notice kind="danger">Сервер не вернул данные графика.</Notice><Button onClick={onRetry}>Повторить загрузку графика</Button></Panel>;
  const draft = deal.installmentDraft;
  const first = deal.invoices?.filter(i => i.purpose === 'first-installment' && i.status === 'issued') ?? [];
  return <Panel title="График рассрочки" padded>
    {draft ? <>
      <h3>Предварительный график — не подлежит оплате</h3>
      <p>Ежемесячные счета появятся автоматически после подписания договора и выставления счёта первого взноса.</p>
      {(!deal.contractSignedOn || !deal.contractReference) && <p>Нужен подписанный договор с датой и номером.</p>}
      {first.length !== 1 ? <p>Нужен один выставленный счёт первого взноса.</p> :
        (first[0]!.amount.amountMinor !== draft.downPayment.amountMinor || first[0]!.amount.currency !== draft.downPayment.currency) && <Notice kind="danger">Счёт первого взноса не совпадает с условиями.</Notice>}
      <InstallmentPreview draft={draft} />
    </> : <p>Условия рассрочки ещё не заданы.</p>}
    {deal.allowedActions.includes('set-installment-terms') ? <Button variant="primary" onClick={onCreate}>{draft ? 'Изменить условия рассрочки' : 'Рассчитать рассрочку'}</Button> :
      <p>{deal.status === 'cancelled' ? 'Сделка отменена.' : 'Для изменения условий нет разрешения.'}</p>}
  </Panel>;
}

function scheduledRemainingBalances(plan: InstallmentPlan) {
  let remaining = BigInt(plan.scheduledTotal.amountMinor);
  const balances = new Map<string, Money>();
  for (const row of [...plan.rows].sort((a, b) => a.number - b.number)) {
    remaining -= BigInt(row.amount.amountMinor);
    balances.set(row.invoiceId, { amountMinor: remaining.toString(), currency: plan.scheduledTotal.currency });
  }
  return balances;
}

function rowStatus(row: InstallmentPlan['rows'][number]) {
  const paid = BigInt(row.paid.amountMinor);
  const pending = BigInt(row.pending.amountMinor);
  const outstanding = BigInt(row.outstanding.amountMinor);
  if (outstanding === 0n && pending === 0n) return 'Оплачен';
  const facts: string[] = [];
  if (pending > 0n) facts.push('Ожидает подтверждения');
  if (paid > 0n) facts.push('Частично оплачен');
  return facts.join(' · ') || 'Не оплачен';
}

function SavedInstallmentPlan({ plan, onSelect, onPayment }: {
  plan: InstallmentPlan; onSelect: (id: string, trigger: HTMLButtonElement) => void;
  onPayment: (payoff: boolean, trigger: HTMLButtonElement) => void;
}) {
  const remainingBalances = scheduledRemainingBalances(plan);
  return <Panel title="График рассрочки" padded>
    <Agreement reference={plan.contractReference} signedOn={plan.contractSignedOn} files={plan.contractFileIds} />
    <p>Сохранённые условия договора. Первый взнос и регистрация не входят в итоги ежемесячных платежей.</p>
    <Details items={[
      ['Полная сумма договора', money(plan.contractTotal)],
      ['Первый взнос (отдельно)', money(plan.downPayment)],
      ['Сумма ежемесячных платежей', money(plan.scheduledTotal)],
      ['Подтверждено по графику', money(plan.paid)],
      ['На проверке по графику', money(plan.pending)],
      ['Остаток по графику', money(plan.outstanding)],
    ]} />
    {plan.settlementState === 'settled' && plan.outstanding.amountMinor === '0' && <Notice kind="success">Расчёты по ежемесячному графику подтверждены.</Notice>}
    {plan.state === 'cancelled' && <Notice>Сделка отменена. Платежи по графику недоступны.</Notice>}
    {plan.allowedActions?.includes('submit-installment-payment') && plan.state !== 'cancelled' && plan.rows.some(row => row.allowedActions?.includes('submit-payment') && BigInt(row.available.amountMinor) > 0n) && <div className="kit-row">
      <button type="button" className="btn btn-secondary" data-installment-plan-payment="manual" onClick={event => onPayment(false, event.currentTarget)}>Внести оплату</button>
      <button type="button" className="btn btn-primary" data-installment-plan-payment="payoff" onClick={event => onPayment(true, event.currentTarget)}>Погасить досрочно</button>
    </div>}
    <p>Остаток после платежа — плановый остаток после платежей по сохранённому графику; фактические оплаты показаны отдельно.</p>
    <Table rows={plan.rows} rowKey={row => row.invoiceId} columns={[
      { title: '№', render: row => row.number },
      { title: 'Дата', render: row => date(row.dueDate) },
      { title: 'Сумма', render: row => money(row.amount) },
      { title: 'Подтверждено', render: row => money(row.paid) },
      { title: 'На проверке', render: row => money(row.pending) },
      { title: 'Статус оплаты', render: row => rowStatus(row) },
      { title: 'Остаток после платежа', render: row => money(remainingBalances.get(row.invoiceId)!) },
      { title: 'Доступно к внесению', render: row => money(row.available) },
      { title: 'Действие', render: row => <button type="button" className="btn btn-secondary btn-sm" data-installment-invoice={row.invoiceId} onClick={event => onSelect(row.invoiceId, event.currentTarget)}>Открыть платёж {row.number}</button> },
    ]} />
  </Panel>;
}
