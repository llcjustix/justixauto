import { createElement } from 'react';
import { QueryClient, QueryClientProvider } from '@tanstack/react-query';
import { cleanup, fireEvent, render, screen, waitFor, within } from '@testing-library/react';
import { MemoryRouter } from 'react-router-dom';
import { afterEach, describe, expect, it, vi } from 'vitest';
import { CompaniesPage } from './pages';
import { SessionGate } from '@justixauto/kit';
import type { Company } from '@justixauto/kit';

afterEach(() => {
  cleanup();
  vi.unstubAllGlobals();
});

const company = {
  id: 'company-1',
  kind: 'seller',
  name: 'Авто плюс',
  legalName: 'АО Старое имя',
  legalForm: 'llc',
  country: { label: 'Казахстан' },
  region: { label: 'Алматы' },
  registration: '12345',
  email: 'old@example.test',
  address: 'Алматы',
  phone: '+7 700 000 00 00',
  access: 'draft',
  accessReason: '',
  revision: '7',
};

function renderCompanies(kind: Company['kind']) {
  const fetchCompany = globalThis.fetch;
  vi.stubGlobal('fetch', (input: RequestInfo | URL, init?: RequestInit) =>
    String(input).endsWith('/identity/session')
      ? Promise.resolve(
          response({
            data: {
              user: { id: 'admin', displayName: 'Admin', status: 'active', passwordChangeRequired: false },
              roles: [],
              permissions: ['platform.companies.access'],
              context: { revision: '1', companyId: null, branchScope: { mode: 'ALL', branchIds: [] } },
              accessibleCompanies: [],
              setup: { next: 'none' },
            },
          }),
        )
      : fetchCompany(input, init),
  );
  const client = new QueryClient({ defaultOptions: { queries: { retry: false } } });
  return render(
    createElement(
      QueryClientProvider,
      { client },
      createElement(
        MemoryRouter,
        null,
        createElement(SessionGate, { title: 'Admin', children: createElement(CompaniesPage, { kind }) }),
      ),
    ),
  );
}

const response = (body: unknown) =>
  new Response(JSON.stringify(body), { status: 200, headers: { 'Content-Type': 'application/json' } });

function fillCreationForm() {
  fireEvent.change(screen.getByLabelText('Название компании'), { target: { value: 'Новая компания' } });
  fireEvent.change(screen.getByLabelText('Страна'), { target: { value: 'Казахстан' } });
  fireEvent.change(screen.getByLabelText('Электронная почта'), { target: { value: 'admin@example.test' } });
  fireEvent.change(screen.getByLabelText('Логин'), { target: { value: 'first-admin' } });
  fireEvent.change(screen.getByLabelText('Пароль'), { target: { value: 'password' } });
  fireEvent.change(screen.getByLabelText('Повторите пароль'), { target: { value: 'password' } });
}

