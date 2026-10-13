import { createElement } from 'react';
import { cleanup, fireEvent, render, screen, within } from '@testing-library/react';
import { afterEach, describe, expect, it, vi } from 'vitest';

vi.mock('./data', () => ({
  modelName: (model: { label: string }) => model.label,
  routeLabel: { local: 'Местный', transit: 'В пути' },
  useOrderModels: () => ({ data: [{ id: 'm-1', label: 'Марка · Модель' }], isLoading: false, error: null, refetch: vi.fn() }),
}));

import { TermsDialog } from './shared';

afterEach(cleanup);

function pick(dialog: HTMLElement, label: string, value: string) {
  const input = within(dialog).getByRole('combobox', { name: label });
  fireEvent.focus(input);
  fireEvent.change(input, { target: { value } });
  fireEvent.click(within(dialog).getByRole('option', { name: value }));
}

describe('TermsDialog autocomplete controls', () => {
  it('commits model IDs and selected extra, currency, route and exact payment amounts', async () => {
    const onSubmit = vi.fn(async () => undefined);
    render(createElement(TermsDialog, {
      title: 'Условия',
      onClose: vi.fn(),
      onSubmit,
      extra: [{ name: 'payment', label: 'Оплата', required: true, options: [['advance', 'Предоплата']] }],
    }));
    const dialog = screen.getByRole('dialog', { name: 'Условия' });
    pick(dialog, 'Модель 1', 'Марка · Модель');
    fireEvent.change(within(dialog).getByLabelText('Цена'), { target: { value: '25000.01' } });
    pick(dialog, 'Оплата', 'Предоплата');
    pick(dialog, 'Валюта', 'EUR');
    pick(dialog, 'Маршрут поставки', 'В пути');
    fireEvent.click(within(dialog).getByRole('button', { name: '+ Добавить платёж' }));
    fireEvent.change(within(dialog).getByLabelText('Сумма платежа'), { target: { value: '25000.01' } });
    fireEvent.change(within(dialog).getByLabelText('Дата платежа'), { target: { value: '2026-10-15' } });
    fireEvent.click(within(dialog).getByRole('button', { name: 'Сохранить' }));

    await vi.waitFor(() => expect(onSubmit).toHaveBeenCalledTimes(1));
    expect(onSubmit).toHaveBeenCalledWith({
      lines: [{ modelId: 'm-1', quantity: '1', unitPrice: { amountMinor: '2500001', currency: 'EUR' } }],
      route: 'transit', deliveryTerms: '', warrantyTerms: '', serviceTerms: '',
      paymentSchedule: [{ amount: { amountMinor: '2500001', currency: 'EUR' }, dueDate: '2026-10-15' }],
    }, { payment: 'advance' });
  });

  it('rejects blank or typed-uncommitted required selections and keeps the modal open on first Escape', () => {
    const onSubmit = vi.fn(async () => undefined);
    const onClose = vi.fn();
    render(createElement(TermsDialog, { title: 'Условия', onClose, onSubmit }));
    const dialog = screen.getByRole('dialog', { name: 'Условия' });
    const model = within(dialog).getByRole('combobox', { name: 'Модель 1' });
    fireEvent.focus(model);
    fireEvent.change(model, { target: { value: 'Марка · Модель' } });
    fireEvent.click(within(dialog).getByRole('button', { name: 'Сохранить' }));
    expect(within(dialog).getByText('Выберите модель')).toBeTruthy();
    expect(onSubmit).not.toHaveBeenCalled();

    const currency = within(dialog).getByRole('combobox', { name: 'Валюта' });
    fireEvent.change(currency, { target: { value: 'X' } });
    fireEvent.keyDown(currency, { key: 'Escape' });
    expect(onClose).not.toHaveBeenCalled();
    fireEvent.click(within(dialog).getByRole('button', { name: 'Сохранить' }));
    expect(within(dialog).getByText('Выберите валюту и маршрут поставки.')).toBeTruthy();
  });
});
