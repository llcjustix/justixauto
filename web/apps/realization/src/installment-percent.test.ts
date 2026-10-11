import { describe, expect, it } from 'vitest';
import { minorToPercent, percentDisplayPrecision, percentToMinor } from './installment-percent';

describe('exact installment down-payment percent', () => {
  it('converts whole and fractional percent with positive half-up minor-unit rounding', () => {
    expect(percentToMinor('10', '11000')).toBe('1100');
    expect(percentToMinor('12.345678', '11000')).toBe('1358');
    expect(percentToMinor('10', '5')).toBe('1');
  });

  it('keeps huge amounts exact and displays a money-origin percentage without using it as money', () => {
    const price = '99999999999999999999999999999999999999';
    expect(percentToMinor('10', price)).toBe('10000000000000000000000000000000000000');
    expect(minorToPercent('1001', '11000')).toBe('9.1');
    expect(minorToPercent('1', '3')).toBe('33.333333');
    expect(percentDisplayPrecision).toBe(6);
  });

  it.each(['', '0', '100', '100.000001', '-1', '1.2345678', 'word'])('rejects invalid or out-of-range percent %s', percent => {
    expect(() => percentToMinor(percent, '11000')).toThrow('больше 0%');
  });

  it('rejects percentages that round to zero or the complete price', () => {
    expect(() => percentToMinor('1', '2')).toThrow('больше 0%');
    expect(() => percentToMinor('99', '2')).toThrow('больше 0%');
  });
});
