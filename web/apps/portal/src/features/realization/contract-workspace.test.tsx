// @vitest-environment jsdom
import { QueryClient, QueryClientProvider } from '@tanstack/react-query';
import { cleanup, fireEvent, render, screen, waitFor } from '@testing-library/react';
import { afterEach, describe, expect, it, vi } from 'vitest';
import { ContractWorkspace } from './contract-workspace';

const json = (body: unknown) => new Response(JSON.stringify(body), { headers: { 'Content-Type': 'application/json' } });
afterEach(() => {
  cleanup();
  vi.unstubAllGlobals();
});
function scene(owner: 'retail' | 'commerce', contracts: unknown[] = []) {
  const requests: { url: string; method: string; body?: Record<string, unknown> }[] = [];
  vi.stubGlobal(
    'fetch',
    vi.fn(async (input: RequestInfo | URL, init: RequestInit = {}) => {
      const url = String(input);
      const method = init.method ?? 'GET';
      if (method !== 'GET') requests.push({ url, method, body: JSON.parse(String(init.body)) });
      if (url.includes('contract-templates')) return json({ items: [] });
      if (url.includes('/contracts')) return method === 'GET' ? json({ items: contracts }) : json({ data: {} });
      return json({ data: {} });
    }),
  );
  const client = new QueryClient({ defaultOptions: { queries: { retry: false } } });
  render(
    <QueryClientProvider client={client}>
      <ContractWorkspace
        owner={owner}
        id="deal-1"
        revision="7"
        price={{ amountMinor: '9999', currency: 'USD' }}
        refresh={[]}
        canWrite
      />
    </QueryClientProvider>,
  );
  return requests;
}
describe('contract workspace', () => {
  it('uses bindingIds when a commerce draft is prepared, despite commerce reading fileIds', async () => {
    const requests = scene('commerce');
    fireEvent.click(await screen.findByRole('button', { name: 'Подготовить договор' }));
    fireEvent.change(screen.getByLabelText('Текст договора'), { target: { value: 'Buyer {{buyer.name}}' } });
    fireEvent.click(screen.getAllByRole('button', { name: 'Подготовить договор' })[1]!);
    await waitFor(() => expect(requests).toHaveLength(1));
    expect(requests[0]).toMatchObject({
      method: 'POST',
      url: '/api/v1/commerce/orders/deal-1/contracts',
      body: { content: 'Buyer {{buyer.name}}', bindingIds: [] },
    });
    expect(requests[0]!.body).not.toHaveProperty('fileIds');
  });
  it('shows signed history as immutable snapshot and never substitutes the live price', async () => {
    scene('retail', [
      {
        id: 'signed-1',
        state: 'signed',
        kind: 'base',
        content: 'Цена в подписанной версии: 10.00',
        reference: 'R-1',
        signedOn: '2026-10-01',
        bindingIds: ['scan-1'],
        contractRevision: '2',
        partySnapshot: {
          seller: { name: 'Продавец' },
          customer: { name: 'Покупатель' },
          vehicle: { vin: 'OLDVIN', model: 'Old car' },
        },
        termsSnapshot: { price: { amountMinor: '1000', currency: 'USD' }, paymentScheme: 'cash' },
      },
    ]);
    fireEvent.click(await screen.findByRole('button', { name: 'Открыть' }));
    expect(screen.getByText('Подписанная версия (только чтение)')).toBeTruthy();
    expect(screen.getAllByText(/10.00/)).toHaveLength(3);
    expect(screen.queryByRole('button', { name: 'Изменить черновик' })).toBeNull();
    expect(screen.getByText(/Подписанную версию нельзя заменить/)).toBeTruthy();
  });
  it('patches an existing retail draft with its revision and retains external bindings when no replacement was uploaded', async () => {
    const requests = scene('retail', [
      {
        id: 'draft-1',
        state: 'draft',
        kind: 'base',
        content: 'Старый текст',
        templateId: 'template-1',
        bindingIds: ['external-1'],
        contractRevision: '3',
      },
    ]);
    fireEvent.click(await screen.findByRole('button', { name: 'Открыть' }));
    fireEvent.click(screen.getByRole('button', { name: 'Изменить черновик' }));
    fireEvent.change(screen.getByLabelText('Текст договора'), { target: { value: 'Исправленный текст' } });
    fireEvent.click(screen.getAllByRole('button', { name: 'Изменить черновик' })[1]!);
    await waitFor(() => expect(requests).toHaveLength(1));
    expect(requests[0]).toMatchObject({
      method: 'PATCH',
      url: '/api/v1/retail/deals/deal-1/contracts/draft-1',
      body: { draftRevision: '3', body: 'Исправленный текст', bindingIds: ['external-1'] },
    });
  });
});
