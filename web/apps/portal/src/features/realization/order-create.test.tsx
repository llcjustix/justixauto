// @vitest-environment jsdom
import { createElement } from 'react';
import { QueryClient, QueryClientProvider } from '@tanstack/react-query';
import { act, cleanup, fireEvent, render, screen, waitFor, within } from '@testing-library/react';
import { afterEach, expect, it, vi } from 'vitest';
import type { Offer } from './data';
import { OrderCreate } from './order-create';

afterEach(() => { cleanup(); vi.unstubAllGlobals(); });
const response = (body: unknown, status = 200) => new Response(JSON.stringify(body), { status, headers: { 'Content-Type': 'application/json' } });
const model = (id = 'm') => {
  const specification = { version: '1', make: 'Make', model: id, variant: 'Sport', exteriorColor: 'White', interiorColor: 'Black' };
  return { id, specification, versions: [specification] };
};
const historicalModel = (id = 'm') => {
  const first = { ...model(id).specification, exteriorColor: '', interiorColor: '', exteriorColors: ['White', 'Blue'], interiorColors: ['Black', 'Tan'] };
  const current = { ...first, version: '2', exteriorColors: ['Red', 'Green'], interiorColors: ['Grey', 'Cream'] };
  return { id, specification: current, versions: [first, current] };
};
const partner = (id = 's', status = 'active') => ({ id: `p-${id}`, status, counterparty: { id, name: `Supplier ${id}` } });
const offer = {
  supplier: { name: 'Promotion supplier' },
  publishedVersion: {
    id: 'v1', total: { amountMinor: '1001', currency: 'EUR' },
    terms: {
      lines: [
        { lineId: 'l1', modelId: 'm', quantity: '1', unitPrice: { amountMinor: '1001', currency: 'EUR' } },
        { lineId: 'l2', modelId: 'm2', quantity: '1', unitPrice: { amountMinor: '2000', currency: 'EUR' } },
      ],
      route: 'factory', deliveryTerms: 'Delivery', warrantyTerms: 'Warranty', serviceTerms: 'Service',
      paymentSchedule: [{ amount: { amountMinor: '3001', currency: 'EUR' }, dueDate: '2026-12-01' }],
    },
  },
} as Offer;

