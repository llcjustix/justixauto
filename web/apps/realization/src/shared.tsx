import { useState } from 'react';
import type { ReactNode } from 'react';
import { Autocomplete, Button, Modal, Notice, errorText, minorToMajor, money, toMinor, useRefresh } from '@justixauto/kit';
import { modelName, routeLabel, useModels, useOrderModels } from './data';
import type { Terms } from './data';
import { vehicleColorsLabel } from './vehicle-colors';

export function TermsView({ terms, modelNameOf }: { terms: Terms; modelNameOf: (id: string) => string }) {
  return (
    <div className="kit-stack">
      <table className="kit-table">
        <thead>
          <tr>
            <th>Модель</th>
            <th>Кол-во</th>
            <th>Цена за ед.</th>
          </tr>
        </thead>
        <tbody>
          {terms.lines.map((l, i) => (
            <tr key={l.lineId ?? i}>
              <td><span>{modelNameOf(l.modelId)}</span><div><small>{vehicleColorsLabel(l)}</small></div></td>
              <td>{l.quantity}</td>
              <td>{money(l.unitPrice)}</td>
            </tr>
          ))}
        </tbody>
      </table>
      <p>Маршрут: {routeLabel[terms.route] ?? terms.route}</p>
      {terms.deliveryTerms && <p>Поставка: {terms.deliveryTerms}</p>}
      {terms.warrantyTerms && <p>Гарантия: {terms.warrantyTerms}</p>}
      {terms.serviceTerms && <p>Сервис: {terms.serviceTerms}</p>}
      {terms.paymentSchedule.length > 0 && (
        <p>График оплаты: {terms.paymentSchedule.map((p) => `${money(p.amount)} до ${p.dueDate}`).join('; ')}</p>
      )}
    </div>
  );
}

interface LineDraft {
  modelId: string;
  quantity: string;
  price: string;
  saved?: Terms['lines'][number];
}

const emptyLine = (): LineDraft => ({ modelId: '', quantity: '1', price: '' });

/** Exact line sum in minor units, or null while the row is incomplete. */
function lineMinor(l: LineDraft): bigint | null {
  const price = toMinor(l.price);
  const qty = Number(l.quantity);
  if (price === null || !Number.isInteger(qty) || qty < 1) return null;
  return BigInt(price) * BigInt(qty);
}

function TermsLinesTable({
  lines,
  models,
  withPrices,
  currency,
  setLine,
  setLines,
}: {
  lines: LineDraft[];
  models: ReturnType<typeof useModels>;
  withPrices: boolean;
  currency: string;
  setLine: (i: number, patch: Partial<LineDraft>) => void;
  setLines: (update: (ls: LineDraft[]) => LineDraft[]) => void;
}) {
  const sums = lines.map(lineMinor);
  const total = sums.reduce<bigint>((n, x) => n + (x ?? 0n), 0n);
  const qty = lines.reduce((n, l) => n + (Number(l.quantity) || 0), 0);
  return (
    <div className="kit-stack">
      <table className="kit-lines">
        <thead>
          <tr>
            <th style={{ width: '46%' }}>Модель *</th>
            <th style={{ width: 110 }}>Кол-во *</th>
            {withPrices && <th>Цена за ед., {currency} *</th>}
            {withPrices && <th className="kit-num">Сумма</th>}
            <th style={{ width: 40 }} />
          </tr>
        </thead>
        <tbody>
          {lines.map((l, i) => (
            <tr key={i}>
              <td>
                <Autocomplete aria-label={`Модель ${i + 1}`} options={[{ value: '', label: 'Выберите модель из каталога' }, ...(models.data ?? []).map((m) => ({ value: m.id, label: modelName(m) }))]} value={l.modelId} onChange={(modelId) => setLine(i, { modelId })} required />
                {l.saved && l.saved.modelId === l.modelId && <small>{vehicleColorsLabel(l.saved)}</small>}
              </td>
              <td>
                <input
                  type="number"
                  min={1}
                  value={l.quantity}
                  onChange={(e) => setLine(i, { quantity: e.target.value })}
                  aria-label="Количество"
                />
              </td>
              {withPrices && (
                <td>
                  <input
                    value={l.price}
                    onChange={(e) => setLine(i, { price: e.target.value })}
                    inputMode="decimal"
                    placeholder="0.00"
                    aria-label="Цена"
                  />
                </td>
              )}
              {withPrices && (
                <td className="kit-num">{sums[i] == null ? '—' : money({ amountMinor: String(sums[i]), currency })}</td>
              )}
              <td>
                {lines.length > 1 && (
                  <Button variant="link" onClick={() => setLines((ls) => ls.filter((_, j) => j !== i))}>
                    ✕
                  </Button>
                )}
              </td>
            </tr>
          ))}
        </tbody>
        {withPrices && (
          <tfoot>
            <tr>
              <td>Итого</td>
              <td>{qty} авто</td>
              <td />
              <td className="kit-num">{money({ amountMinor: String(total), currency })}</td>
              <td />
            </tr>
          </tfoot>
        )}
      </table>
      <div>
        <Button size="sm" onClick={() => setLines((ls) => [...ls, emptyLine()])}>
          + Добавить модель
        </Button>
      </div>
      {!models.isLoading && !(models.data ?? []).length && (
        <Notice kind="warning">Каталог автомобилей пуст: модели добавляет администратор платформы.</Notice>
      )}
    </div>
  );
}

