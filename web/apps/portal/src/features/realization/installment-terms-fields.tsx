import { Autocomplete, Details, Notice, Table, minorToMajor, money, toMinor } from '@justixauto/kit';
import type { InstallmentDraft, Money } from './data';
import { minorToPercent, percentToMinor } from './installment-percent';
import { previewTerms } from './installment-terms-values';
import type { TermsValues } from './installment-terms-values';

export function InstallmentPreview({ draft }: { draft: InstallmentDraft }) {
  return (
    <section aria-label="Предварительный график">
      <p>Беспроцентная рассрочка. Последний платёж включает остаток округления.</p>
      <Details
        items={[
          ['Цена', money(draft.price)],
          ['Первый взнос', money(draft.downPayment)],
          ['Сумма рассрочки', money(draft.scheduledTotal)],
          ['Срок, месяцев', draft.termMonths],
          ['Обычный платёж', money(draft.regularPayment)],
        ]}
      />
      <Table
        rows={draft.rows}
        rowKey={(row) => String(row.number)}
        columns={[
          { title: '№', render: (row) => row.number },
          { title: 'Дата', render: (row) => row.dueDate },
          { title: 'Платёж', render: (row) => money(row.amount) },
          { title: 'Остаток', render: (row) => money(row.balance) },
        ]}
      />
    </section>
  );
}
export function InstallmentTermsForm({
  price,
  values,
  onChange,
  fixedDown,
  disabled = false,
}: {
  price: Money;
  values: TermsValues;
  onChange: (values: TermsValues) => void;
  fixedDown?: Money | undefined;
  disabled?: boolean;
}) {
  const preview = previewTerms(price, values, fixedDown);
  const presets = ['12', '24', '36', '48', '60'];
  const amountMinor = toMinor(values.down) ?? '';
  const shownPercent =
    values.downOrigin === 'percent' ? values.percent : minorToPercent(amountMinor, price.amountMinor);
  const shownDown =
    values.downOrigin === 'percent'
      ? (() => {
          try {
            return minorToMajor(percentToMinor(values.percent, price.amountMinor));
          } catch {
            return '';
          }
        })()
      : values.down;
  const changePercent = (percent: string) => {
    try {
      const amount = percentToMinor(percent, price.amountMinor);
      onChange({ ...values, percent, down: minorToMajor(amount), downOrigin: 'percent' });
    } catch {
      onChange({ ...values, percent, down: '', downOrigin: 'percent' });
    }
  };
  const changeMoney = (down: string) => onChange({ ...values, down, percent: '', downOrigin: 'money' });
  return (
    <div className="kit-stack sale-terms-form">
      <div className="sale-terms-fields">
        <label className="kit-field">
          Первый взнос, %
          <input
            aria-label="Первый взнос, %"
            inputMode="decimal"
            value={shownPercent}
            readOnly={!!fixedDown}
            disabled={disabled}
            onChange={fixedDown ? undefined : (e) => changePercent(e.target.value)}
          />
        </label>
        <label className="kit-field">
          Первый взнос ({price.currency})
          <input
            aria-label="Первый взнос"
            inputMode="decimal"
            value={shownDown}
            readOnly={!!fixedDown}
            disabled={disabled}
            onChange={fixedDown ? undefined : (e) => changeMoney(e.target.value)}
          />
        </label>
        <p className="cell-sub">Процент и сумма первого взноса синхронизируются.</p>
        {fixedDown && <p>Первый взнос зафиксирован выставленным счётом: {money(fixedDown)}. Изменение недоступно.</p>}
        <label className="kit-field">
          Вариант срока
          <Autocomplete
            aria-label="Вариант срока"
            disabled={disabled}
            required
            allowClear={false}
            value={values.preset ?? (presets.includes(values.months) ? values.months : 'custom')}
            onChange={(preset) =>
              onChange({ ...values, preset, months: preset ? (preset === 'custom' ? '' : preset) : values.months })
            }
            options={[
              ...presets.map((months) => ({ value: months, label: `${months} месяцев` })),
              { value: 'custom', label: 'Другой срок' },
            ]}
          />
        </label>
        <label className="kit-field">
          Срок, месяцев
          <input
            aria-label="Срок, месяцев"
            type="number"
            min="1"
            max="1000"
            step="1"
            value={values.months}
            disabled={disabled}
            onChange={(e) =>
              onChange({
                ...values,
                months: e.target.value,
                preset: presets.includes(e.target.value) ? e.target.value : 'custom',
              })
            }
          />
        </label>
        <label className="kit-field">
          Дата первого платежа
          <input
            aria-label="Дата первого платежа"
            type="date"
            value={values.firstDueDate}
            disabled={disabled}
            onChange={(e) => onChange({ ...values, firstDueDate: e.target.value })}
          />
        </label>
        {preview.draft ? <InstallmentPreview draft={preview.draft} /> : <Notice kind="danger">{preview.error}</Notice>}
      </div>
    </div>
  );
}