function setup(options: {
  offer?: Offer;
  read?: (url: string) => Response | undefined | Promise<Response | undefined>;
  post?: (payload: unknown) => Promise<Response>;
} = {}) {
  const posts = vi.fn(options.post ?? (async () => response({ data: { id: 'created' } }, 201)));
  const fetcher = vi.fn(async (input: RequestInfo | URL, init?: RequestInit) => {
    const url = String(input);
    if (init?.method === 'POST') return posts(JSON.parse(String(init.body)));
    const custom = await options.read?.(url);
    if (custom) return custom;
    if (/\/vehicle-models\/[^?]+$/.test(url)) return response({ data: model(url.split('/').at(-1)) });
    return response({ items: url.includes('warehouses') ? [{ id: 'w', name: 'Warehouse', free: '9' }]
      : url.includes('vehicle-models') ? [model(), model('m2')]
        : url.includes('partnerships') ? [partner()] : [] });
  });
  vi.stubGlobal('fetch', fetcher);
  const client = new QueryClient({ defaultOptions: { queries: { retry: false } } });
  const invalidate = vi.spyOn(client, 'invalidateQueries');
  const created = vi.fn();
  const view = render(createElement(QueryClientProvider, { client }, createElement(OrderCreate, {
    mode: options.offer ? { kind: 'offer', offer: options.offer } : { kind: 'direct' }, onClose: vi.fn(), onCreated: created,
  })));
  return { posts, created, invalidate, fetcher, client, replaceOffer: (replacement: Offer) => view.rerender(createElement(QueryClientProvider, { client }, createElement(OrderCreate, {
    mode: { kind: 'offer', offer: replacement }, onClose: vi.fn(), onCreated: created,
  }))) };
}
const change = (label: string, value: string) => fireEvent.change(screen.getByLabelText(label, { selector: 'input,textarea' }), { target: { value } });
const pick = (label: string, name: string | RegExp) => {
  screen.getAllByRole('combobox').forEach(input => fireEvent.blur(input));
  fireEvent.focus(screen.getByRole('combobox', { name: label, exact: true }));
  fireEvent.click(screen.getByRole('option', { name, exact: true }));
};
const openOptions = (label: string) => {
  screen.getAllByRole('combobox').forEach(input => fireEvent.blur(input));
  fireEvent.focus(screen.getByRole('combobox', { name: label, exact: true }));
  return within(screen.getByRole('listbox'));
};
const next = () => fireEvent.click(screen.getByRole('button', { name: 'Далее' }));
const back = () => fireEvent.click(screen.getByRole('button', { name: 'Назад' }));
async function enterLines(isOffer = false, waitColors = true) {
  fireEvent.focus(screen.getByRole('combobox', { name: 'Склад получения' }));
  fireEvent.click(await screen.findByRole('option', { name: /Warehouse/ }));
  if (!isOffer) { fireEvent.focus(screen.getByRole('combobox', { name: 'Поставщик' })); fireEvent.click(await screen.findByRole('option', { name: 'Supplier s' })); }
  next();
  if (!isOffer) {
    fireEvent.focus(screen.getByRole('combobox', { name: 'Модель 1' }));
    fireEvent.click(await screen.findByRole('option', { name: 'Make m Sport' })); change('Цена', '10.01');
  }
  if (waitColors) await waitFor(() => expect((screen.getByLabelText('Цвет кузова 1') as HTMLInputElement).disabled).toBe(false));
}

it('identifies and focuses step errors; never clamps invalid direct quantities', async () => {
  const { posts } = setup();
  next();
  expect(screen.getByRole('alert').textContent).toContain('Выберите склад получения');
  expect(document.activeElement).toBe(screen.getByRole('alert'));
  await enterLines();
  fireEvent.click(screen.getByRole('button', { name: 'Уменьшить количество' }));
  expect((screen.getByLabelText('Количество') as HTMLInputElement).value).toBe('1');
  change('Количество', '10000');
  fireEvent.click(screen.getByRole('button', { name: 'Увеличить количество' }));
  expect((screen.getByLabelText('Количество') as HTMLInputElement).value).toBe('10000');
  for (const value of ['10001', '0', '1.5', '']) {
    change('Количество', value); next();
    expect(screen.getByRole('alert').textContent).toContain('целое число от 1 до 10000');
    expect((screen.getByLabelText('Количество') as HTMLInputElement).value).toBe(value);
  }
  expect(posts).not.toHaveBeenCalled();
});

