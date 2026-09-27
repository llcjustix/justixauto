// @vitest-environment jsdom
import { createElement } from 'react';
import { cleanup, fireEvent, render, screen, waitFor } from '@testing-library/react';
import { afterEach, describe, expect, it, vi } from 'vitest';
import { FormDialog } from './ui';
import type { FieldSpec } from './ui';

afterEach(cleanup);

const base: FieldSpec = {
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

const suggestions = (input: HTMLInputElement) =>
  [...document.getElementById(input.getAttribute('list') ?? '')!.querySelectorAll('option')].map((o) => o.value);

function renderDialog(field: FieldSpec) {
  const submit = vi.fn().mockResolvedValue(undefined);
  render(createElement(FormDialog, { title: 'Партнёр', fields: [field], onSubmit: submit, onClose: vi.fn() }));
  return { submit, input: screen.getByLabelText('Компания') as HTMLInputElement };
}

describe('searchable select', () => {
  it('suggests the option labels and submits the chosen option value', async () => {
    const { submit, input } = renderDialog(base);
    expect(suggestions(input)).toEqual(['Авто плюс · Узбекистан', 'Банк Один · Казахстан']);
    fireEvent.change(input, { target: { value: 'банк один · казахстан' } });
    fireEvent.click(screen.getByRole('button', { name: 'Сохранить' }));
    await waitFor(() => expect(submit).toHaveBeenCalledWith(expect.objectContaining({ companyId: 'c-2' })));
  });

  it('does not submit text that matches no option', async () => {
    const { submit, input } = renderDialog(base);
    fireEvent.change(input, { target: { value: 'Несуществующая' } });
    fireEvent.click(screen.getByRole('button', { name: 'Сохранить' }));
    expect(await screen.findByText('выберите вариант из списка')).toBeTruthy();
    expect(submit).not.toHaveBeenCalled();
  });

  it('searches on the server: first page on open, then the typed text', async () => {
    const search = vi.fn(async (query: string): Promise<[string, string][]> =>
      query === '' ? [['c-1', 'Авто плюс']] : [['c-9', `${query} найдено`]],
    );
    const { submit, input } = renderDialog({ ...base, options: [], search });

    await waitFor(() => expect(suggestions(input)).toEqual(['Авто плюс']));
    expect(search).toHaveBeenCalledWith('');

    fireEvent.change(input, { target: { value: 'Лиз' } });
    await waitFor(() => expect(suggestions(input)).toEqual(['Лиз найдено']));
    expect(search).toHaveBeenLastCalledWith('Лиз');

    fireEvent.change(input, { target: { value: 'Лиз найдено' } });
    fireEvent.click(screen.getByRole('button', { name: 'Сохранить' }));
    await waitFor(() => expect(submit).toHaveBeenCalledWith(expect.objectContaining({ companyId: 'c-9' })));
  });

  it('selects by name alone and does not search again after a suggestion is picked', async () => {
    const search = vi.fn(async (): Promise<[string, string][]> => [['c-1', 'Авто плюс · Узбекистан']]);
    const { submit, input } = renderDialog({ ...base, options: [], search });
    await waitFor(() => expect(suggestions(input)).toEqual(['Авто плюс · Узбекистан']));

    fireEvent.change(input, { target: { value: 'Авто плюс · Узбекистан' } }); // picked from the list
    await new Promise((r) => setTimeout(r, 300));
    expect(search).toHaveBeenCalledTimes(1); // no search for "Name · Country"
    expect(screen.queryByText('Ничего не найдено')).toBeNull();

    fireEvent.change(input, { target: { value: 'авто плюс' } }); // typed the name only
    await waitFor(() => expect(search).toHaveBeenLastCalledWith('авто плюс'));
    fireEvent.click(screen.getByRole('button', { name: 'Сохранить' }));
    await waitFor(() => expect(submit).toHaveBeenCalledWith(expect.objectContaining({ companyId: 'c-1' })));
  });
});
