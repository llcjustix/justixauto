// @vitest-environment jsdom
import { useState } from 'react';
import { cleanup, fireEvent, render, screen } from '@testing-library/react';
import { afterEach, describe, expect, it } from 'vitest';
import { validateVins, VinEditor } from './vin-editor';

afterEach(cleanup);
const vin = (index: number) => `LGXC16DF0P${String(index).padStart(7, '0')}`;
function Editor({ cap = 5, initial = [], required = false }: { cap?: number; initial?: string[]; required?: boolean }) {
  const [rows, setRows] = useState(initial);
  return <VinEditor rows={rows} onChange={setRows} cap={cap} required={required} />;
}

describe('VIN row editor', () => {
  it('normalizes optional rows without imposing extra national or check-digit rules', () => {
    expect(validateVins(['  lgxc16df0p0000001 ', ''], 4)).toMatchObject({ valid: true, vins: [vin(1)] });
    expect(validateVins([], 4).valid).toBe(true);
    expect(validateVins([''], 4, true).errors).toContain('Введите хотя бы один VIN');
    for (const value of ['LGXC16DF0P000000I', 'LGXC16DF0P000000O', 'LGXC16DF0P000000Q', '短'.repeat(17), '123']) {
      expect(validateVins([value], 4).valid).toBe(false);
    }
    expect(validateVins([vin(1), vin(1).toLowerCase()], 4).rowErrors).toEqual(['Повторяющийся VIN', 'Повторяющийся VIN']);
  });

  it('adds, edits, normalizes on blur, and removes individual rows', () => {
    render(<Editor />);
    fireEvent.click(screen.getByRole('button', { name: 'Добавить VIN' }));
    fireEvent.change(screen.getByLabelText('VIN 1'), { target: { value: ' lgxc16df0p0000001 ' } });
    fireEvent.blur(screen.getByLabelText('VIN 1'));
    expect((screen.getByLabelText('VIN 1') as HTMLInputElement).value).toBe(vin(1));
    fireEvent.click(screen.getByRole('button', { name: 'Добавить VIN' }));
    fireEvent.change(screen.getByLabelText('VIN 2'), { target: { value: vin(1) } });
    expect(screen.getAllByText('Повторяющийся VIN')).toHaveLength(2);
    fireEvent.click(screen.getByRole('button', { name: 'Удалить VIN 1' }));
    expect(screen.queryByText('Повторяющийся VIN')).toBeNull();
    expect((screen.getByLabelText('VIN 1') as HTMLInputElement).value).toBe(vin(1));
  });

  it('keeps all pasted values over the identification cap and renders only one page', () => {
    render(<Editor cap={1000} required />);
    fireEvent.click(screen.getByText('Вставить список VIN'));
    fireEvent.change(screen.getByLabelText('Список VIN для добавления'), { target: { value: Array.from({ length: 1001 }, (_, index) => vin(index)).join('\n') } });
    fireEvent.click(screen.getByRole('button', { name: 'Добавить список' }));
    expect(screen.getByText(/VIN больше допустимого: 1001 из 1000/)).toBeTruthy();
    expect(screen.getAllByRole('textbox')).toHaveLength(51);
    expect(screen.getByText('Страница VIN 1 из 21')).toBeTruthy();
    fireEvent.click(screen.getByRole('button', { name: 'Следующие VIN' }));
    expect((screen.getByLabelText('VIN 51') as HTMLInputElement).value).toBe(vin(50));
  });

  it('uses the caller shipping quantity without an invented 1000-VIN shipment limit', () => {
    const rows = Array.from({ length: 1001 }, (_, index) => vin(index));
    expect(validateVins(rows, 1200).valid).toBe(true);
    expect(validateVins(rows, 1000, true).valid).toBe(false);
    render(<Editor cap={10000} initial={Array.from({ length: 10000 }, (_, index) => vin(index))} />);
    expect(screen.getAllByLabelText(/^VIN \d+$/)).toHaveLength(50);
    expect(screen.getByText('Страница VIN 1 из 200')).toBeTruthy();
  });
});