it('preserves every commercial field on Back and sends exact EUR totals and schedule', async () => {
  const { posts, created, invalidate } = setup();
  await enterLines();
  change('Количество', '3'); change('Цена', '90071992547409.91');
  pick('Валюта', 'EUR'); pick('Маршрут поставки', 'Автомобиль в пути');
  change('Условия поставки', 'Delivery'); change('Гарантия', 'Warranty'); change('Сервис', 'Service');
  fireEvent.click(screen.getByRole('button', { name: '+ Добавить платёж' }));
  change('Сумма платежа', '270215977642229.73'); change('Дата платежа', '2026-12-01');
  back();
  expect((screen.getByLabelText('Поставщик') as HTMLInputElement).value).toBe('Supplier s');
  next();
  expect((screen.getByLabelText('Маршрут поставки') as HTMLInputElement).value).toBe('Автомобиль в пути');
  expect((screen.getByLabelText('Сумма платежа') as HTMLInputElement).value).toBe('270215977642229.73');
  next();
  expect(screen.getByText('270 215 977 642 229.73 EUR')).toBeTruthy();
  expect(screen.getByText('Make m Sport')).toBeTruthy();
  expect(screen.getByText('Поставщик: Supplier s')).toBeTruthy();
  expect(posts).not.toHaveBeenCalled();
  back(); next();
  fireEvent.click(screen.getByRole('button', { name: 'Отправить заказ' }));
  await waitFor(() => expect(created).toHaveBeenCalledWith('created'));
  expect(invalidate).toHaveBeenCalledWith({ queryKey: ['orders'] });
  expect(posts).toHaveBeenCalledWith({ supplierCompanyId: 's', warehouseId: 'w', terms: {
    lines: [{ modelId: 'm', quantity: '3', modelSpecificationVersion: '1', exteriorColor: 'White', interiorColor: 'Black', unitPrice: { amountMinor: '9007199254740991', currency: 'EUR' } }],
    route: 'in-transit', deliveryTerms: 'Delivery', warrantyTerms: 'Warranty', serviceTerms: 'Service',
    paymentSchedule: [{ amount: { amountMinor: '27021597764222973', currency: 'EUR' }, dueDate: '2026-12-01' }],
  } });
});

it('searches later catalog pages, retains labels on refresh and invalidates edited IDs', async () => {
  const { fetcher, client, posts } = setup({ read: url => {
    if (url.includes('partnerships')) return response({ items: url.includes('offset=0') ? Array.from({ length: 100 }, (_, i) => partner(`old${i}`, 'inactive')) : [partner('later')] });
    if (url.includes('vehicle-models?')) return response({ items: url.includes('offset=0') ? Array.from({ length: 100 }, (_, i) => model(`old${i}`)) : [model('later')] });
  } });
  expect(screen.getAllByRole('combobox')).toHaveLength(2);
  expect(screen.queryByLabelText('Поставщик: поиск')).toBeNull();
  fireEvent.focus(screen.getByLabelText('Поставщик')); change('Поставщик', 'later');
  fireEvent.click(await screen.findByRole('option', { name: 'Supplier later' }));
  await act(async () => { await client.invalidateQueries({ queryKey: ['partnerships', 'order-selector'] }); });
  expect((screen.getByLabelText('Поставщик') as HTMLInputElement).value).toBe('Supplier later');
  pick('Склад получения', /Warehouse/);
  change('Поставщик', 'no match'); next();
  expect(screen.getByRole('alert').textContent).toContain('Выберите поставщика');
  change('Поставщик', 'later'); pick('Поставщик', 'Supplier later'); next();
  fireEvent.focus(screen.getByLabelText('Модель 1')); change('Модель 1', 'later');
  fireEvent.click(await screen.findByRole('option', { name: 'Make later Sport' }));
  change('Модель 1', 'no match'); next();
  expect(screen.getByRole('alert').textContent).toContain('Выберите модель 1');
  expect(posts).not.toHaveBeenCalled();
  expect(fetcher.mock.calls.some(([url]) => String(url).includes('partnerships?limit=100&offset=100'))).toBe(true);
  expect(fetcher.mock.calls.some(([url]) => String(url).includes('vehicle-models?limit=100&offset=100'))).toBe(true);
});

it('retries required-data failures without losing another selected field', async () => {
  let fail = true;
  setup({ read: url => url.includes('partnerships') && fail ? response({ error: { code: 'unexpected', message: 'unavailable' } }, 500) : undefined });
  fireEvent.focus(screen.getByLabelText('Склад получения')); fireEvent.click(await screen.findByRole('option', { name: /Warehouse/ }));
  await screen.findByRole('button', { name: 'Повторить: поставщики' });
  fail = false; fireEvent.click(screen.getByRole('button', { name: 'Повторить: поставщики' }));
  fireEvent.focus(screen.getByLabelText('Поставщик')); await screen.findByRole('option', { name: 'Supplier s' });
  expect((screen.getByLabelText('Склад получения') as HTMLInputElement).value).toBe('Warehouse · свободно 9');
});