describe('company onboarding form', () => {
  it.each([
    ['seller', '/identity/admin/seller-companies', 'llc', 'ООО'],
    ['bank', '/identity/admin/provider-companies', 'sole_proprietor', 'ИП'],
    ['mfo', '/identity/admin/provider-companies', 'other', 'Другое'],
    ['insurance', '/identity/admin/provider-companies', 'llc', 'ООО'],
  ] as const)(
    'submits the selected legal form to the company endpoint for %s',
    async (kind, endpoint, legalForm, label) => {
      const fetch = vi.fn(async (input: RequestInfo | URL, init?: RequestInit) => {
        if (String(input).includes('/admin/companies')) return response({ items: [] });
        if (String(input).includes(endpoint)) return response({ data: {}, revision: '1' });
        throw new Error(`Unexpected request ${String(input)} ${init?.method}`);
      });
      vi.stubGlobal('fetch', fetch);
      renderCompanies(kind);

      fireEvent.click(
        await screen.findByRole('button', {
          name:
            kind === 'seller'
              ? '+ Добавить компанию'
              : `+ Подключить: ${kind === 'bank' ? 'Банк' : kind === 'mfo' ? 'МФО' : 'Страховая'}`,
        }),
      );
      expect(
        screen
          .getAllByRole('group')
          .filter((group) => group.classList.contains('form-fieldset'))
          .map((group) => group.querySelector(':scope > legend')?.textContent),
      ).toEqual(['Информация о компании', 'Адрес', 'Контакты', 'Реквизиты']);
      // Login fields live in the company section; the contact e-mail is the company's only.
      expect(screen.getAllByRole('group')[0]!.contains(screen.getByLabelText('Логин'))).toBe(true);
      expect(screen.getAllByLabelText('Электронная почта')).toHaveLength(1);
      expect((screen.getByLabelText('Организационная форма') as HTMLInputElement).value).toBe('Не указана');
      if (kind === 'seller') {
        expect((screen.getByRole('radio', { name: 'Дилер' }) as HTMLInputElement).checked).toBe(true);
        expect((screen.getByRole('radio', { name: 'Дистрибьютор' }) as HTMLInputElement).checked).toBe(false);
        fireEvent.click(screen.getByRole('radio', { name: 'Дистрибьютор' }));
        expect((screen.getByRole('radio', { name: 'Дилер' }) as HTMLInputElement).checked).toBe(false);
      } else {
        expect(screen.queryByRole('radio', { name: 'Дилер' })).toBeNull();
        expect(screen.queryByText('Бизнес-тип')).toBeNull();
      }
      fillCreationForm();
      fireEvent.change(screen.getByLabelText('Организационная форма'), { target: { value: label } });
      fireEvent.click(screen.getByRole('button', { name: 'Создать компанию' }));

      await waitFor(() => expect(fetch).toHaveBeenCalledWith(expect.stringContaining(endpoint), expect.any(Object)));
      const [, init] = fetch.mock.calls.find(([input]) => String(input).includes(endpoint))!;
      const body = JSON.parse(String(init?.body));
      expect(body.company).toMatchObject({
        name: 'Новая компания',
        email: 'admin@example.test',
        legalName: '',
        legalForm,
      });
      expect(body.firstAdmin).toMatchObject({ email: '', login: 'first-admin' });
      expect(body.firstAdmin).not.toHaveProperty('adminEmail');
      if (kind === 'seller') expect(body.userType).toBe('distributor');
      if (kind !== 'seller') expect(body.kind).toBe(kind);
    },
  );

  it('only requires the company name, admin login and password to create a seller company', async () => {
    const fetch = vi.fn(async (input: RequestInfo | URL, init?: RequestInit) => {
      if (String(input).includes('/admin/companies')) return response({ items: [] });
      if (String(input).includes('/admin/seller-companies')) return response({ data: {}, revision: '1' });
      throw new Error(`Unexpected request ${String(input)} ${init?.method}`);
    });
    vi.stubGlobal('fetch', fetch);
    renderCompanies('seller');

    fireEvent.click(await screen.findByRole('button', { name: '+ Добавить компанию' }));
    expect(screen.queryByLabelText('Регистрационный номер (ИНН/БИН)')).toBeNull();

    fireEvent.change(screen.getByLabelText('Название компании'), { target: { value: 'Минимальная компания' } });
    fireEvent.change(screen.getByLabelText('Логин'), { target: { value: 'bare-admin' } });
    fireEvent.change(screen.getByLabelText('Пароль'), { target: { value: 'password' } });
    fireEvent.change(screen.getByLabelText('Повторите пароль'), { target: { value: 'password' } });
    fireEvent.click(screen.getByRole('button', { name: 'Создать компанию' }));

    await waitFor(() =>
      expect(fetch).toHaveBeenCalledWith(expect.stringContaining('/admin/seller-companies'), expect.any(Object)),
    );
    const [, init] = fetch.mock.calls.find(([input]) => String(input).includes('/admin/seller-companies'))!;
    const body = JSON.parse(String(init?.body));
    expect(body.company).toMatchObject({ name: 'Минимальная компания', registration: '', legalForm: '' });
    expect(body.userType).toBe('dealer');
    expect(body.firstAdmin).toMatchObject({ login: 'bare-admin' });
  });

  it('hides the registration sub-line when a company has no registration number', async () => {
    const noRegistration = { ...company, registration: '' };
    const fetch = vi.fn(async (input: RequestInfo | URL, init?: RequestInit) => {
      if (String(input).includes('/admin/companies')) return response({ items: [noRegistration] });
      throw new Error(`Unexpected request ${String(input)} ${init?.method}`);
    });
    vi.stubGlobal('fetch', fetch);
    renderCompanies('seller');

    await screen.findByText('Авто плюс');
    expect(screen.queryByText('Реализация')?.textContent).toBe('Реализация');
    expect(screen.queryByText(/·\s*Реализация/)).toBeNull();
  });

  it('shows, prefills and preserves or clears legal form when editing displayed requisites', async () => {
    const fetch = vi.fn(async (input: RequestInfo | URL, init?: RequestInit) => {
      const url = String(input);
      if (url.includes('/admin/companies/company-1/users')) {
        return response({
          items: [
            {
              id: 'user-1',
              displayName: 'Первый админ',
              email: '',
              login: 'first-admin',
              status: 'active',
              roles: [],
              revision: '3',
              createdAt: '2026-01-01T00:00:00Z',
            },
          ],
        });
      }
      if (url.includes('/admin/companies')) return response({ items: [company] });
      if (url.endsWith('/identity/companies/company-1')) {
        if (init?.method === 'PATCH') return response({ data: company, revision: '8' });
        return response({ data: company, revision: '7' });
      }
      if (url.endsWith('/admin/users/user-1') && init?.method === 'PATCH') return response({ data: {}, revision: '4' });
      throw new Error(`Unexpected request ${url} ${init?.method}`);
    });
    vi.stubGlobal('fetch', fetch);
    renderCompanies('seller');

    fireEvent.click(await screen.findByText('Авто плюс'));
    expect(await screen.findByText('Организационная форма')).toBeTruthy();
    expect(screen.getByText('ООО')).toBeTruthy();
    fireEvent.click(await screen.findByRole('button', { name: 'Изменить' }));
    const dialog = await screen.findByRole('dialog', { name: 'Изменить: Авто плюс' });
    // The administrator's login data is prefilled in the same dialog.
    await waitFor(() => expect((within(dialog).getByLabelText('Логин') as HTMLInputElement).value).toBe('first-admin'));
    expect((within(dialog).getByLabelText('Организационная форма') as HTMLInputElement).value).toBe('ООО');
    fireEvent.change(within(dialog).getByLabelText('Название компании'), { target: { value: 'Новое имя' } });
    fireEvent.change(within(dialog).getByLabelText('Электронная почта'), { target: { value: 'new@example.test' } });
    fireEvent.change(within(dialog).getByLabelText('Логин'), { target: { value: 'renamed-admin' } });
    fireEvent.change(within(dialog).getByLabelText('Организационная форма'), { target: { value: 'Не указана' } });
    fireEvent.click(within(dialog).getByRole('button', { name: 'Изменить' }));

    await waitFor(() =>
      expect(fetch).toHaveBeenCalledWith(expect.stringContaining('/identity/companies/company-1'), expect.any(Object)),
    );
    const [, init] = fetch.mock.calls.find(([, request]) => request?.method === 'PATCH')!;
    expect(JSON.parse(String(init?.body))).toMatchObject({
      name: 'Новое имя',
      email: 'new@example.test',
      legalName: 'АО Старое имя',
      registration: '12345',
      legalForm: '',
    });
    expect(new Headers(init?.headers).get('If-Match')).toBe('"7"');
    await waitFor(() =>
      expect(fetch).toHaveBeenCalledWith(expect.stringContaining('/admin/users/user-1'), expect.any(Object)),
    );
    const [, userInit] = fetch.mock.calls.find(([input]) => String(input).endsWith('/admin/users/user-1'))!;
    expect(JSON.parse(String(userInit?.body))).toMatchObject({
      displayName: 'Первый админ',
      login: 'renamed-admin',
      roleIds: [],
    });
    expect(new Headers(userInit?.headers).get('If-Match')).toBe('"3"');
    expect(fetch.mock.calls.some(([input]) => String(input).includes('/password'))).toBe(false);
  });

  it('soft-deletes a company with a reason and closes its card', async () => {
    let deleted = false;
    const fetch = vi.fn(async (input: RequestInfo | URL, init?: RequestInit) => {
      const url = String(input);
      if (url.endsWith('/identity/admin/companies/company-1/delete') && init?.method === 'POST') {
        deleted = true;
        return response({ data: company, revision: '8' });
      }
      if (url.includes('/admin/companies')) return response({ items: deleted ? [] : [company] });
      if (url.endsWith('/identity/companies/company-1')) return response({ data: company, revision: '7' });
      throw new Error(`Unexpected request ${url} ${init?.method}`);
    });
    vi.stubGlobal('fetch', fetch);
    renderCompanies('seller');

    fireEvent.click(await screen.findByText('Авто плюс'));
    fireEvent.click(await screen.findByRole('button', { name: 'Удалить' }));
    const dialog = await screen.findByRole('dialog', { name: 'Удалить' });
    fireEvent.change(within(dialog).getByLabelText('Основание'), { target: { value: 'Дубликат' } });
    fireEvent.click(within(dialog).getByRole('button', { name: 'Удалить' }));

    await waitFor(() => expect(deleted).toBe(true));
    const [, init] = fetch.mock.calls.find(([url]) => String(url).endsWith('/delete'))!;
    expect(JSON.parse(String(init?.body))).toEqual({ reason: 'Дубликат' });
    expect(new Headers(init?.headers).get('If-Match')).toBe('"7"');
    await waitFor(() => expect(screen.queryByText('Авто плюс')).toBeNull());
  });
});
