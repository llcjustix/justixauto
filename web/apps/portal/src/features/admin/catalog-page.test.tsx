import { createElement } from 'react';
import { QueryClient, QueryClientProvider } from '@tanstack/react-query';
import { cleanup, fireEvent, render, screen, waitFor, within } from '@testing-library/react';
import { MemoryRouter } from 'react-router-dom';
import { afterEach, describe, expect, it, vi } from 'vitest';
import { CatalogPage } from './pages';

afterEach(() => {
  cleanup();
  vi.unstubAllGlobals();
});

const response = (body: unknown) =>
  new Response(JSON.stringify(body), { status: 200, headers: { 'Content-Type': 'application/json' } });

const spec = {
  make: 'Chevrolet',
  model: 'Cobalt',
  variant: 'LTZ',
  year: 2025,
  bodyType: 'Седан',
  exteriorColor: 'Белый',
  interiorColor: 'Чёрный',
  powertrain: '1.5 бензин',
  drivetrain: 'Передний',
  version: 1,
};
const models = [{ id: 'm-1', specification: spec, revision: '3' }];

interface Write {
  url: string;
  body: unknown;
  headers: Headers;
}

function stubApi(onWrite?: (w: Write) => void) {
  vi.stubGlobal(
    'fetch',
    vi.fn(async (input: RequestInfo | URL, init?: RequestInit) => {
      const url = String(input);
      if (init?.method === 'POST') {
        onWrite?.({ url, body: JSON.parse(String(init.body)), headers: new Headers(init.headers) });
        return response({ data: models[0], revision: '4' });
      }
      if (url.includes('/inventory/vehicle-models')) return response({ items: models });
      throw new Error(`Unexpected request ${url} ${init?.method}`);
    }),
  );
}

function renderPage() {
  const client = new QueryClient({ defaultOptions: { queries: { retry: false } } });
  return render(
    createElement(QueryClientProvider, { client }, createElement(MemoryRouter, null, createElement(CatalogPage))),
  );
}

function fill(dialog: HTMLElement, values: Record<string, string>) {
  for (const [label, value] of Object.entries(values)) {
    if (['Год', 'Кузов', 'Двигатель', 'Привод'].includes(label)) pick(dialog, label, value);
    else fireEvent.change(within(dialog).getByRole(label === 'Комплектация' ? 'textbox' : 'combobox', { name: label }), { target: { value } });
  }
}

function pick(dialog: HTMLElement, label: string, value: string) {
  const input = within(dialog).getByRole('combobox', { name: label });
  fireEvent.focus(input);
  fireEvent.change(input, { target: { value } });
  fireEvent.click(within(dialog).getByRole('option', { name: value }));
}