it('requires real currency, route and singleton color picks after editing before Next', async () => {
  const { posts } = setup(); await enterLines();
  change('Цена', '90071992547409.91');
  for (const [label, choice, error] of [
    ['Валюта', 'EUR', 'Выберите валюту'],
    ['Маршрут поставки', 'Автомобиль в пути', 'Выберите маршрут'],
    ['Цвет кузова 1', 'White', 'Выберите цвет'],
    ['Цвет салона 1', 'Black', 'Выберите цвет'],
  ]) {
    change(label!, 'unmatched'); next();
    expect(screen.getByRole('alert').textContent).toContain(error);
    expect(posts).not.toHaveBeenCalled();
    change(label!, ''); pick(label!, choice!);
  }
  expect((screen.getByLabelText('Цена') as HTMLInputElement).value).toBe('90071992547409.91');
  next(); fireEvent.click(screen.getByRole('button', { name: 'Отправить заказ' }));
  await waitFor(() => expect(posts).toHaveBeenCalledTimes(1));
  expect(posts.mock.calls[0]![0]).toMatchObject({ terms: { route: 'in-transit', lines: [{ exteriorColor: 'White', interiorColor: 'Black', unitPrice: { amountMinor: '9007199254740991', currency: 'EUR' } }] } });
});

it.each([false, true])('retains the direct draft after a server error (field error=%s)', async fieldError => {
  let fail = true;
  const { posts, created } = setup({ post: async () => fail
    ? response({ error: { code: 'validation', message: 'Please correct order', fields: fieldError ? { 'terms.paymentSchedule': 'Schedule must equal total' } : {} } }, 422)
    : response({ data: { id: 'created' } }, 201) });
  await enterLines(); change('Количество', '7'); next();
  fireEvent.click(screen.getByRole('button', { name: 'Отправить заказ' }));
  await screen.findByRole('alert');
  expect(screen.getByRole('alert').textContent).toContain(fieldError ? 'Schedule must equal total' : 'Please correct order');
  back(); expect((screen.getByLabelText('Количество') as HTMLInputElement).value).toBe('7');
  expect((screen.getByLabelText('Цена') as HTMLInputElement).value).toBe('10.01');
  fail = false; next(); fireEvent.click(screen.getByRole('button', { name: 'Отправить заказ' }));
  await waitFor(() => expect(created).toHaveBeenCalledWith('created'));
  expect(posts).toHaveBeenCalledTimes(2);
});

it('uses the captured promotion, omits zero rows, totals chosen quantities and guards repeated sends', async () => {
  let finish!: (response: Response) => void;
  const { posts, created } = setup({ offer, post: () => new Promise(resolve => { finish = resolve; }) });
  await enterLines(true); next();
  expect(screen.getByRole('alert').textContent).toContain('положительным количеством');
  fireEvent.click(screen.getByRole('button', { name: 'Уменьшить количество 1' }));
  expect((screen.getByLabelText('Количество 1') as HTMLInputElement).value).toBe('0');
  change('Количество 1', '10000'); fireEvent.click(screen.getByRole('button', { name: 'Увеличить количество 1' }));
  expect((screen.getByLabelText('Количество 1') as HTMLInputElement).value).toBe('10000');
  change('Количество 1', '3'); next();
  expect(screen.getByText('30.03 EUR')).toBeTruthy();
  expect(screen.getByText(/график оплаты акции не переносится/)).toBeTruthy();
  expect(screen.queryByText('Make m2 Sport')).toBeNull();
  expect(screen.getByText('Гарантия: Warranty')).toBeTruthy();
  const send = screen.getByRole('button', { name: 'Отправить заказ' });
  fireEvent.click(send); fireEvent.click(send);
  expect(posts).toHaveBeenCalledTimes(1);
  expect(posts).toHaveBeenCalledWith({ offerVersionId: 'v1', warehouseId: 'w', lines: [{ offerLineId: 'l1', quantity: '3', modelSpecificationVersion: '1', exteriorColor: 'White', interiorColor: 'Black' }] });
  finish(response({ data: { id: 'created' } }, 201));
  await waitFor(() => expect(created).toHaveBeenCalledWith('created'));
});

