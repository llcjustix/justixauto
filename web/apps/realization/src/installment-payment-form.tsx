import { useEffect, useRef, useState } from 'react';
import { ApiError, Button, Notice, minorToMajor, money, post, toMinor, upload } from '@justixauto/kit';
import type { InstallmentPlan, RetailInvoice } from './data';

export interface InstallmentPaymentFormProps {
  dealId: string;
  plan: InstallmentPlan;
  invoices: RetailInvoice[];
  selectedInvoiceId?: string;
  /** P6 uses this for the plan-level "pay remaining schedule" entry. */
  initialFullRemaining?: boolean;
  onBack: () => void;
  /** Mirrors upload, POST and host-refresh busy state for the enclosing Modal. */
  onBusyChange?: (busy: boolean) => void;
  /** Refreshes host data after a successful receipt; a failure must only affect reading. */
  onSubmitted: () => Promise<void>;
  blocked?: boolean;
}

type DraftAllocation = { invoiceId: string; amount: string };

export function eligibleInstallmentInvoices(dealId: string, plan: InstallmentPlan, invoices: RetailInvoice[]) {
  if (plan.state === 'cancelled' || !plan.allowedActions?.includes('submit-installment-payment')) return [];
  const byId = new Map(invoices.map(invoice => [invoice.id, invoice]));
  return plan.rows.map(row => ({ row, invoice: byId.get(row.invoiceId) })).filter((value): value is { row: InstallmentPlan['rows'][number]; invoice: RetailInvoice } => {
    const { row, invoice } = value;
    return !!invoice && invoice.dealId === dealId && invoice.purpose === 'monthly-installment' && invoice.status === 'issued'
      && invoice.installmentNumber === row.number && invoice.amount.amountMinor === row.amount.amountMinor
      && row.allowedActions?.includes('submit-payment') && invoice.allowedActions?.includes('submit-payment')
      && [row.amount, row.available, invoice.amount, invoice.available, invoice.pending, invoice.outstanding].every(amount => amount.currency === plan.scheduledTotal.currency)
      && BigInt(invoice.available.amountMinor) > 0n && BigInt(row.available.amountMinor) > 0n;
  });
}

function parseAmount(value: string) {
  const minor = toMinor(value);
  return minor === null ? null : BigInt(minor);
}

