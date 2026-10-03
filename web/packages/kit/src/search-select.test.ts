// @vitest-environment jsdom
import { createElement } from 'react';
import { act, cleanup, fireEvent, render, screen, waitFor } from '@testing-library/react';
import { afterEach, describe, expect, it, vi } from 'vitest';
import { FormDialog } from './ui';
import type { FieldSpec } from './ui';

afterEach(() => { cleanup(); vi.useRealTimers(); });

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

const suggestions = (input: HTMLInputElement) => {
  fireEvent.focus(input);
  return screen.queryAllByRole('option').map((o) => o.textContent);
};

function renderDialog(field: FieldSpec) {
  const submit = vi.fn().mockResolvedValue(undefined);
  render(createElement(FormDialog, { title: 'Партнёр', fields: [field], onSubmit: submit, onClose: vi.fn() }));
  return { submit, input: screen.getByLabelText('Компания') as HTMLInputElement };
}

describe('searchable select', () => {
  it('suggests the option labels and submits the chosen option value', async () => {
    const { submit, input } = renderDialog(base);
    expect(suggestions(input)).toEqual(['Авто плюс · Узбекистан', 'Банк Один · Казахстан']);
    fireEvent.click(screen.getByRole('option', { name: 'Банк Один · Казахстан' }));
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

    fireEvent.click(screen.getByRole('option', { name: 'Лиз найдено' }));
    fireEvent.click(screen.getByRole('button', { name: 'Сохранить' }));
    await waitFor(() => expect(submit).toHaveBeenCalledWith(expect.objectContaining({ companyId: 'c-9' })));
  });

  it('selects by name alone and does not search again after a suggestion is picked', async () => {
    const search = vi.fn(async (): Promise<[string, string][]> => [['c-1', 'Авто плюс · Узбекистан']]);
    const { submit, input } = renderDialog({ ...base, options: [], search });
    await waitFor(() => expect(suggestions(input)).toEqual(['Авто плюс · Узбекистан']));

    fireEvent.click(screen.getByRole('option', { name: 'Авто плюс · Узбекистан' }));
    await new Promise((r) => setTimeout(r, 300));
    expect(search).toHaveBeenCalledTimes(1); // no search for "Name · Country"
    expect(screen.queryByText('Ничего не найдено')).toBeNull();

    fireEvent.change(input, { target: { value: 'авто плюс' } }); // typed the name only
    await waitFor(() => expect(search).toHaveBeenLastCalledWith('авто плюс'));
    fireEvent.click(screen.getByRole('button', { name: 'Сохранить' }));
    await waitFor(() => expect(submit).toHaveBeenCalledWith(expect.objectContaining({ companyId: 'c-1' })));
  });
});

function deferred() {
  let resolve!: (options: [string, string][]) => void;
  let reject!: (reason: Error) => void;
  const promise = new Promise<[string, string][]>((yes, no) => { resolve = yes; reject = no; });
  return { promise, resolve, reject };
}
const advance = (ms: number) => act(async () => { await vi.advanceTimersByTimeAsync(ms); });

it('does not guess an ID from duplicate exact labels and allows a real pick', async () => {
  const { submit, input } = renderDialog({ ...base, options: [['a', 'Same'], ['b', 'Same']] });
  fireEvent.change(input, { target: { value: 'Same' } });
  expect(input.checkValidity()).toBe(false);
  fireEvent.click(screen.getByRole('button', { name: 'Сохранить' }));
  expect(submit).not.toHaveBeenCalled();
  fireEvent.click(screen.getAllByRole('option', { name: 'Same' })[1]!);
  fireEvent.click(screen.getByRole('button', { name: 'Сохранить' }));
  await waitFor(() => expect(submit).toHaveBeenCalledWith({ companyId: 'b' }));
});

it('accepts an unambiguous case-insensitive exact label for compatibility', async () => {
  const { submit, input } = renderDialog(base);
  fireEvent.change(input, { target: { value: 'банк один · казахстан' } });
  fireEvent.click(screen.getByRole('button', { name: 'Сохранить' }));
  await waitFor(() => expect(submit).toHaveBeenCalledWith({ companyId: 'c-2' }));
});