it('keeps the published payment schedule when every promotion quantity is unchanged', async () => {
  setup({ offer }); await enterLines(true);
  change('Количество 1', '1'); change('Количество 2', '1'); next();
  expect(screen.getByText(/График оплаты: 30.01 EUR до 2026-12-01/)).toBeTruthy();
});

const pinnedOffer = (knownColors = false): Offer => ({ ...offer, publishedVersion: { ...offer.publishedVersion!, terms: {
  ...offer.publishedVersion!.terms,
  lines: [{ ...offer.publishedVersion!.terms.lines[0]!, quantity: '2', modelSpecificationVersion: '1',
    ...(knownColors ? { exteriorColor: 'White', interiorColor: 'Tan' } : {}) }],
  paymentSchedule: [{ amount: { amountMinor: '2002', currency: 'EUR' }, dueDate: '2026-12-01' }],
} } });
const choosePair = (row: number, exterior: string, interior: string) => {
  pick(`Цвет кузова ${row}`, exterior); pick(`Цвет салона ${row}`, interior);
};

it('colors direct repeated models keep independent historical pairs, review labels and exact totals', async () => {
  const detail = historicalModel();
  const { posts } = setup({ read: url => url.includes('vehicle-models?')
    ? response({ items: [{ ...detail, specification: detail.versions[0] }, model('m2')] })
    : url.endsWith('/vehicle-models/m') ? response({ data: detail }) : undefined });
  await enterLines();
  expect((screen.getByLabelText('Цвет кузова 1') as HTMLInputElement).value).toBe('');
  next(); expect(screen.getByRole('alert').textContent).toContain('Выберите цвет кузова и салона');
  choosePair(1, 'White', 'Black');
  fireEvent.click(screen.getByRole('button', { name: 'Добавить модель' }));
  pick('Модель 2', 'Make m Sport'); change('Цена 2', '10.01'); choosePair(2, 'Blue', 'Tan');
  expect((screen.getByLabelText('Цвет кузова 1') as HTMLInputElement).value).toBe('White');
  next();
  expect(screen.getByText('Кузов: White · Салон: Black · Версия: 1')).toBeTruthy();
  expect(screen.getByText('Кузов: Blue · Салон: Tan · Версия: 1')).toBeTruthy();
  expect(screen.getByText('20.02 USD')).toBeTruthy();
  fireEvent.click(screen.getByRole('button', { name: 'Отправить заказ' }));
  await waitFor(() => expect(posts).toHaveBeenCalledTimes(1));
  const payload = posts.mock.calls[0]![0] as { terms: { lines: unknown[] } };
  expect(payload.terms.lines).toEqual([
    { modelId: 'm', quantity: '1', modelSpecificationVersion: '1', exteriorColor: 'White', interiorColor: 'Black', unitPrice: { amountMinor: '1001', currency: 'USD' } },
    { modelId: 'm', quantity: '1', modelSpecificationVersion: '1', exteriorColor: 'Blue', interiorColor: 'Tan', unitPrice: { amountMinor: '1001', currency: 'USD' } },
  ]);
});

