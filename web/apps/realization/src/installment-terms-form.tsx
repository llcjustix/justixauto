import { Autocomplete, Details, Notice, Table, minorToMajor, money, toMinor } from '@justixauto/kit';
import type { InstallmentDraft, InstallmentTerms, Money } from './data';
import { calculateInstallment } from './installment-calculator';

export type TermsValues = { down: string; months: string; firstDueDate: string; preset?: string };
export const termsValues = (terms?: InstallmentTerms | null, fixedDown?: Money): TermsValues => ({
  down: fixedDown || terms ? minorToMajor((fixedDown ?? terms!.downPayment).amountMinor) : '',
  months: terms ? String(terms.termMonths) : '', firstDueDate: terms?.firstDueDate ?? '',
});
export function previewTerms(price: Money, values: TermsValues, fixedDown?: Money): { draft: InstallmentDraft | null; error: string } {
  try {
    if (values.preset === '') throw new Error('Выберите вариант срока.');
    const downPayment = fixedDown ?? { amountMinor: toMinor(values.down) ?? '', currency: price.currency };
    if (!/^\d+$/.test(values.months)) throw new Error('Укажите целый срок от 1 до 1000 месяцев.');
    return { draft: calculateInstallment(price, { downPayment, termMonths: Number(values.months), firstDueDate: values.firstDueDate }), error: '' };
  } catch (error) { return { draft: null, error: (error as Error).message }; }
}
export function draftTerms(draft: InstallmentDraft): InstallmentTerms {
  return { downPayment: draft.downPayment, termMonths: draft.termMonths, firstDueDate: draft.firstDueDate };
}
export function InstallmentPreview({ draft }: { draft: InstallmentDraft }) {
  return <section aria-label="Предварительный график">
    <p>Беспроцентная рассрочка. Последний платёж включает остаток округления.</p>
    <Details items={[
      ['Цена', money(draft.price)], ['Первый взнос', money(draft.downPayment)],
      ['Сумма рассрочки', money(draft.scheduledTotal)], ['Срок, месяцев', draft.termMonths],
      ['Обычный платёж', money(draft.regularPayment)],
    ]} />
    <Table rows={draft.rows} rowKey={row => String(row.number)} columns={[
      { title: '№', render: row => row.number }, { title: 'Дата', render: row => row.dueDate },
      { title: 'Платёж', render: row => money(row.amount) }, { title: 'Остаток', render: row => money(row.balance) },
    ]} />
  </section>;
}
export function InstallmentTermsForm({ price, values, onChange, fixedDown, disabled = false }: {
  price: Money; values: TermsValues; onChange: (values: TermsValues) => void; fixedDown?: Money | undefined; disabled?: boolean;
}) {
  const preview = previewTerms(price, values, fixedDown);
  const presets = ['12', '24', '36', '48', '60'];
  return <div className="kit-stack sale-terms-form">
    <div className="sale-terms-fields">
    <label className="kit-field">Первый взнос ({price.currency})
      <input aria-label="Первый взнос" inputMode="decimal" value={values.down} readOnly={!!fixedDown} disabled={disabled}
        onChange={e => onChange({ ...values, down: e.target.value })} />
    </label>
    {fixedDown && <p>Первый взнос зафиксирован выставленным счётом: {money(fixedDown)}. Изменение недоступно.</p>}
    <label className="kit-field">Вариант срока
      <Autocomplete aria-label="Вариант срока" disabled={disabled} required allowClear={false}
        value={values.preset ?? (presets.includes(values.months) ? values.months : 'custom')}
        onChange={preset => onChange({ ...values, preset, months: preset ? preset === 'custom' ? '' : preset : values.months })}
        options={[...presets.map(months => ({ value: months, label: `${months} месяцев` })), { value: 'custom', label: 'Другой срок' }]} />
    </label>
    <label className="kit-field">Срок, месяцев
      <input aria-label="Срок, месяцев" type="number" min="1" max="1000" step="1" value={values.months} disabled={disabled}
        onChange={e => onChange({ ...values, months: e.target.value, preset: presets.includes(e.target.value) ? e.target.value : 'custom' })} />
    </label>
    <label className="kit-field">Дата первого платежа
      <input aria-label="Дата первого платежа" type="date" value={values.firstDueDate} disabled={disabled}
        onChange={e => onChange({ ...values, firstDueDate: e.target.value })} />
    </label>
      {preview.draft ? <InstallmentPreview draft={preview.draft} /> : <Notice kind="danger">{preview.error}</Notice>}
    </div>
  </div>;
}
