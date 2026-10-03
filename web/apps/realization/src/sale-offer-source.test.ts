import { afterEach, describe, expect, it, vi } from 'vitest';
import { toMinor } from '@justixauto/kit';
import type { Listing, Offer, Vehicle } from './data';
import { checkAppliedSource, matchesSaleSource, readSaleVehicles, resolveSaleSource, salePriceText, saleSelectorItems, shownSourceSeed } from './sale-offer-source';

const price = { amountMinor: '90071992547409931234567890123456789012', currency: 'USD' };
const line = { lineId: 'line', modelId: 'model', quantity: '50', unitPrice: price, modelSpecificationVersion: 'v1', exteriorColor: ' WHITE ', interiorColor: 'Black' };
const version = { id: 'v1', number: 1, publishedAt: 'today', total: { ...price, amountMinor: '5' }, terms: { lines: [line] } };
const offer = { id: 'offer', supplier: { id: 'supplier', name: 'Supplier' }, status: 'published', publishedVersion: version } as Offer;
const seed = { kind: 'supplier-offer', id: 'offer', versionId: 'v1' } as const;
const vehicle = { id: 'car', vin: 'VIN', modelId: 'model', modelSpecificationVersion: 'v1', exteriorColor: 'White', interiorColor: ' BLACK ', reserved: false, placement: { warehouseId: 'w' } } as Vehicle;
afterEach(() => vi.unstubAllGlobals());

describe('sale offer source', () => {
  it('round-trips huge exact Money and selects the line unit price, never quantity or total', () => {
    expect(toMinor(salePriceText(price.amountMinor))).toBe(price.amountMinor);
    expect(salePriceText('1')).toBe('0.01'); expect(salePriceText('100')).toBe('1.00');
    expect(resolveSaleSource(offer, seed, 'id:line', 'company').price).toEqual(price);
  });
  it('requires explicit model/version/color facts, normalizes only color trim/case, and rejects ineligible VINs', () => {
    const source = resolveSaleSource(offer, seed, 'id:line', 'company');
    expect(matchesSaleSource(vehicle, source)).toBe(true);
    for (const patch of [{ modelId: 'other' }, { modelSpecificationVersion: '' }, { exteriorColor: '' }, { interiorColor: 'Grey' }, { reserved: true }, { placement: null }, { vin: '  ' }])
      expect(matchesSaleSource({ ...vehicle, ...patch }, source)).toBe(false);
    const unrestricted = { ...source, line: { ...line, modelSpecificationVersion: '', exteriorColor: '', interiorColor: '' } };
    expect(matchesSaleSource({ ...vehicle, modelSpecificationVersion: '', exteriorColor: undefined, interiorColor: undefined }, unrestricted)).toBe(true);
  });
  it.each(['draft', 'published'])('binds %s own listing to exact vehicle and detects revision/price changes', status => {
    const listing = { id: 'listing', status, vehicleId: 'car', askingPrice: price, revision: '1', text: 'Client offer' } as Listing;
    const source = resolveSaleSource(listing, { kind: 'own-listing', id: 'listing' }, '', 'company');
    expect(matchesSaleSource(vehicle, source)).toBe(true);
    expect(matchesSaleSource({ ...vehicle, id: 'another' }, source)).toBe(false);
    expect(checkAppliedSource({ ...listing, revision: '2' }, source, 'company')).toContain('изменилось');
    expect(checkAppliedSource({ ...listing, status: 'withdrawn' }, source, 'company')).toContain('недоступно');
  });
  it('requires current supplier publication but pins an applied own draft even when a newer version appears', () => {
    const source = resolveSaleSource(offer, seed, 'id:line', 'company');
    expect(checkAppliedSource({ ...offer, publishedVersion: { ...version, id: 'v2' } } as Offer, source, 'company')).toContain('версия');
    expect(() => resolveSaleSource({ ...offer, status: 'withdrawn' }, seed, 'id:line', 'company')).toThrow();
    const draft = { ...version, id: 'draft', number: 2, publishedAt: null };
    const own = { ...offer, supplier: { id: 'company', name: 'Us' }, versions: [version, draft] } as Offer;
    const ownSeed = shownSourceSeed('own-offer', own);
    expect(ownSeed).toEqual({ kind: 'own-offer', id: 'offer', versionId: 'draft' });
    const pinned = resolveSaleSource(own, ownSeed, 'id:line', 'company', true);
    expect(pinned.label).toContain('Черновик');
    const newer = { ...own, versions: [...own.versions!, { ...draft, id: 'third', number: 3 }] };
    expect(checkAppliedSource(newer, pinned, 'company')).toBe('');
    expect(() => resolveSaleSource(newer, ownSeed, 'id:line', 'company', true)).toThrow('Показанная версия');
    expect(checkAppliedSource({ ...own, versions: [version] }, pinned, 'company')).toContain('недоступна');
  });
  it('distinguishes repeated-model lines and scopes legacy index selection to a pinned version', () => {
    const detail = { ...offer, publishedVersion: { ...version, terms: { lines: [{ ...line, lineId: undefined }, { ...line, lineId: undefined, unitPrice: { ...price, amountMinor: '42' } }] } } } as Offer;
    expect(() => resolveSaleSource(detail, seed, '', 'company')).toThrow('одну строку');
    expect(resolveSaleSource(detail, seed, 'index:1', 'company').price.amountMinor).toBe('42');
    expect(() => resolveSaleSource(detail, { ...seed, versionId: 'other' }, 'index:1', 'company')).toThrow();
  });
  it('includes page 101 and preserves warehouse/eligible parameters', async () => {
    const paths: string[] = [];
    vi.stubGlobal('fetch', vi.fn(async (url: string) => {
      paths.push(url); const offset = new URL(url, 'http://test').searchParams.get('offset');
      return new Response(JSON.stringify({ items: offset === '0' ? Array.from({ length: 100 }, (_, i) => ({ ...vehicle, id: String(i), modelId: 'other' })) : [vehicle] }));
    }));
    const rows = await readSaleVehicles();
    expect(rows).toHaveLength(101); expect(rows.filter(v => matchesSaleSource(v, resolveSaleSource(offer, seed, 'id:line', 'company')))).toEqual([vehicle]);
    expect(paths).toHaveLength(2); expect(paths[1]).toContain('placement=warehouse&eligible=true&limit=100&offset=100');
  });
  it('reads the empty page after a full 100 and deduplicates overlapping pages', async () => {
    const pages = [Array.from({ length: 100 }, (_, i) => ({ id: String(i) })), []];
    const fetch = vi.fn(async () => new Response(JSON.stringify({ items: pages.shift() })));
    vi.stubGlobal('fetch', fetch); expect(await saleSelectorItems('/items?scope=own')).toHaveLength(100); expect(fetch).toHaveBeenCalledTimes(2);
    pages.push(Array.from({ length: 100 }, (_, i) => ({ id: String(i) })), [{ id: '99' }, { id: '100' }]);
    expect(await saleSelectorItems('/items')).toHaveLength(101);
  });
  it.each(['failure', 'repeated'])('rejects incomplete %s later pages', async mode => {
    let calls = 0;
    vi.stubGlobal('fetch', vi.fn(async () => {
      if (++calls > 1 && mode === 'failure') return new Response(JSON.stringify({ error: { message: 'page failed' } }), { status: 503 });
      return new Response(JSON.stringify({ items: Array.from({ length: 100 }, (_, i) => ({ id: String(i) })) }));
    }));
    await expect(saleSelectorItems('/items')).rejects.toThrow(); expect(calls).toBe(2);
  });
});