it('colors changing model or version clears stale choices and requires a new pair', async () => {
  const first = historicalModel();
  const second = historicalModel('m2');
  const { client, posts } = setup({ read: url => url.includes('vehicle-models?')
    ? response({ items: [{ ...first, specification: first.versions[0] }, second] })
    : url.endsWith('/vehicle-models/m') ? response({ data: first })
      : url.endsWith('/vehicle-models/m2') ? response({ data: second }) : undefined });
  await enterLines(); choosePair(1, 'Blue', 'Tan');
  pick('Модель 1', 'Make m2 Sport');
  await waitFor(() => expect((screen.getByLabelText('Цвет кузова 1') as HTMLInputElement).disabled).toBe(false));
  expect((screen.getByLabelText('Цвет кузова 1') as HTMLInputElement).value).toBe('');
  expect((screen.getByLabelText('Цвет салона 1') as HTMLInputElement).value).toBe('');
  expect(openOptions('Цвет кузова 1').queryByRole('option', { name: 'Blue' })).toBeNull();
  next(); expect(screen.getByRole('alert').textContent).toContain('Выберите цвет');
  pick('Модель 1', 'Make m Sport');
  await waitFor(() => expect((screen.getByLabelText('Цвет кузова 1') as HTMLInputElement).disabled).toBe(false));
  choosePair(1, 'White', 'Black');
  await act(async () => { client.setQueryData(['models', 'order-selector'], [first, second]); });
  await waitFor(() => expect((screen.getByLabelText('Модель 1') as HTMLInputElement).value).toBe('Make m Sport'));
  // A deliberate new model selection captures its new version; the old pair cannot survive it.
  pick('Модель 1', 'Make m2 Sport'); pick('Модель 1', 'Make m Sport');
  await waitFor(() => expect(screen.getByText('Версия модели: 2')).toBeTruthy());
  expect((screen.getByLabelText('Цвет кузова 1') as HTMLInputElement).value).toBe('');
  expect((screen.getByLabelText('Цвет салона 1') as HTMLInputElement).value).toBe('');
  expect(posts).not.toHaveBeenCalled();
});

it('colors repeated offer rows retain captured version and price with aggregate schedule and submit guards', async () => {
  let finish!: (response: Response) => void;
  const original = pinnedOffer();
  const { posts, created, replaceOffer } = setup({ offer: original,
    read: url => url.endsWith('/vehicle-models/m') ? response({ data: historicalModel() }) : undefined,
    post: () => new Promise(resolve => { finish = resolve; }) });
  await enterLines(true); change('Количество 1', '1'); choosePair(1, 'White', 'Black');
  fireEvent.click(screen.getByRole('button', { name: 'Добавить сочетание 1' }));
  choosePair(2, 'Blue', 'Tan');
  const replacement = structuredClone(original);
  replacement.publishedVersion!.id = 'new-offer-version';
  replacement.publishedVersion!.terms.lines[0]!.unitPrice.amountMinor = '999999';
  replaceOffer(replacement);
  next();
  expect(screen.getByText('Кузов: White · Салон: Black · Версия: 1')).toBeTruthy();
  expect(screen.getByText('Кузов: Blue · Салон: Tan · Версия: 1')).toBeTruthy();
  expect(screen.getByText('20.02 EUR')).toBeTruthy();
  expect(screen.getByText('График оплаты: 20.02 EUR до 2026-12-01')).toBeTruthy();
  expect(screen.queryByText(/график оплаты акции не переносится/)).toBeNull();
  const send = screen.getByRole('button', { name: 'Отправить заказ' });
  fireEvent.click(send); fireEvent.click(send);
  expect(posts).toHaveBeenCalledTimes(1);
  expect(posts).toHaveBeenCalledWith({ offerVersionId: 'v1', warehouseId: 'w', lines: [
    { offerLineId: 'l1', quantity: '1', modelSpecificationVersion: '1', exteriorColor: 'White', interiorColor: 'Black' },
    { offerLineId: 'l1', quantity: '1', modelSpecificationVersion: '1', exteriorColor: 'Blue', interiorColor: 'Tan' },
  ] });
  expect((screen.getByRole('button', { name: 'Назад' }) as HTMLButtonElement).disabled).toBe(true);
  finish(response({ data: { id: 'created' } }, 201));
  await waitFor(() => expect(created).toHaveBeenCalledWith('created'));
});