export function PaymentScheduleEditor({
  schedule,
  setSchedule,
}: {
  schedule: { amount: string; dueDate: string }[];
  setSchedule: (update: (s: { amount: string; dueDate: string }[]) => { amount: string; dueDate: string }[]) => void;
}) {
  return (
    <div className="kit-stack">
      <div className="kit-field">График оплаты (сумма платежей = итог)</div>
      {schedule.map((p, i) => (
        <div key={i} className="kit-row kit-field">
          <input
            style={{ flex: 1 }}
            value={p.amount}
            onChange={(e) => setSchedule((s) => s.map((x, j) => (j === i ? { ...x, amount: e.target.value } : x)))}
            inputMode="decimal"
            placeholder="Сумма"
            aria-label="Сумма платежа"
          />
          <input
            style={{ flex: 1 }}
            type="date"
            value={p.dueDate}
            onChange={(e) => setSchedule((s) => s.map((x, j) => (j === i ? { ...x, dueDate: e.target.value } : x)))}
            aria-label="Дата платежа"
          />
          <Button variant="link" onClick={() => setSchedule((s) => s.filter((_, j) => j !== i))}>
            ✕
          </Button>
        </div>
      ))}
      <div>
        <Button size="sm" onClick={() => setSchedule((s) => [...s, { amount: '', dueDate: '' }])}>
          + Добавить платёж
        </Button>
      </div>
    </div>
  );
}

/** Checks the rows before sending; returns a Russian message or ''. */
export function linesProblem(lines: LineDraft[], withPrices: boolean): string {
  for (const [i, l] of lines.entries()) {
    const row = lines.length > 1 ? ` (строка ${i + 1})` : '';
    if (!l.modelId) return `Выберите модель${row}`;
    const q = Number(l.quantity);
    if (!Number.isInteger(q) || q < 1 || q > 10000) return `Количество — целое число от 1 до 10000${row}`;
    if (withPrices) {
      const p = toMinor(l.price);
      if (p === null || p === '0') return `Укажите цену за единицу, например 25000.00${row}`;
    }
  }
  return '';
}

function ExtraFieldControl({
  f,
  more,
  setMore,
}: {
  f: ExtraField;
  more: Record<string, string | string[]>;
  setMore: (v: Record<string, string | string[]>) => void;
}) {
  if (f.multiple && f.options) {
    const picked = (more[f.name] as string[] | undefined) ?? [];
    return (
      <fieldset className="kit-field">
        <span>{f.label}</span>
        {f.options.map(([v, l]) => (
          <label key={v} className="kit-row">
            <input
              type="checkbox"
              checked={picked.includes(v)}
              onChange={(e) =>
                setMore({ ...more, [f.name]: e.target.checked ? [...picked, v] : picked.filter((x) => x !== v) })
              }
            />
            {l}
          </label>
        ))}
      </fieldset>
    );
  }
  return (
    <label className="kit-field">
      {f.label}
      {f.required ? ' *' : ''}
      {f.options ? (
        <Autocomplete aria-label={f.label} options={[{ value: '', label: '—' }, ...f.options.map(([value, label]) => ({ value, label }))]} value={String(more[f.name] ?? '')} onChange={(value) => setMore({ ...more, [f.name]: value })} required={f.required} />
      ) : (
        <textarea value={String(more[f.name] ?? '')} onChange={(e) => setMore({ ...more, [f.name]: e.target.value })} />
      )}
    </label>
  );
}

