// @vitest-environment jsdom
import { createElement, useState } from 'react';
import { cleanup, fireEvent, render, screen } from '@testing-library/react';
import { afterEach, describe, expect, it, vi } from 'vitest';
import { Autocomplete, Modal } from './ui';
import type { AutocompleteProps } from './ui';

afterEach(cleanup);
const options = [{ value: 'a', label: 'Same' }, { value: 'b', label: 'Same' }, { value: 'c', label: 'Other' }];
function Control(props: Partial<AutocompleteProps>) {
  const [value, setValue] = useState(props.value ?? '');
  return createElement(Autocomplete, { options, 'aria-label': 'Choice', ...props, value,
    onChange: (next) => { setValue(next); props.onChange?.(next); } });
}
const input = () => screen.getByRole('combobox') as HTMLInputElement;

describe('value-keyed autocomplete', () => {
  it('picks equal labels by mouse and keyboard with distinct selected and active identities', () => {
    const change = vi.fn();
    render(createElement(Control, { onChange: change }));
    input().focus();
    fireEvent.focus(input());
    const second = screen.getAllByRole('option', { name: 'Same' })[1]!;
    const secondId = second.id;
    expect(fireEvent.mouseDown(second)).toBe(false);
    fireEvent.click(second);
    expect(change).toHaveBeenLastCalledWith('b');
    expect(input().value).toBe('Same');
    expect(document.activeElement).toBe(input());
    expect(screen.queryByRole('listbox')).toBeNull();
    fireEvent.click(input());
    expect(screen.getAllByRole('option', { name: 'Same' })[1]!.id).toBe(secondId);
    fireEvent.keyDown(input(), { key: 'ArrowDown' });
    const first = screen.getAllByRole('option', { name: 'Same' })[0]!;
    expect(input().getAttribute('aria-activedescendant')).toBe(first.id);
    expect(first.getAttribute('aria-selected')).toBe('false');
    expect(screen.getAllByRole('option', { name: 'Same' })[1]!.getAttribute('aria-selected')).toBe('true');
    expect(fireEvent.keyDown(input(), { key: 'Enter' })).toBe(false);
    expect(change).toHaveBeenLastCalledWith('a');
  });

  it('clears the strict ID immediately on edit without inventing a match', () => {
    const change = vi.fn();
    render(createElement(Control, { value: 'a', onChange: change, required: true }));
    fireEvent.change(input(), { target: { value: 'Same' } });
    // No change event occurs for the unchanged label; actually editing invalidates it.
    fireEvent.change(input(), { target: { value: 'Sam' } });
    expect(change).toHaveBeenLastCalledWith('');
    expect(input().value).toBe('Sam');
    expect(input().checkValidity()).toBe(false);
    fireEvent.change(input(), { target: { value: 'Same' } });
    expect(change).toHaveBeenLastCalledWith('');
    fireEvent.click(screen.getByRole('button', { name: 'Очистить' }));
    expect(input().value).toBe('');
  });

  it('retains known labels across pages and follows controlled resets and arriving options', () => {
    const props = { options, value: 'b', onChange: vi.fn(), 'aria-label': 'Choice' };
    const view = render(createElement(Autocomplete, props));
    view.rerender(createElement(Autocomplete, { ...props, options: [] }));
    expect(input().value).toBe('Same');
    view.rerender(createElement(Autocomplete, { ...props, options: [], value: '' }));
    expect(input().value).toBe('');
    view.rerender(createElement(Autocomplete, { ...props, options: [], value: 'late' }));
    view.rerender(createElement(Autocomplete, { ...props, options: [{ value: 'late', label: 'Arrived' }], value: 'late' }));
    expect(input().value).toBe('Arrived');
    view.rerender(createElement(Autocomplete, { ...props, options: [], value: 'remote', selectedOption: { value: 'remote', label: 'Remote label' } }));
    expect(input().value).toBe('Remote label');
  });

  it('accepts free text and picks suggestions as values', () => {
    const change = vi.fn();
    render(createElement(Control, { mode: 'free', options: [{ value: 'Chery', label: 'Chery' }], onChange: change }));
    fireEvent.change(input(), { target: { value: 'Custom' } });
    expect(change).toHaveBeenLastCalledWith('Custom');
    expect(input().value).toBe('Custom');
    fireEvent.change(input(), { target: { value: 'che' } });
    fireEvent.keyDown(input(), { key: 'ArrowDown' });
    fireEvent.keyDown(input(), { key: 'Enter' });
    expect(change).toHaveBeenLastCalledWith('Chery');
  });

  it('preserves blank/all labels while required validity uses the committed value', () => {
    render(createElement(Control, { options: [{ value: '', label: 'All' }, ...options], required: true }));
    expect(input().value).toBe('All');
    expect(input().checkValidity()).toBe(false);
    fireEvent.focus(input());
    fireEvent.click(screen.getByRole('option', { name: 'Other' }));
    expect(input().checkValidity()).toBe(true);
    fireEvent.click(input());
    fireEvent.click(screen.getByRole('option', { name: 'All' }));
    expect(input().value).toBe('All');
    expect(input().checkValidity()).toBe(false);
  });

  it('skips disabled options, supports upward navigation and closes on Tab and blur', () => {
    render(createElement(Control, { options: [{ value: 'x', label: 'Disabled', disabled: true }, ...options] }));
    fireEvent.focus(input());
    fireEvent.click(screen.getByRole('option', { name: 'Disabled' }));
    expect(input().value).toBe('');
    fireEvent.keyDown(input(), { key: 'ArrowUp' });
    expect(input().getAttribute('aria-activedescendant')).toBe(screen.getByRole('option', { name: 'Other' }).id);
    fireEvent.keyDown(input(), { key: 'Tab' });
    expect(screen.queryByRole('listbox')).toBeNull();
    fireEvent.focus(input());
    fireEvent.blur(input());
    expect(screen.queryByRole('listbox')).toBeNull();
  });

  it('does not open or change disabled controls', () => {
    const change = vi.fn();
    render(createElement(Control, { disabled: true, value: 'a', onChange: change }));
    expect(input().disabled).toBe(true);
    fireEvent.focus(input());
    fireEvent.change(input(), { target: { value: 'Other' } });
    expect(screen.queryByRole('listbox')).toBeNull();
    expect(change).not.toHaveBeenCalled();
  });

  it('consumes popup Escape before the parent modal and prevents Enter submit', () => {
    const close = vi.fn();
    const submit = vi.fn((event) => event.preventDefault());
    render(createElement(Modal, { title: 'Dialog', onClose: close,
      children: createElement('form', { onSubmit: submit }, createElement(Control, {})) }));
    fireEvent.focus(input());
    expect(input().getAttribute('aria-expanded')).toBe('true');
    expect(fireEvent.keyDown(input(), { key: 'Enter' })).toBe(false);
    fireEvent.keyDown(input(), { key: 'Escape' });
    expect(close).not.toHaveBeenCalled();
    fireEvent.keyDown(input(), { key: 'Escape' });
    expect(close).toHaveBeenCalledTimes(1);
    expect(submit).not.toHaveBeenCalled();
  });

  it('exposes the entire local catalog beyond the first hundred choices', () => {
    render(createElement(Control, { options: Array.from({ length: 125 }, (_, i) => ({ value: String(i), label: `Item ${i}` })) }));
    fireEvent.focus(input());
    expect(screen.getAllByRole('option')).toHaveLength(125);
    fireEvent.change(input(), { target: { value: 'Item 124' } });
    fireEvent.click(screen.getByRole('option', { name: 'Item 124' }));
    expect(input().value).toBe('Item 124');
  });
});