export function InstallmentPaymentForm({ dealId, plan, invoices, selectedInvoiceId, initialFullRemaining = false, onBack, onBusyChange, onSubmitted, blocked = false }: InstallmentPaymentFormProps) {
  const initial = eligibleInstallmentInvoices(dealId, plan, invoices);
  const [rows, setRows] = useState<DraftAllocation[]>(() => initialFullRemaining ? initial.filter(({ invoice }) => BigInt(invoice.available.amountMinor) > 0n).map(({ invoice }) => ({ invoiceId: invoice.id, amount: minorToMajor(invoice.available.amountMinor) })) : selectedInvoiceId && initial.some(value => value.invoice.id === selectedInvoiceId) ? [{ invoiceId: selectedInvoiceId, amount: '' }] : []);
  const [claimed, setClaimed] = useState(() => initialFullRemaining ? minorToMajor(initial.reduce((sum, { invoice }) => sum + BigInt(invoice.available.amountMinor), 0n).toString()) : '');
  const [paidOn, setPaidOn] = useState('');
  const [reference, setReference] = useState('');
  const [file, setFile] = useState<File | null>(null);
  const [error, setError] = useState('');
  const [busy, setBusy] = useState(false);
  const [completed, setCompleted] = useState(false);
  const sending = useRef(false);
  const binding = useRef<{ file: File; id: string } | null>(null);
  useEffect(() => { if (completed && !blocked) setError(''); }, [completed, blocked]);
  const currency = plan.scheduledTotal.currency;
  const selected = new Set(rows.map(row => row.invoiceId));
  const allocations = rows.map(row => ({ ...row, minor: parseAmount(row.amount) }));
  const allocationTotal = allocations.reduce((sum, row) => sum + (row.minor ?? 0n), 0n);
  const claimedMinor = parseAmount(claimed);
  const available = new Map(initial.map(({ invoice }) => [invoice.id, BigInt(invoice.available.amountMinor)]));
  const invalid = blocked || !rows.length || rows.some(row => {
    const amount = parseAmount(row.amount);
    return amount === null || amount <= 0n || amount > (available.get(row.invoiceId) ?? 0n);
  }) || claimedMinor === null || claimedMinor <= 0n || claimedMinor !== allocationTotal;
  const back = () => { if (!busy && !blocked) onBack(); };
  const setRow = (invoiceId: string, enabled: boolean) => setRows(current => enabled
    ? [...current, { invoiceId, amount: '' }]
    : current.filter(row => row.invoiceId !== invoiceId));
  const changeAmount = (invoiceId: string, amount: string) => setRows(current => current.map(row => row.invoiceId === invoiceId ? { ...row, amount } : row));
  const payoff = () => {
    setRows(initial.filter(({ invoice }) => BigInt(invoice.available.amountMinor) > 0n)
      .map(({ invoice }) => ({ invoiceId: invoice.id, amount: minorToMajor(invoice.available.amountMinor) })));
    setClaimed(minorToMajor(initial.reduce((sum, { invoice }) => sum + BigInt(invoice.available.amountMinor), 0n).toString()));
  };
  async function submit() {
    if (sending.current || completed || invalid || !paidOn || !reference.trim()) return;
    sending.current = true; setBusy(true); onBusyChange?.(true); setError('');
    try {
      if (file && binding.current?.file !== file) binding.current = { file, id: (await upload(file, 'payment-evidence')).id };
      const attachmentBindingIds = file && binding.current ? [binding.current.id] : [];
      await post(`/retail/deals/${dealId}/installment-payments`, {
        claimedAmount: { amountMinor: claimedMinor!.toString(), currency }, paidOn,
        externalReference: reference.trim(), attachmentBindingIds,
        allocations: rows.map(row => ({ invoiceId: row.invoiceId, amount: { amountMinor: parseAmount(row.amount)!.toString(), currency } })),
      });
      setCompleted(true);
      try { await onSubmitted(); }
      catch { setError('Не удалось обновить данные; повторите загрузку перед следующим действием.'); }
    } catch (reason) { setError(reason instanceof ApiError ? [reason.message, ...Object.values(reason.fields)].filter(Boolean).join(' · ') : String(reason)); }
    finally { sending.current = false; setBusy(false); onBusyChange?.(false); }
  }
  return <section className="kit-stack" aria-label="Ручное распределение платежа">
    <h3>Распределить оплату по графику</h3>
    <p>Выберите месяцы и суммы. Деньги не распределяются автоматически; график считается закрытым только после подтверждения финансовым сотрудником.</p>
    {completed && <Notice kind="success">Платёж отправлен на проверку. Подтверждение финансового сотрудника ещё не получено.</Notice>}
    {error && <Notice kind="danger">{error}</Notice>}
    {!completed && initial.length === 0 && <Notice>Нет доступных месяцев или разрешения на внесение оплаты.</Notice>}
    {initial.some(({ invoice }) => BigInt(invoice.pending.amountMinor) > 0n) && <Notice kind="warning">Суммы «доступно» исключают платежи на проверке. Даже полное покрытие останется неподтверждённым до решения финансового сотрудника.</Notice>}
    {!completed && initial.length > 0 && <fieldset disabled={busy || blocked} className="kit-stack">
      <div className="kit-row"><Button onClick={payoff} disabled={busy || !initial.some(({ invoice }) => BigInt(invoice.available.amountMinor) > 0n)}>Оплатить остаток графика</Button></div>
      <table className="kit-lines"><thead><tr><th>Месяц</th><th>Дата</th><th>Доступно</th><th>Сумма</th></tr></thead><tbody>{initial.map(({ row, invoice }) => <tr key={invoice.id}>
        <td><label><input aria-label={`Выбрать месяц ${row.number}`} type="checkbox" checked={selected.has(invoice.id)} disabled={busy || BigInt(invoice.available.amountMinor) === 0n} onChange={event => setRow(invoice.id, event.target.checked)} /> {row.number}</label></td>
        <td>{row.dueDate}</td><td>{money(invoice.available)}</td>
        <td>{selected.has(invoice.id) && <input aria-label={`Сумма месяца ${row.number}`} inputMode="decimal" value={rows.find(value => value.invoiceId === invoice.id)?.amount ?? ''} disabled={busy} onChange={event => changeAmount(invoice.id, event.target.value)} />}</td>
      </tr>)}</tbody></table>
      <label className="kit-field">Сумма по чеку<input aria-label="Сумма по чеку" inputMode="decimal" value={claimed} disabled={busy} onChange={event => setClaimed(event.target.value)} /></label>
      <p>Выбрано: {money({ amountMinor: allocationTotal.toString(), currency })}. Не распределено: {claimedMinor === null ? '—' : money({ amountMinor: (claimedMinor - allocationTotal).toString(), currency })}.</p>
      <label className="kit-field">Дата оплаты<input aria-label="Дата оплаты" type="date" value={paidOn} disabled={busy} onChange={event => setPaidOn(event.target.value)} /></label>
      <label className="kit-field">Номер платёжки / чека<input aria-label="Номер платёжки / чека" value={reference} disabled={busy} onChange={event => setReference(event.target.value)} /></label>
      <label className="kit-field">Подтверждение (файл)<input aria-label="Подтверждение (файл)" type="file" disabled={busy} onChange={event => setFile(event.target.files?.[0] ?? null)} /></label>
    </fieldset>}
    <div className="kit-row"><Button icon="back" disabled={busy || blocked} onClick={back}>Назад</Button>{!completed && initial.length > 0 && <Button variant="primary" busy={busy} disabled={invalid || !paidOn || !reference.trim()} onClick={() => void submit()}>Отправить на проверку</Button>}</div>
  </section>;
}