export function TermsDialog({
  title,
  initial,
  withPrices = true,
  submitLabel = 'Сохранить',
  intro,
  onSubmit,
  onClose,
  extra,
}: {
  title: string;
  initial?: Terms | undefined;
  withPrices?: boolean;
  submitLabel?: string;
  intro?: ReactNode;
  onClose: () => void;
  onSubmit: (terms: Terms, extra: Record<string, string | string[]>) => Promise<unknown>;
  extra?: ExtraField[];
}) {
  const models = useOrderModels();
  const [currency, setCurrency] = useState(initial?.lines[0]?.unitPrice.currency ?? 'USD');
  const [lines, setLines] = useState<LineDraft[]>(
    initial?.lines.map((l) => ({
      modelId: l.modelId,
      quantity: l.quantity,
      price: minorToMajor(l.unitPrice.amountMinor),
      saved: l,
    })) ?? [emptyLine()],
  );
  const [route, setRoute] = useState(initial?.route ?? 'local');
  const [texts, setTexts] = useState({
    deliveryTerms: initial?.deliveryTerms ?? '',
    warrantyTerms: initial?.warrantyTerms ?? '',
    serviceTerms: initial?.serviceTerms ?? '',
  });
  const [schedule, setSchedule] = useState<{ amount: string; dueDate: string }[]>(
    initial?.paymentSchedule.map((p) => ({ amount: minorToMajor(p.amount.amountMinor), dueDate: p.dueDate })) ?? [],
  );
  const [more, setMore] = useState<Record<string, string | string[]>>({});
  const [error, setError] = useState('');
  const [busy, setBusy] = useState(false);
  const setLine = (i: number, patch: Partial<LineDraft>) =>
    setLines((ls) => ls.map((l, j) => (j === i ? { ...l, ...patch } : l)));
  const hasMore = Boolean(
    initial && (initial.deliveryTerms || initial.warrantyTerms || initial.serviceTerms || schedule.length),
  );

  async function submit() {
    setError('');
    const missing = extra?.find((f) => f.required && !more[f.name]);
    if (missing) {
      setError(`Заполните поле «${missing.label}»`);
      return;
    }
    if (!currency || !route) {
      setError('Выберите валюту и маршрут поставки.');
      return;
    }
    const problem = linesProblem(lines, withPrices);
    if (problem) {
      setError(problem);
      return;
    }
    const out: Terms = { lines: [], route, ...texts, paymentSchedule: [] };
    for (const l of lines) {
      const minor = withPrices ? toMinor(l.price)! : '0';
      // Preserve identity and historical facts on unrelated edits, including unknown legacy fields.
      out.lines.push({ ...(l.saved?.modelId === l.modelId ? l.saved : {}), modelId: l.modelId, quantity: l.quantity, unitPrice: { amountMinor: minor, currency } });
    }
    for (const p of schedule) {
      const minor = toMinor(p.amount);
      if (minor === null || !p.dueDate) {
        setError('Заполните сумму и дату каждого платежа');
        return;
      }
      out.paymentSchedule.push({ amount: { amountMinor: minor, currency }, dueDate: p.dueDate });
    }
    setBusy(true);
    try {
      await onSubmit(out, more);
      onClose();
    } catch (e) {
      setError(errorText(e));
    } finally {
      setBusy(false);
    }
  }

  return (
    <Modal
      title={title}
      onClose={onClose}
      size="wide"
      footer={
        <>
          <Button onClick={onClose}>Отмена</Button>
          <Button variant="primary" busy={busy} onClick={() => void submit()}>
            {submitLabel}
          </Button>
        </>
      }
    >
      {intro}
      {models.isLoading && <p role="status">Загрузка моделей…</p>}
      {models.error && <Notice kind="danger">Не удалось загрузить модели. <Button onClick={() => void models.refetch()}>Повторить загрузку моделей</Button></Notice>}
      {error && <Notice kind="danger">{error}</Notice>}
      <div className="kit-grid-2">
        {extra
          ?.filter((f) => !f.multiple)
          .map((f) => (
            <ExtraFieldControl key={f.name} f={f} more={more} setMore={setMore} />
          ))}
        {withPrices && (
          <label className="kit-field">
            Валюта
            <Autocomplete aria-label="Валюта" options={['USD', 'UZS', 'EUR'].map((value) => ({ value, label: value }))} value={currency} onChange={setCurrency} required />
          </label>
        )}
        <label className="kit-field">
          Маршрут поставки
          <Autocomplete aria-label="Маршрут поставки" options={Object.entries(routeLabel).map(([value, label]) => ({ value, label }))} value={route} onChange={setRoute} required />
        </label>
      </div>
      <TermsLinesTable
        lines={lines}
        models={models}
        withPrices={withPrices}
        currency={currency}
        setLine={setLine}
        setLines={setLines}
      />
      {withPrices && (
        <details className="kit-details" open={hasMore}>
          <summary>Дополнительные условия (необязательно)</summary>
          <div className="kit-stack">
            <label className="kit-field">
              Условия поставки
              <textarea
                value={texts.deliveryTerms}
                onChange={(e) => setTexts({ ...texts, deliveryTerms: e.target.value })}
              />
            </label>
            <div className="kit-grid-2">
              <label className="kit-field">
                Гарантия
                <input
                  value={texts.warrantyTerms}
                  onChange={(e) => setTexts({ ...texts, warrantyTerms: e.target.value })}
                />
              </label>
              <label className="kit-field">
                Сервис
                <input
                  value={texts.serviceTerms}
                  onChange={(e) => setTexts({ ...texts, serviceTerms: e.target.value })}
                />
              </label>
            </div>
            <PaymentScheduleEditor schedule={schedule} setSchedule={setSchedule} />
          </div>
        </details>
      )}
      {extra
        ?.filter((f) => f.multiple)
        .map((f) => (
          <ExtraFieldControl key={f.name} f={f} more={more} setMore={setMore} />
        ))}
    </Modal>
  );
}

