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
      title: 'Заказать',
      fields: [{ name: 'line-1', label: 'Количество', type: 'number', initial: '0', min: 0, max: 1 }],
      onSubmit: submit,
      onClose: vi.fn(),
    }),
  );
  return { submit, input: screen.getByLabelText('Количество') as HTMLInputElement };
}

describe('number field bounds', () => {
  it('refuses a quantity above the maximum before sending', async () => {
    const { submit, input } = renderDialog();
    fireEvent.change(input, { target: { value: '50' } });
    fireEvent.click(screen.getByRole('button', { name: 'Сохранить' }));
    expect(await screen.findByText('введите от 0 до 1')).toBeTruthy();
    expect(submit).not.toHaveBeenCalled();
  });

  it('submits a quantity within the bounds', async () => {
    const { submit, input } = renderDialog();
    fireEvent.change(input, { target: { value: '1' } });
    fireEvent.click(screen.getByRole('button', { name: 'Сохранить' }));
    await waitFor(() => expect(submit).toHaveBeenCalledWith({ 'line-1': '1' }));
  });
});