it.each(['success', 'failure'] as const)('invalidates stale %s during the next debounce window', async (outcome) => {
  vi.useFakeTimers();
  const old = deferred();
  const latest = deferred();
  const search = vi.fn().mockReturnValueOnce(old.promise).mockReturnValueOnce(latest.promise);
  const { input } = renderDialog({ ...base, options: [], search });
  await advance(0);
  expect(search).toHaveBeenCalledWith('');
  fireEvent.change(input, { target: { value: 'New' } });
  await act(async () => {
    if (outcome === 'success') old.resolve([['stale', 'Stale']]);
    else old.reject(new Error('Stale failure'));
  });
  expect(screen.queryByRole('option', { name: 'Stale' })).toBeNull();
  expect(screen.queryByRole('alert')).toBeNull();
  await advance(250);
  await act(async () => latest.resolve([['fresh', 'Server synonym']]));
  // Server results are authoritative even when they do not contain the query.
  expect(screen.getByRole('option', { name: 'Server synonym' })).toBeTruthy();
  expect(screen.queryByText('Поиск…')).toBeNull();
});

it.each(['success', 'failure'] as const)('invalidates in-flight %s on a pick and retains its label', async (outcome) => {
  vi.useFakeTimers();
  const pending = deferred();
  const search = vi.fn().mockReturnValue(pending.promise);
  const { input } = renderDialog({ ...base, options: [['chosen', 'Chosen label']], search });
  await advance(0);
  fireEvent.focus(input);
  fireEvent.click(screen.getByRole('option', { name: 'Chosen label' }));
  await act(async () => {
    if (outcome === 'success') pending.resolve([['late', 'Late page']]);
    else pending.reject(new Error('Late failure'));
  });
  expect(input.value).toBe('Chosen label');
  fireEvent.click(input);
  expect(screen.getByRole('option', { name: 'Chosen label' }).getAttribute('aria-selected')).toBe('true');
  expect(screen.queryByRole('option', { name: 'Late page' })).toBeNull();
  expect(screen.queryByRole('alert')).toBeNull();
  await advance(300);
  expect(search).toHaveBeenCalledTimes(1);
});

it('shows read errors separately and retries the current query', async () => {
  vi.useFakeTimers();
  const search = vi.fn().mockResolvedValueOnce([]).mockRejectedValueOnce(new Error('Read failed'))
    .mockResolvedValueOnce([['retried', 'Recovered']]);
  const { input } = renderDialog({ ...base, options: [], search });
  await advance(0);
  fireEvent.focus(input);
  expect(screen.getByText('Ничего не найдено')).toBeTruthy();
  fireEvent.change(input, { target: { value: 'query' } });
  expect(screen.getByText('Поиск…')).toBeTruthy();
  await advance(250);
  expect(screen.getByRole('alert').textContent).toContain('Read failed');
  expect(screen.queryByText('Ничего не найдено')).toBeNull();
  fireEvent.click(screen.getByRole('button', { name: 'Повторить' }));
  await advance(250);
  expect(search).toHaveBeenLastCalledWith('query');
  expect(screen.getByRole('option', { name: 'Recovered' })).toBeTruthy();
});

it.each(['success', 'failure'] as const)('ignores pending %s after unmount without leaking into another field', async (outcome) => {
  vi.useFakeTimers();
  const pending = deferred();
  const search = vi.fn().mockReturnValue(pending.promise);
  renderDialog({ ...base, options: [], search });
  await advance(0);
  cleanup();
  const { input } = renderDialog({ ...base, initial: 'c-1' });
  await act(async () => {
    if (outcome === 'success') pending.resolve([['old', 'Old field']]);
    else pending.reject(new Error('Old failure'));
  });
  expect(input.value).toBe('Авто плюс · Узбекистан');
  expect(screen.queryByRole('alert')).toBeNull();
});

it('keeps a captured selected label when a remote first page omits that ID', async () => {
  const { input } = renderDialog({ ...base, initial: 'c-1', search: async () => [['other', 'Other page']] });
  fireEvent.focus(input);
  await waitFor(() => expect(screen.getByRole('option', { name: 'Other page' })).toBeTruthy());
  expect(input.value).toBe('Авто плюс · Узбекистан');
});

it('uses options that arrive after the field mounts', () => {
  const props = { title: 'Company', onSubmit: vi.fn(), onClose: vi.fn() };
  const view = render(createElement(FormDialog, { ...props, fields: [{ ...base, initial: 'late', options: [] }] }));
  view.rerender(createElement(FormDialog, { ...props, fields: [{ ...base, initial: 'late', options: [['late', 'Late label']] }] }));
  expect((screen.getByLabelText('Компания') as HTMLInputElement).value).toBe('Late label');
});
