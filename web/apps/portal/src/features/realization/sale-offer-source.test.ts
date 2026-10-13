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
      paths.push(url); if (url.includes('/consignments/retail-candidates?')) return new Response(JSON.stringify({ items: [], nextCursor: null }));
      const offset = new URL(url, 'http://test').searchParams.get('offset');
      return new Response(JSON.stringify({ items: offset === '0' ? Array.from({ length: 100 }, (_, i) => ({ ...vehicle, id: String(i), modelId: 'other' })) : [vehicle] }));
    }));
    const rows = await readSaleVehicles();
    expect(rows).toHaveLength(101); expect(rows.filter(v => matchesSaleSource(v, resolveSaleSource(offer, seed, 'id:line', 'company')))).toEqual([vehicle]);
    expect(paths.filter(path => path.includes('/vehicle-units?'))).toHaveLength(2);
    expect(paths.find(path => path.includes('placement=warehouse&eligible=true&limit=100&offset=100'))).toBeTruthy();
    expect(paths.filter(path => path.includes('/consignments/retail-candidates?'))).toHaveLength(1);
  });
  it('hydrates every candidate page through data.vehicle, dedupes IDs, and keeps only exact eligible detail', async () => {
    const paths: string[] = [];
    vi.stubGlobal('fetch', vi.fn(async (url: string) => {
      paths.push(url);
      if (url.includes('/vehicle-units?')) return new Response(JSON.stringify({ items: [vehicle] }));
      if (url.includes('/retail-candidates?')) {
        const offset = new URL(url, 'http://test').searchParams.get('offset');
        const ids = offset === '0' ? Array.from({ length: 100 }, (_, i) => String(i)) : ['99', '100', 'car'];
        return new Response(JSON.stringify({ items: ids.map(vehicleId => ({ vehicleId })), nextCursor: offset === '0' ? '100' : null }));
      }
      const id = decodeURIComponent(url.split('/').at(-1)!);
      return new Response(JSON.stringify({ data: { vehicle: { ...vehicle, id, vin: id === '0' ? ' ' : id, reserved: id === '1' }, specification: { id: 'wrong' } } }));
    }));
    const rows = await readSaleVehicles();
    expect(rows).toHaveLength(100); // owned + 101 distinct candidates, minus two ineligible details
    expect(rows[0]).toEqual(vehicle);
    expect(rows.some(row => row.id === '0' || row.id === '1')).toBe(false);
    expect(paths.filter(path => path.includes('/vehicle-units/'))).toHaveLength(101);
    expect(paths.filter(path => path.includes('/retail-candidates?')).map(path => new URL(path, 'http://test').searchParams.get('offset'))).toEqual(['0', '100']);
  });
  it.each(['page', 'detail', 'identity', 'permission'])('fails the complete combined load on %s failure', async mode => {
    vi.stubGlobal('fetch', vi.fn(async (url: string) => {
      if (url.includes('/vehicle-units?')) return new Response(JSON.stringify({ items: [vehicle] }));
      if (url.includes('/retail-candidates?')) {
        if (mode === 'permission') return new Response(JSON.stringify({ error: { message: 'Forbidden' } }), { status: 403 });
        const offset = new URL(url, 'http://test').searchParams.get('offset');
        if (offset === '100' && mode === 'page') return new Response(JSON.stringify({ error: { message: 'Later page failed' } }), { status: 503 });
        return new Response(JSON.stringify({ items: (offset === '0' && mode === 'page' ? Array.from({ length: 100 }, (_, i) => String(i)) : ['consigned']).map(vehicleId => ({ vehicleId })), nextCursor: offset === '0' && mode === 'page' ? '100' : null }));
      }
      if (mode === 'detail') return new Response(JSON.stringify({ error: { message: 'Detail failed' } }), { status: 503 });
      return new Response(JSON.stringify({ data: { vehicle: { ...vehicle, id: mode === 'identity' ? 'wrong' : 'consigned' } } }));
    }));
    await expect(readSaleVehicles()).rejects.toThrow();
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
