import { minorToMajor, toMinor } from '@justixauto/kit';
import type { InstallmentDraft, InstallmentTerms, Money } from './data';
import { calculateInstallment, positiveAmount } from './installment-calculator';
import { percentToMinor } from './installment-percent';

export type TermsValues = {
  down: string;
  percent: string;
  downOrigin?: 'money' | 'percent';
  months: string;
  firstDueDate: string;
  preset?: string;
};
export const termsValues = (terms?: InstallmentTerms | null, fixedDown?: Money): TermsValues => ({
  down: fixedDown || terms ? minorToMajor((fixedDown ?? terms!.downPayment).amountMinor) : '',
  percent: '',
  downOrigin: 'money',
  months: terms ? String(terms.termMonths) : '',
  firstDueDate: terms?.firstDueDate ?? '',
});
export function previewTerms(
  price: Money,
  values: TermsValues,
  fixedDown?: Money,
): { draft: InstallmentDraft | null; error: string } {
  try {
    if (!positiveAmount(price.amountMinor)) throw new Error('Укажите положительную цену сделки.');
    if (values.preset === '') throw new Error('Выберите вариант срока.');
    const downPayment = fixedDown ?? {
      amountMinor:
        values.downOrigin === 'percent'
          ? percentToMinor(values.percent, price.amountMinor)
          : (toMinor(values.down) ?? ''),
      currency: price.currency,
    };
    if (!/^\d+$/.test(values.months)) throw new Error('Укажите целый срок от 1 до 1000 месяцев.');
    return {
      draft: calculateInstallment(price, {
        downPayment,
        termMonths: Number(values.months),
        firstDueDate: values.firstDueDate,
      }),
      error: '',
    };
  } catch (error) {
    return { draft: null, error: (error as Error).message };
  }
}
export function draftTerms(draft: InstallmentDraft): InstallmentTerms {
  return { downPayment: draft.downPayment, termMonths: draft.termMonths, firstDueDate: draft.firstDueDate };
}
