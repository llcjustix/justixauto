import { createElement } from 'react';
import { QueryClient, QueryClientProvider } from '@tanstack/react-query';
import { cleanup, fireEvent, render, screen, waitFor, within } from '@testing-library/react';
import { MemoryRouter } from 'react-router-dom';
import { afterEach, describe, expect, it, vi } from 'vitest';
import { PermissionsPage } from './pages';

afterEach(() => {
  cleanup();
  vi.unstubAllGlobals();
});

const response = (body: unknown) =>
  new Response(JSON.stringify(body), { status: 200, headers: { 'Content-Type': 'application/json' } });

const catalog = [
  { key: 'retail.read', scope: 'company', name: 'Продажи: просмотр', assignable: true },
  { key: 'platform.audit.read', scope: 'platform', name: 'Платформа: журнал действий', assignable: true },
];

function stubApi(onWrite?: (url: string, body: unknown, method?: string) => void) {
  vi.stubGlobal(
    'fetch',
    vi.fn(async (input: RequestInfo | URL, init?: RequestInit) => {
      const url = String(input);
      if (init?.method === 'POST' || init?.method === 'PATCH') {
        onWrite?.(url, JSON.parse(String(init.body)), init.method);
        return response({ data: catalog[0], revision: '1' });
      }
      if (url.includes('/identity/admin/permission-catalog')) return response({ items: catalog });
      throw new Error(`Unexpected request ${url} ${init?.method}`);
    }),
  );
}

function renderPage() {
  const client = new QueryClient({ defaultOptions: { queries: { retry: false } } });
  return render(
    createElement(QueryClientProvider, { client }, createElement(MemoryRouter, null, createElement(PermissionsPage))),
  );
}

describe('permissions page', () => {
  it('lists the catalog and filters by scope', async () => {
    stubApi();
    renderPage();
    expect(await screen.findByText('Продажи: просмотр')).toBeTruthy();
    fireEvent.change(screen.getByDisplayValue('Все области'), { target: { value: 'platform' } });
    await waitFor(() => expect(screen.queryByText('Продажи: просмотр')).toBeNull());
    expect(screen.getByText('Платформа: журнал действий')).toBeTruthy();
  });

  it('adds a permission with key, name and scope', async () => {
    let written: { url: string; body: unknown } | undefined;
    stubApi((url, body) => (written = { url, body }));
    renderPage();
    await screen.findByText('Продажи: просмотр');

    fireEvent.click(screen.getByRole('button', { name: '+ Добавить разрешение' }));
    const dialog = await screen.findByRole('dialog', { name: 'Новое разрешение' });
    fireEvent.change(within(dialog).getByLabelText('Название'), { target: { value: 'Продажи: отчёты' } });
    fireEvent.change(within(dialog).getByLabelText('Ключ'), { target: { value: 'retail.reports.read' } });
    fireEvent.change(within(dialog).getByLabelText('Область'), { target: { value: 'company' } });
    fireEvent.click(within(dialog).getByRole('button', { name: 'Добавить' }));

    await waitFor(() => expect(written).toBeDefined());
    expect(written!.url).toContain('/identity/admin/permission-catalog');
    expect(written!.body).toMatchObject({
      name: 'Продажи: отчёты',
      key: 'retail.reports.read',
      scope: 'company',
      assignable: true,
    });
  });

  it('renames a permission by its key', async () => {
    let written: { url: string; body: unknown; method?: string } | undefined;
    stubApi((url, body, method) => (written = { url, body, method }));
    renderPage();
    const row = (await screen.findByText('Продажи: просмотр')).closest('tr')!;
    fireEvent.click(within(row).getByRole('button', { name: 'Изменить' }));
    const dialog = await screen.findByRole('dialog', { name: 'Изменить разрешение' });
    fireEvent.change(within(dialog).getByLabelText('Название'), { target: { value: 'Продажи: чтение' } });
    fireEvent.click(within(dialog).getByRole('button', { name: 'Сохранить' }));

    await waitFor(() => expect(written).toBeDefined());
    expect(written!.method).toBe('PATCH');
    expect(written!.url).toContain('/identity/admin/permission-catalog/retail.read');
    expect(written!.body).toMatchObject({ name: 'Продажи: чтение', assignable: true });
  });
});
