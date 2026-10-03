// @vitest-environment jsdom
import { createElement, useState } from 'react';
import { cleanup, fireEvent, render, screen } from '@testing-library/react';
import { afterEach, expect, it, vi } from 'vitest';
import { FilterSelect } from './ui';
afterEach(cleanup);

it('filters by actual IDs and restores the explicit all sentinel', () => {
  const change = vi.fn();
  function Filter() {
    const [value, setValue] = useState('');
    return createElement(FilterSelect, { value, onChange: (v) => { setValue(v); change(v); }, all: 'Все компании',
      options: [['a', 'Компания'], ['b', 'Компания']] });
  }
  render(createElement(Filter));
  const input = screen.getByRole('combobox', { name: 'Все компании' }) as HTMLInputElement;
  expect(input.value).toBe('Все компании');
  fireEvent.focus(input);
  fireEvent.click(screen.getAllByRole('option', { name: 'Компания' })[1]!);
  expect(change).toHaveBeenLastCalledWith('b');
  fireEvent.click(input);
  fireEvent.click(screen.getByRole('option', { name: 'Все компании' }));
  expect(change).toHaveBeenLastCalledWith('');
  expect(input.value).toBe('Все компании');
  expect(document.querySelector('select')).toBeNull();
});