/** Extra input next to the terms: text, a select, or checkboxes (multiple). */
export interface ExtraField {
  name: string;
  label: string;
  required?: boolean;
  options?: [string, string][];
  multiple?: boolean;
}

/**
 * A button opening the editor for the contract's CommercialTerms: model lines,
 * one currency, route, texts and an optional payment schedule that must add up
 * to the total. Refreshes the given keys afterwards.
 */
export function TermsButton(props: {
  label: string;
  title?: string;
  submitLabel?: string;
  intro?: ReactNode;
  size?: 'sm';
  initial?: Terms | undefined;
  withPrices?: boolean;
  variant?: 'primary';
  extra?: ExtraField[];
  refresh?: unknown[][];
  onSubmit: (t: Terms, extra: Record<string, string | string[]>) => Promise<unknown>;
}) {
  const [open, setOpen] = useState(false);
  const reload = useRefresh();
  return (
    <>
      <Button variant={props.variant} size={props.size} onClick={() => setOpen(true)}>
        {props.label}
      </Button>
      {open && (
        <TermsDialog
          title={props.title ?? props.label}
          {...(props.submitLabel ? { submitLabel: props.submitLabel } : {})}
          {...(props.intro ? { intro: props.intro } : {})}
          initial={props.initial}
          {...(props.withPrices === undefined ? {} : { withPrices: props.withPrices })}
          {...(props.extra ? { extra: props.extra } : {})}
          onClose={() => setOpen(false)}
          onSubmit={async (t, x) => {
            const r = await props.onSubmit(t, x);
            await reload(...(props.refresh ?? []));
            return r;
          }}
        />
      )}
    </>
  );
}