describe('car catalog page', () => {
  it('lists the catalog models', async () => {
    stubApi();
    renderPage();
    expect(await screen.findByText('Chevrolet Cobalt')).toBeTruthy();
    expect(screen.getByText('LTZ')).toBeTruthy();
  });

  it('adds a model with independent palette choices and custom labels', async () => {
    let written: Write | undefined;
    stubApi((w) => (written = w));
    renderPage();
    await screen.findByText('Chevrolet Cobalt');

    fireEvent.click(screen.getByRole('button', { name: '+ Добавить модель' }));
    const dialog = await screen.findByRole('dialog', { name: 'Новая модель' });
    fill(dialog, {
      Марка: 'Kia',
      Модель: 'K5',
      Комплектация: 'Prestige',
      Год: '2026',
      Кузов: 'Седан',
      Двигатель: 'Бензин',
      Привод: 'Передний',
    });
    fireEvent.click(within(dialog).getAllByLabelText('Серый')[0]!);
    fireEvent.click(within(dialog).getAllByLabelText('Бежевый')[1]!);
    const custom = within(dialog).getAllByLabelText('Свой цвет');
    fireEvent.change(custom[0]!, { target: { value: '  Матовый графит  ' } });
    fireEvent.click(within(dialog).getAllByRole('button', { name: 'Добавить цвет' })[0]!);
    fireEvent.change(custom[0]!, { target: { value: 'матовый графит' } });
    fireEvent.click(within(dialog).getAllByRole('button', { name: 'Добавить цвет' })[0]!);
    expect(within(dialog).getByText('Этот цвет уже выбран.')).toBeTruthy();
    fireEvent.click(within(dialog).getByRole('button', { name: 'Добавить' }));

    await waitFor(() => expect(written).toBeDefined());
    expect(written!.url).toMatch(/\/inventory\/vehicle-models$/);
    expect(written!.body).toEqual({
      specification: {
        make: 'Kia',
        model: 'K5',
        variant: 'Prestige',
        year: 2026,
        bodyType: 'Седан',
        powertrain: 'Бензин',
        drivetrain: 'Передний',
        exteriorColors: ['Серый', 'Матовый графит'],
        interiorColors: ['Бежевый'],
      },
    });
  });

  it('maps scalar fixtures to chips, removes them, and posts a new specification palette', async () => {
    let written: Write | undefined;
    stubApi((w) => (written = w));
    renderPage();
    const row = (await screen.findByText('Chevrolet Cobalt')).closest('tr')!;
    fireEvent.click(within(row).getByRole('button', { name: 'Новая версия' }));
    const dialog = await screen.findByRole('dialog', { name: 'Новая версия характеристик' });
    expect(within(dialog).getByRole('button', { name: 'Удалить Белый' })).toBeTruthy();
    expect(within(dialog).getByRole('button', { name: 'Удалить Чёрный' })).toBeTruthy();
    expect((within(dialog).getByRole('combobox', { name: 'Двигатель' }) as HTMLInputElement).value).toBe('1.5 бензин');
    fireEvent.click(within(dialog).getByRole('button', { name: 'Удалить Белый' }));
    fireEvent.click(within(dialog).getAllByLabelText('Синий')[0]!);
    fireEvent.click(within(dialog).getAllByLabelText('Бежевый')[1]!);
    fill(dialog, { Двигатель: 'Гибрид' });
    fireEvent.click(within(dialog).getByRole('button', { name: 'Сохранить' }));

    await waitFor(() => expect(written).toBeDefined());
    expect(written!.url).toContain('/inventory/vehicle-models/m-1/specification-versions');
    expect(written!.headers.get('If-Match')).toContain('3');
    expect(written!.body).toEqual({
      specification: {
        make: 'Chevrolet', model: 'Cobalt', variant: 'LTZ', year: 2025, bodyType: 'Седан',
        powertrain: 'Гибрид', drivetrain: 'Передний', exteriorColors: ['Синий'], interiorColors: ['Чёрный', 'Бежевый'],
      },
    });
  });

  it('requires a body and interior choice before submitting', async () => {
    let written: Write | undefined;
    stubApi((w) => (written = w));
    renderPage();
    await screen.findByText('Chevrolet Cobalt');
    fireEvent.click(screen.getByRole('button', { name: '+ Добавить модель' }));
    const dialog = await screen.findByRole('dialog', { name: 'Новая модель' });
    fill(dialog, { Марка: 'Kia', Модель: 'K5', Комплектация: 'Prestige', Год: '2026', Кузов: 'Седан', Двигатель: 'Бензин', Привод: 'Передний' });
    fireEvent.click(within(dialog).getByRole('button', { name: 'Добавить' }));
    expect(await within(dialog).findByText('Выберите хотя бы один цвет кузова и салона.')).toBeTruthy();
    expect(written).toBeUndefined();
  });

  it('keeps free custom make and model text while resetting the dependent model, and picks strict values by keyboard', async () => {
    stubApi();
    renderPage();
    await screen.findByText('Chevrolet Cobalt');
    fireEvent.click(screen.getByRole('button', { name: '+ Добавить модель' }));
    const dialog = await screen.findByRole('dialog', { name: 'Новая модель' });

    const make = within(dialog).getByRole('combobox', { name: 'Марка' });
    fireEvent.change(make, { target: { value: 'Своя марка' } });
    const model = within(dialog).getByRole('combobox', { name: 'Модель' });
    fireEvent.change(model, { target: { value: 'Своя модель' } });
    expect((model as HTMLInputElement).value).toBe('Своя модель');
    fireEvent.change(make, { target: { value: 'Другая марка' } });
    expect((model as HTMLInputElement).value).toBe('');

    const body = within(dialog).getByRole('combobox', { name: 'Кузов' });
    fireEvent.focus(body);
    fireEvent.change(body, { target: { value: 'Седан' } });
    fireEvent.keyDown(body, { key: 'ArrowDown' });
    fireEvent.keyDown(body, { key: 'Enter' });
    expect((body as HTMLInputElement).value).toBe('Седан');
  });
});
