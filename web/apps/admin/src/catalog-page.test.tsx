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
    fireEvent.change(within(dialog).getByLabelText(label), { target: { value } });
  }
}

describe('car catalog page', () => {
  it('lists the catalog models', async () => {
    stubApi();
    renderPage();
    expect(await screen.findByText('Chevrolet Cobalt')).toBeTruthy();
    expect(screen.getByText('LTZ')).toBeTruthy();
  });

  it('adds a model with a numeric year', async () => {
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
      Двигатель: '2.0 бензин',
      Привод: 'Передний',
      'Цвет кузова': 'Серый',
      'Цвет салона': 'Бежевый',
    });
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
        powertrain: '2.0 бензин',
        drivetrain: 'Передний',
        exteriorColor: 'Серый',
        interiorColor: 'Бежевый',
      },
    });
  });

  it('adds a specification version with the model revision', async () => {
    let written: Write | undefined;
    stubApi((w) => (written = w));
    renderPage();
    const row = (await screen.findByText('Chevrolet Cobalt')).closest('tr')!;
    fireEvent.click(within(row).getByRole('button', { name: 'Новая версия' }));
    const dialog = await screen.findByRole('dialog', { name: 'Новая версия характеристик' });
    fill(dialog, { Двигатель: '1.5 гибрид' });
    fireEvent.click(within(dialog).getByRole('button', { name: 'Сохранить' }));

    await waitFor(() => expect(written).toBeDefined());
    expect(written!.url).toContain('/inventory/vehicle-models/m-1/specification-versions');
    expect(written!.headers.get('If-Match')).toContain('3');
    expect(written!.body).toMatchObject({ specification: { make: 'Chevrolet', year: 2025, powertrain: '1.5 гибрид' } });
  });
});
