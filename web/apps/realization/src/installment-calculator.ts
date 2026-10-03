import type { InstallmentDraft, InstallmentTerms, Money } from './data';

export function positiveAmount(value: string): boolean {
  return /^\d{1,38}$/.test(value) && BigInt(value) > 0n;
}

const daysInMonth = (year: number, month: number) => month === 2
  ? (year % 4 === 0 && (year % 100 !== 0 || year % 400 === 0) ? 29 : 28)
  : [4, 6, 9, 11].includes(month) ? 30 : 31;

/** Exact zero-interest policy v1. Calendar dates never pass through local time. */
export function calculateInstallment(price: Money, terms: InstallmentTerms): InstallmentDraft {
  const { downPayment, termMonths, firstDueDate } = terms;
  if (!price.currency || price.currency !== downPayment.currency) throw new Error('Валюта первого взноса должна совпадать с валютой сделки.');
  if (!positiveAmount(price.amountMinor)) throw new Error('Укажите положительную цену (не более 38 цифр в минимальных единицах).');
  if (!positiveAmount(downPayment.amountMinor) || BigInt(downPayment.amountMinor) >= BigInt(price.amountMinor)) throw new Error('Первый взнос должен быть больше нуля и меньше цены.');
  if (!Number.isInteger(termMonths) || termMonths < 1 || termMonths > 1000) throw new Error('Укажите целый срок от 1 до 1000 месяцев.');
  const principal = BigInt(price.amountMinor) - BigInt(downPayment.amountMinor);
  if (principal < BigInt(termMonths)) throw new Error('Суммы недостаточно для положительного платежа в каждом месяце.');
  const match = /^(\d{4})-(\d{2})-(\d{2})$/.exec(firstDueDate);
  const year = Number(match?.[1]), month = Number(match?.[2]), day = Number(match?.[3]);
  if (!match || year < 1 || month < 1 || month > 12 || day < 1 || day > daysInMonth(year, month)) throw new Error('Укажите действительную дату первого платежа.');
  if (year * 12 + month - 1 + termMonths - 1 >= 10000 * 12) throw new Error('Последняя дата платежа должна быть не позднее 9999 года.');
  const amount = (value: bigint): Money => ({ amountMinor: value.toString(), currency: price.currency });
  const regular = principal / BigInt(termMonths);
  let balance = principal;
  const rows = Array.from({ length: termMonths }, (_, index) => {
    const serial = year * 12 + month - 1 + index;
    const y = Math.floor(serial / 12), m = serial % 12 + 1;
    const payment = index === termMonths - 1 ? balance : regular;
    balance -= payment;
    return { number: index + 1, dueDate: `${String(y).padStart(4, '0')}-${String(m).padStart(2, '0')}-${String(Math.min(day, daysInMonth(y, m))).padStart(2, '0')}`,
      amount: amount(payment), balance: amount(balance) };
  });
  return { policyId: 'own-interest-free-equal', policyVersion: 1, price, ...terms, scheduledTotal: amount(principal), regularPayment: amount(regular), rows };
}
