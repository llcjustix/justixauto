// @vitest-environment jsdom
import { createElement } from 'react';
import { cleanup, fireEvent, render, screen, waitFor } from '@testing-library/react';
import { afterEach, describe, expect, it, vi } from 'vitest';
import { FormDialog } from './ui';

afterEach(cleanup);

function renderDialog() {
  const submit = vi.fn().mockResolvedValue(undefined);
  render(
    createElement(FormDialog, {
      title: 'Модель',
      fields: [{ name: 'make', label: 'Марка', type: 'combobox', options: ['BYD', 'Chevrolet', 'Chery'] }],
      onSubmit: submit,
      onClose: vi.fn(),
    }),
  );
  return { submit, input: screen.getByLabelText('Марка') as HTMLInputElement };
}

const shown = () => screen.queryAllByRole('option').map((o) => o.textContent);

describe('combobox suggestions', () => {
  it('lists all options on focus and filters by the typed text', () => {
    const { input } = renderDialog();
    fireEvent.focus(input);
    expect(shown()).toEqual(['BYD', 'Chevrolet', 'Chery']);
    fireEvent.change(input, { target: { value: 'che' } });
    expect(shown()).toEqual(['Chevrolet', 'Chery']);
  });

  it('picks with the keyboard and closes the list', async () => {
    const { submit, input } = renderDialog();
    fireEvent.change(input, { target: { value: 'che' } });
    fireEvent.keyDown(input, { key: 'ArrowDown' });
    fireEvent.keyDown(input, { key: 'ArrowDown' });
    fireEvent.keyDown(input, { key: 'Enter' });
    expect(input.value).toBe('Chery');
    expect(shown()).toEqual([]);
    fireEvent.click(screen.getByRole('button', { name: 'Сохранить' }));
    await waitFor(() => expect(submit).toHaveBeenCalledWith({ make: 'Chery' }));
  });

  it('keeps free text that is not in the list', async () => {
    const { submit, input } = renderDialog();
    fireEvent.change(input, { target: { value: 'Li Auto' } });
    fireEvent.click(screen.getByRole('button', { name: 'Сохранить' }));
    await waitFor(() => expect(submit).toHaveBeenCalledWith({ make: 'Li Auto' }));
  });
});
