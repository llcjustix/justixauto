// @vitest-environment jsdom
import { createElement } from 'react';
import { cleanup, fireEvent, render, screen, waitFor } from '@testing-library/react';
import { afterEach, describe, expect, it, vi } from 'vitest';
import { FormDialog, minorToMajor, money, toMinor } from './ui';
afterEach(cleanup);

it('currency autocomplete rejects an unmatched currency and keeps exact minor units', async () => {
  const submit = vi.fn().mockResolvedValue(undefined);
  render(createElement(FormDialog, { title: 'Money', fields: [
    { name: 'amount', label: 'Amount', type: 'money', required: true, initial: '90071992547409931.23' },
  ], onSubmit: submit, onClose: vi.fn() }));
  const currency = screen.getByRole('combobox', { name: 'Валюта' }) as HTMLInputElement;
  expect(currency.value).toBe('USD');
  fireEvent.change(currency, { target: { value: 'other' } });
  expect(currency.checkValidity()).toBe(false);
  fireEvent.click(screen.getByRole('button', { name: 'Сохранить' }));
  expect(submit).not.toHaveBeenCalled();
  expect(screen.getByText('выберите валюту из списка')).toBeTruthy();
  fireEvent.change(currency, { target: { value: 'UZS' } });
  fireEvent.click(screen.getByRole('option', { name: 'UZS' }));
  fireEvent.click(screen.getByRole('button', { name: 'Сохранить' }));
  await waitFor(() => expect(submit).toHaveBeenCalledWith({ amount: { amountMinor: '9007199254740993123', currency: 'UZS' } }));
});

describe('money helpers', () => {
  it('parses decimal input to minor units without floats', () => {
    expect(toMinor('1 234,5')).toBe('123450');
    expect(toMinor('0.07')).toBe('7');
    expect(toMinor('1.234')).toBeNull();
    expect(toMinor('-1')).toBeNull();
  });
  it('round-trips minor units', () => {
    expect(minorToMajor('123450')).toBe('1234.50');
    expect(toMinor(minorToMajor('9007199254740993123'))).toBe('9007199254740993123');
  });
  it('formats money with its currency', () => {
    expect(money({ amountMinor: '100', currency: 'USD' })).toContain('USD');
    expect(money(undefined)).toBe('—');
  });
});

describe('error messages', () => {
  it('translates known API codes', async () => {
    const { errorMessages } = await import('./messages');
    expect(errorMessages.stale_revision).toMatch(/Обновите/);
  });
});
