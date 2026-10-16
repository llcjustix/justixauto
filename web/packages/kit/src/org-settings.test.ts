// @vitest-environment jsdom
import { createElement } from 'react';
import { QueryClient, QueryClientProvider } from '@tanstack/react-query';
import { cleanup, render, screen } from '@testing-library/react';
import { afterEach, describe, expect, it, vi } from 'vitest';
import { OrgSettings } from './org-settings';

const session = {
  company: { id: 'company-1', kind: 'bank' },
  view: { user: { displayName: 'Оператор' }, roles: [{ name: 'Менеджер' }] },
  refresh: vi.fn(async () => undefined),
};

vi.mock('./session', () => ({ ChangePassword: () => null, useSession: () => session }));
vi.mock('./company-employees', () => ({ CompanyEmployees: () => createElement('div', null, 'Сотрудники') }));
vi.mock('./company-roles', () => ({ CompanyRoles: () => createElement('div', null, 'Роли компании') }));

afterEach(() => {
  cleanup();
  vi.unstubAllGlobals();
});

function renderSettings(legalForm: string | undefined) {
  vi.stubGlobal(
    'fetch',
    vi.fn(
      async () =>
        new Response(
          JSON.stringify({
            data: {
              name: 'Партнёр',
              legalName: '',
              legalForm,
              country: { label: 'Казахстан' },
              region: null,
              registration: '',
              email: '',
              phone: '',
              address: '',
              access: 'active',
              userType: null,
              verification: 'verified',
              verificationReason: '',
            },
          }),
          { status: 200, headers: { 'Content-Type': 'application/json' } },
        ),
    ),
  );
  return render(
    createElement(
      QueryClientProvider,
      { client: new QueryClient({ defaultOptions: { queries: { retry: false } } }) },
      createElement(OrgSettings),
    ),
  );
}

describe('provider organization legal form', () => {
  it.each([
    ['bank', 'llc', 'ООО'],
    ['mfo', 'sole_proprietor', 'ИП'],
    ['insurance', 'other', 'Другое'],
    ['bank', undefined, 'Не указана'],
    ['insurance', 'unsupported', 'Не указана'],
  ])('shows %s legal form as %s', async (kind, legalForm, expected) => {
    session.company.kind = kind;
    renderSettings(legalForm);

    expect(await screen.findByText(expected)).toBeTruthy();
    expect(screen.queryByRole('button', { name: /изменить/i })).toBeNull();
    expect(screen.queryByLabelText('Организационная форма')).toBeNull();
  });
});
