import { get, list } from '@justixauto/kit';
import type { Listing, Money, Offer, Terms, Vehicle } from './data';

export type SaleSourceSeed =
  | { kind: 'supplier-offer' | 'own-offer'; id: string; versionId: string }
  | { kind: 'own-listing'; id: string };
export type SaleSourceKind = SaleSourceSeed['kind'];
export type SaleSourceDetail = Offer | Listing;
export const sourceLabels: Record<SaleSourceKind, string> = {
  'supplier-offer': 'Предложение поставщика', 'own-offer': 'Своё предложение партнёрам', 'own-listing': 'Своё предложение клиентам',
};
export interface AppliedSaleSource {
  seed: SaleSourceSeed;
  lineKey: string;
  label: string;
  price: Money;
  line?: Terms['lines'][number];
  vehicleId?: string;
  fingerprint: string;
}

/** Decimal text only: even 38-digit amounts round-trip through toMinor. */
export function salePriceText(amountMinor: string): string {
  if (!/^(0|[1-9]\d*)$/.test(amountMinor)) throw new Error('Некорректная цена предложения');
  const digits = amountMinor.padStart(3, '0');
  return `${digits.slice(0, -2)}.${digits.slice(-2)}`;
}

/** A result is published only after every offset page succeeds. */
export async function saleSelectorItems<T extends { id: string }>(path: string): Promise<T[]> {
  const [base, search = ''] = path.split('?');
  const params = new URLSearchParams(search);
  const items = new Map<string, T>();
  for (let offset = 0; ; offset += 100) {
    params.set('limit', '100'); params.set('offset', String(offset));
    const page = await list<T>(`${base}?${params}`);
    const before = items.size;
    for (const item of page) items.set(item.id, item);
    if (page.length && before === items.size) throw new Error('Список не продвигается. Повторите загрузку.');
    if (page.length < 100) return [...items.values()];
  }
}
export const readSaleVehicles = () => saleSelectorItems<Vehicle>('/inventory/vehicle-units?placement=warehouse&eligible=true');
export const readSaleSources = (kind: SaleSourceKind): Promise<SaleSourceDetail[]> => kind === 'own-listing'
  ? saleSelectorItems<Listing>('/retail/listings')
  : saleSelectorItems<Offer>(`/commerce/offers?scope=${kind === 'own-offer' ? 'own' : 'available'}`);
export const sourceDetailKey = (company: string, seed?: SaleSourceSeed) =>
  ['sale-source', company, seed?.kind ?? '', seed?.id ?? '', seed && 'versionId' in seed ? seed.versionId : ''];
export async function readSaleSource(seed: SaleSourceSeed): Promise<SaleSourceDetail> {
  return seed.kind === 'own-listing'
    ? (await get<Listing>(`/retail/listings/${encodeURIComponent(seed.id)}`)).data
    : (await get<Offer>(`/commerce/offers/${encodeURIComponent(seed.id)}`)).data;
}
export function shownSourceSeed(kind: SaleSourceKind, detail: SaleSourceDetail): SaleSourceSeed {
  if (kind === 'own-listing') return { kind, id: detail.id };
  const offer = detail as Offer;
  const version = kind === 'own-offer' ? offer.versions?.at(-1) : offer.publishedVersion;
  return { kind, id: offer.id, versionId: version?.id ?? '' };
}
export const saleLineKey = (line: Terms['lines'][number], index: number) => line.lineId ? `id:${line.lineId}` : `index:${index}`;

export function sourceVersion(detail: SaleSourceDetail, seed: SaleSourceSeed, company: string, selecting = false) {
  if (seed.kind === 'own-listing') throw new Error('Выберите предложение партнёрам');
  const offer = detail as Offer;
  if (offer.id !== seed.id || offer.status === 'withdrawn') throw new Error('Предложение недоступно');
  if (seed.kind === 'supplier-offer') {
    if (offer.supplier.id === company || offer.status !== 'published' || offer.publishedVersion?.id !== seed.versionId)
      throw new Error('Опубликованная версия изменилась или недоступна. Выберите актуальную версию.');
    return offer.publishedVersion;
  }
  const version = offer.versions?.find(v => v.id === seed.versionId);
  if (offer.supplier.id !== company || !version) throw new Error('Выбранная версия недоступна');
  if (selecting && offer.versions?.at(-1)?.id !== seed.versionId) throw new Error('Показанная версия изменилась. Выберите актуальную версию.');
  return version;
}
export function resolveSaleSource(detail: SaleSourceDetail, seed: SaleSourceSeed, lineKey: string, company: string, selecting = false): AppliedSaleSource {
  if (seed.kind === 'own-listing') {
    const listing = detail as Listing;
    if (listing.id !== seed.id || !['draft', 'published'].includes(listing.status)) throw new Error('Предложение клиентам недоступно');
    salePriceText(listing.askingPrice.amountMinor);
    return { seed, lineKey: '', price: listing.askingPrice, vehicleId: listing.vehicleId,
      label: `${sourceLabels[seed.kind]} · ${listing.status === 'draft' ? 'Черновик' : 'Опубликовано'} · ${listing.text || listing.id}`,
      fingerprint: JSON.stringify([listing.revision, listing.vehicleId, listing.askingPrice]),
    };
  }
  const version = sourceVersion(detail, seed, company, selecting);
  const line = version.terms.lines.find((l, index) => saleLineKey(l, index) === lineKey);
  if (!line) throw new Error('Выберите одну строку предложения');
  salePriceText(line.unitPrice.amountMinor);
  return { seed, lineKey, line, price: line.unitPrice,
    label: `${sourceLabels[seed.kind]} · Версия №${version.number}${version.publishedAt ? ' · Опубликовано' : ' · Черновик'} · ${line.modelId} · ${lineKey}`,
    fingerprint: JSON.stringify([seed, lineKey, line]),
  };
}
export function checkAppliedSource(detail: SaleSourceDetail, applied: AppliedSaleSource, company: string): string {
  try {
    const next = resolveSaleSource(detail, applied.seed, applied.lineKey, company);
    return next.fingerprint === applied.fingerprint ? '' : 'Предложение изменилось. Примените его заново или отключите.';
  } catch (error) { return error instanceof Error ? error.message : String(error); }
}
export const eligibleSaleVehicle = (vehicle: Vehicle) => !!vehicle.vin?.trim() && !!vehicle.placement?.warehouseId && vehicle.reserved === false;
const normalized = (value?: string) => value?.trim().toLowerCase() ?? '';
export function matchesSaleSource(vehicle: Vehicle, source?: AppliedSaleSource | null): boolean {
  if (!eligibleSaleVehicle(vehicle)) return false;
  if (!source) return true;
  if (source.vehicleId) return vehicle.id === source.vehicleId;
  const line = source.line;
  return !!line && vehicle.modelId === line.modelId
    && (!line.modelSpecificationVersion?.trim() || vehicle.modelSpecificationVersion === line.modelSpecificationVersion)
    && (!normalized(line.exteriorColor) || normalized(vehicle.exteriorColor) === normalized(line.exteriorColor))
    && (!normalized(line.interiorColor) || normalized(vehicle.interiorColor) === normalized(line.interiorColor));
}
