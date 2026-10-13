import { expect, it } from 'vitest';
import type { Model, Spec } from './data';
import { colorOptions, colorPalette, exactSpecification, selectedColor, vehicleColorsLabel } from './vehicle-colors';

const spec = (fields: Partial<Spec>) => ({ version: '1', make: 'Make', model: 'Model', variant: '', year: 2026,
  bodyType: 'sedan', powertrain: '', drivetrain: '', exteriorColor: '', interiorColor: '', ...fields });

it('color helper selects exact history and never falls back to current or an unpinned version', () => {
  const v1 = spec({ exteriorColors: ['White', 'Blue'], interiorColors: ['Black', 'Tan'] });
  const v2 = spec({ version: '2', exteriorColors: ['Red'], interiorColors: ['Grey'] });
  const model: Model = { id: 'm', revision: '2', specification: v2, versions: [v1, v2] };
  expect(exactSpecification(model, '1')).toBe(v1);
  expect(colorPalette(exactSpecification(model, '1'), 'exterior')).toEqual(['White', 'Blue']);
  expect(exactSpecification({ ...model, versions: [v2] }, '1')).toBeUndefined();
  expect(exactSpecification(model, '')).toBeUndefined();
  expect(exactSpecification(undefined, '1')).toBeUndefined();
});

it('color helper supports scalar fixtures without inventing multi-palette defaults', () => {
  const legacy = spec({ exteriorColor: 'White', interiorColor: 'Black' });
  expect(colorPalette(legacy, 'exterior')).toEqual(['White']);
  expect(selectedColor(colorPalette(legacy, 'interior'))).toBe('Black');
  expect(selectedColor(['White', 'Blue'])).toBe('');
  expect(colorPalette(spec({ exteriorColor: 'White', exteriorColors: [] }), 'exterior')).toEqual([]);
});

it('color helper restricts offered choices and clears stale or wrong-version options', () => {
  const palette = spec({ exteriorColors: ['White', 'Blue'], interiorColors: ['Black', 'Tan'] });
  expect(colorOptions(palette, 'exterior', 'white')).toEqual(['White']);
  expect(colorOptions(palette, 'exterior', 'Red')).toEqual([]);
  expect(selectedColor(['White', 'Blue'], 'Red')).toBe('');
  expect(selectedColor(['White', 'Blue'], 'blue')).toBe('Blue');
  expect(selectedColor([], 'White')).toBe('');
});

it('color helper labels each unknown independently without a catalogue fallback', () => {
  expect(vehicleColorsLabel({ exteriorColor: 'White' })).toBe('Кузов: White · Салон: Не указан · Версия: Не указан');
  expect(vehicleColorsLabel({ interiorColor: 'Tan', modelSpecificationVersion: '1' })).toBe('Кузов: Не указан · Салон: Tan · Версия: 1');
});
