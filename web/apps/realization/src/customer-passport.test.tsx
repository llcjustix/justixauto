// @vitest-environment jsdom
import { QueryClient, QueryClientProvider } from '@tanstack/react-query';
import { cleanup, fireEvent, render, screen, waitFor, within } from '@testing-library/react';
import { afterEach, expect, it, vi } from 'vitest';
import { CustomerPassportAction } from './customer-passport';
import type { Customer } from './data';

const json = (body: unknown, status = 200) => new Response(JSON.stringify(body), { status, headers: { 'Content-Type': 'application/json' } });
const customer: Customer = {
  id: 'customer-1', displayName: 'Алия', phone: '+998901234567', passportSeries: 'AB', passportNumber: '1234567', passportIssuer: 'ОВД',
  passportIssuedOn: '2020-01-02', passportExpiresOn: '2030-01-02', birthDate: '1990-01-01', personalId: '123', registeredAddress: 'Ташкент', passportCopyFileIds: ['old-copy'], revision: '7',
};

afterEach(() => { cleanup(); vi.unstubAllGlobals(); });

it('uploads a sensitive passport copy and sends the exact profile binding fields', async () => {
  const submitted = vi.fn().mockResolvedValue(undefined);
  vi.stubGlobal('fetch', vi.fn(async (input: RequestInfo | URL) => {
    expect(String(input)).toBe('/api/v1/documents/files');
    return json({ data: { id: 'new-copy', fileName: 'passport.pdf', mime: 'application/pdf', byteLength: 10, sensitive: true } }, 201);
  }));
  const client = new QueryClient({ defaultOptions: { queries: { retry: false } } });
  render(<QueryClientProvider client={client}><CustomerPassportAction label="Новый клиент" onSubmit={submitted} /></QueryClientProvider>);
  fireEvent.click(screen.getByRole('button', { name: 'Новый клиент' }));
  fireEvent.change(screen.getByLabelText('Имя'), { target: { value: 'Алия' } });
  fireEvent.change(screen.getByLabelText('Серия документа'), { target: { value: 'AB' } });
  fireEvent.change(screen.getByLabelText('Дата выдачи'), { target: { value: '2020-01-02' } });
  fireEvent.change(screen.getByLabelText('Копия документа'), { target: { files: [new File(['copy'], 'passport.pdf', { type: 'application/pdf' })] } });
  fireEvent.click(within(screen.getByRole('dialog')).getByRole('button', { name: 'Новый клиент' }));
  await waitFor(() => expect(submitted).toHaveBeenCalledWith(expect.objectContaining({
    displayName: 'Алия', passportSeries: 'AB', passportIssuedOn: '2020-01-02', passportCopyFileIds: ['new-copy'],
  })));
});

it('preserves saved copies and uses the authenticated download path while editing', async () => {
  const submitted = vi.fn().mockResolvedValue(undefined);
  const client = new QueryClient({ defaultOptions: { queries: { retry: false } } });
  render(<QueryClientProvider client={client}><CustomerPassportAction customer={customer} label="Изменить" onSubmit={submitted} /></QueryClientProvider>);
  fireEvent.click(screen.getByRole('button', { name: 'Изменить' }));
  expect(screen.getByRole('link', { name: 'Копия документа 1' }).getAttribute('href')).toBe('/api/v1/documents/files/old-copy/content');
  fireEvent.click(within(screen.getByRole('dialog')).getByRole('button', { name: 'Изменить' }));
  await waitFor(() => expect(submitted).toHaveBeenCalledWith(expect.objectContaining({
    passportSeries: 'AB', passportNumber: '1234567', passportCopyFileIds: ['old-copy'],
  })));
});
