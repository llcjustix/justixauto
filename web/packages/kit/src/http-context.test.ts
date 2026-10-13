import { afterEach, expect, it, vi } from 'vitest';
import { admitHttpContext, call, configureHttp, get, retireHttpContext, setCsrf } from './http';

function deferred<T>() {
  let resolve!: (value: T) => void;
  const promise = new Promise<T>((done) => { resolve = done; });
  return { promise, resolve };
}

const response = (status: number, body: unknown, token?: string) => new Response(JSON.stringify(body), {
  status,
  headers: token ? { 'X-CSRF-Token': token } : {},
});

afterEach(() => {
  vi.unstubAllGlobals();
  retireHttpContext();
  setCsrf('');
});

it('rejects late tenant data and leaves the live CSRF token untouched', async () => {
  admitHttpContext();
  setCsrf('old');
  const old = deferred<Response>();
  const fetch = vi.fn()
    .mockImplementationOnce(() => old.promise)
    .mockResolvedValueOnce(response(200, { data: { label: 'B' } }, 'new'))
    .mockResolvedValueOnce(response(200, { data: null }));
  vi.stubGlobal('fetch', fetch);
  const pending = get<{ label: string }>('/records');
  retireHttpContext();
  await expect(get('/records')).rejects.toMatchObject({ code: 'context_retired' });
  admitHttpContext();
  expect((await get<{ label: string }>('/records')).data.label).toBe('B');
  old.resolve(response(200, { data: { label: 'A' } }, 'late-old'));
  await expect(pending).rejects.toMatchObject({ code: 'context_retired' });
  await call('POST', '/records', {});
  expect((fetch.mock.calls[2]![1] as RequestInit).headers).toMatchObject({});
  expect(((fetch.mock.calls[2]![1] as RequestInit).headers as Headers).get('X-CSRF-Token')).toBe('new');
});

it('does not let a late 401 evict the new session', async () => {
  admitHttpContext();
  const old = deferred<Response>();
  const unauthenticated = vi.fn();
  const dispose = configureHttp({ onUnauthenticated: unauthenticated });
  vi.stubGlobal('fetch', vi.fn().mockImplementationOnce(() => old.promise));
  const pending = get('/records');
  retireHttpContext();
  admitHttpContext();
  old.resolve(response(401, { error: { code: 'unauthorized' } }, 'late'));
  await expect(pending).rejects.toMatchObject({ code: 'context_retired' });
  expect(unauthenticated).not.toHaveBeenCalled();
  dispose();
});

it('checks retirement after an awaited response body and preserves uncertain mutation errors', async () => {
  admitHttpContext();
  const body = deferred<unknown>();
  vi.stubGlobal('fetch', vi.fn()
    .mockResolvedValueOnce({ status: 200, ok: true, headers: new Headers({ 'X-CSRF-Token': 'stale' }), json: () => body.promise })
    .mockRejectedValueOnce(new Error('offline')));
  const pending = get('/records');
  await Promise.resolve();
  retireHttpContext();
  admitHttpContext();
  body.resolve({ data: { label: 'A' } });
  await expect(pending).rejects.toMatchObject({ code: 'context_retired' });
  await expect(call('POST', '/records', {})).rejects.toMatchObject({
    code: 'network', message: 'Нет ответа от сервера: результат неизвестен. Обновите данные перед повтором.',
  });
});
