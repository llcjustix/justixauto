import { describe, expect, it } from 'vitest';
import { calculateInstallment } from './installment-calculator';
const m = (amountMinor: string, currency = 'USD') => ({ amountMinor, currency });
const generate = (price = '1100', down = '100', termMonths = 3, firstDueDate = '2024-01-31') => calculateInstallment(m(price), { downPayment: m(down), termMonths, firstDueDate });

describe('exact interest-free schedule', () => {
  it('divides principal exactly and assigns the remainder to the final row', () => {
    const d = generate();
    expect(d.rows.map(r => r.amount.amountMinor)).toEqual(['333', '333', '334']);
    expect(d.rows.map(r => r.balance.amountMinor)).toEqual(['667', '334', '0']);
    expect(d.scheduledTotal).toEqual(m('1000'));
    expect(generate('1100', '100', 1).rows[0]!.amount).toEqual(m('1000'));
    expect(generate('1100', '100', 4).rows.map(r => r.amount.amountMinor)).toEqual(['250', '250', '250', '250']);
  });
  it('keeps all 38 digits and values beyond Number precision', () => {
    const price = '99999999999999999999999999999999999999';
    const d = generate(price, '90071992547409931', 1000);
    expect(d.rows.reduce((total, row) => total + BigInt(row.amount.amountMinor), BigInt(d.downPayment.amountMinor))).toBe(BigInt(price));
    expect(d.rows[999]!.balance.amountMinor).toBe('0');
    expect(() => generate('1'.repeat(39))).toThrow('38');
    expect(() => generate('1000', '1'.repeat(39))).toThrow('Первый взнос');
  });
  it('clamps short months to the original day and restores it, including historical years', () => {
    expect(generate().rows.map(r => r.dueDate)).toEqual(['2024-01-31', '2024-02-29', '2024-03-31']);
    expect(generate('1100', '100', 3, '2023-01-30').rows.map(r => r.dueDate)).toEqual(['2023-01-30', '2023-02-28', '2023-03-30']);
    expect(generate('1100', '100', 3, '0001-11-30').rows.map(r => r.dueDate)).toEqual(['0001-11-30', '0001-12-30', '0002-01-30']);
    expect(generate('1100', '100', 1, '9999-12-31').rows[0]!.dueDate).toBe('9999-12-31');
  });
  it.each(['2023-02-29', '1900-02-29', '2024-04-31', '0000-01-01', '2024-13-01', '2024-00-01', '2024-01-00', '2024-1-01', ''])('rejects invalid date %s', date => {
    expect(() => generate('1100', '100', 3, date)).toThrow('дату');
  });
  it('checks currencies, positive amounts, positive rows, integer bounds and year overflow', () => {
    expect(() => calculateInstallment(m('1100'), { downPayment: m('100', 'EUR'), termMonths: 3, firstDueDate: '2024-01-01' })).toThrow('Валюта');
    for (const down of ['0', '-1', '1100', '1200']) expect(() => generate('1100', down)).toThrow('Первый взнос');
    for (const months of [0, -1, 1.5, 1001, NaN]) expect(() => generate('1100', '100', months)).toThrow('1000');
    expect(() => generate('2', '1', 2)).toThrow('положительного');
    expect(() => generate('1100', '100', 2, '9999-12-01')).toThrow('9999');
    expect(() => generate('0')).toThrow('цену');
  });
});