it('colors known offered facts restrict both palettes and survive a repeated row', async () => {
  const { posts } = setup({ offer: pinnedOffer(true), read: url => url.endsWith('/vehicle-models/m') ? response({ data: historicalModel() }) : undefined });
  await enterLines(true); change('Количество 1', '1');
  expect((screen.getByLabelText('Цвет кузова 1') as HTMLInputElement).value).toBe('White');
  expect((screen.getByLabelText('Цвет салона 1') as HTMLInputElement).value).toBe('Tan');
  expect(openOptions('Цвет кузова 1').queryByRole('option', { name: 'Blue' })).toBeNull();
  fireEvent.blur(screen.getByRole('combobox', { name: 'Цвет кузова 1' }));
  expect(openOptions('Цвет салона 1').queryByRole('option', { name: 'Black' })).toBeNull();
  fireEvent.click(screen.getByRole('button', { name: 'Добавить сочетание 1' }));
  expect((screen.getByLabelText('Цвет салона 2') as HTMLInputElement).value).toBe('Tan');
  next(); fireEvent.click(screen.getByRole('button', { name: 'Отправить заказ' }));
  await waitFor(() => expect(posts).toHaveBeenCalledTimes(1));
  expect((posts.mock.calls[0]![0] as { lines: { exteriorColor: string; interiorColor: string }[] }).lines.map(line => [line.exteriorColor, line.interiorColor])).toEqual([['White', 'Tan'], ['White', 'Tan']]);
});

it('colors loading missing history and failed retry block progress without current-palette fallback', async () => {
  let finish!: (response: Response) => void;
  let state = 'loading';
  const detail = historicalModel();
  const { posts, client } = setup({ offer: pinnedOffer(), read: url => {
    if (!url.endsWith('/vehicle-models/m')) return;
    if (state === 'loading') return new Promise(resolve => { finish = resolve; });
    if (state === 'error') return response({ error: { message: 'Unavailable' } }, 503);
    return response({ data: detail });
  } });
  await enterLines(true, false); change('Количество 1', '2'); next();
  expect(screen.getByRole('alert').textContent).toContain('Загрузите точные версии');
  finish(response({ data: { ...detail, versions: [detail.specification] } }));
  await screen.findByText('Точная версия модели недоступна.'); next();
  expect(screen.getByText(/Версия модели в строке 1 недоступна/)).toBeTruthy();
  expect((screen.getByLabelText('Цвет кузова 1') as HTMLInputElement).disabled).toBe(true);
  state = 'error'; fireEvent.click(screen.getByRole('button', { name: 'Повторить загрузку версий' }));
  await screen.findByText('Не удалось загрузить версии моделей.'); next();
  expect(screen.getByText('Загрузите точные версии моделей для выбора цветов')).toBeTruthy();
  state = 'ready'; fireEvent.click(screen.getByRole('button', { name: 'Повторить загрузку версий' }));
  await waitFor(() => expect((screen.getByLabelText('Цвет кузова 1') as HTMLInputElement).disabled).toBe(false));
  expect((screen.getByLabelText('Количество 1') as HTMLInputElement).value).toBe('2');
  choosePair(1, 'Blue', 'Tan'); next();
  // Revalidation also occurs at the final POST boundary if historical data becomes unavailable.
  client.setQueryData(['models', 'order-details', 'm'], [{ ...detail, versions: [detail.specification] }]);
  await waitFor(() => expect(screen.getByText('Кузов: Не указан · Салон: Не указан · Версия: 1')).toBeTruthy());
  fireEvent.click(screen.getByRole('button', { name: 'Отправить заказ' }));
  expect(screen.getByText(/Версия модели в строке 1 недоступна/)).toBeTruthy();
  expect(posts).not.toHaveBeenCalled();
});
