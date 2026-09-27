// @vitest-environment jsdom
import { createElement } from 'react';
import { cleanup, fireEvent, render, screen, waitFor } from '@testing-library/react';
import { afterEach, describe, expect, it, vi } from 'vitest';
import { FormDialog } from './ui';
import type { FieldSpec } from './ui';

afterEach(cleanup);

const field: FieldSpec = {
  name: 'companyId',
  label: 'Компания',
  type: 'select',
  searchable: true,
  full: true,
  required: true,
  options: [
    ['c-1', 'Авто плюс · Узбекистан'],
    ['c-2', 'Банк Один · Казахстан'],
  ],
};

describe('searchable select', () => {
  it('suggests the option labels and submits the chosen option value', async () => {
    const submit = vi.fn().mockResolvedValue(undefined);
    render(createElement(FormDialog, { title: 'Партнёр', fields: [field], onSubmit: submit, onClose: vi.fn() }));

    const input = screen.getByLabelText('Компания') as HTMLInputElement;
    const list = document.getElementById(input.getAttribute('list')!)!;
    expect([...list.querySelectorAll('option')].map((o) => o.value)).toEqual([
      'Авто плюс · Узбекистан',
      'Банк Один · Казахстан',
    ]);
    fireEvent.change(input, { target: { value: 'банк один · казахстан' } });
    fireEvent.click(screen.getByRole('button', { name: 'Сохранить' }));
    await waitFor(() => expect(submit).toHaveBeenCalled());
    expect(submit.mock.calls[0][0]).toMatchObject({ companyId: 'c-2' });
  });

  it('does not submit text that matches no company', async () => {
    const submit = vi.fn().mockResolvedValue(undefined);
    render(createElement(FormDialog, { title: 'Партнёр', fields: [field], onSubmit: submit, onClose: vi.fn() }));
    fireEvent.change(screen.getByLabelText('Компания'), { target: { value: 'Несуществующая' } });
    fireEvent.click(screen.getByRole('button', { name: 'Сохранить' }));
    expect(await screen.findByText('выберите вариант из списка')).toBeTruthy();
    expect(submit).not.toHaveBeenCalled();
  });
});
