import { useEffect, useRef, useState } from 'react';
import { ApiError, Autocomplete, Button, Notice, Stat, errorText, get, post, useData, useRefresh } from '@justixauto/kit';
import { shortId } from './order-workspace';
import { VinEditor, validateVins } from './vin-editor';
import { useModelDetail, useWarehouses } from './data';
import { colorPalette, exactSpecification, selectedColor, vehicleColorsLabel } from './vehicle-colors';

export interface ReceiptIdentificationBatch {
  id: string; modelId: string; warehouseId: string; confirmedQuantity: string; identifiedCount: string; unidentifiedCount: string;
  warehouseUnidentifiedCount?: string; revision: string; modelSpecificationVersion?: string; exteriorColor?: string; interiorColor?: string;
}
interface ReceiptLocations { batch: ReceiptIdentificationBatch; locations: { warehouseId: string; unidentifiedCount: string }[] }

/** A receipt is global; the user explicitly selects which current location supplies the VINs. */
export function ReceiptIdentification({ batch, modelName, refresh, onBack, onDone }: { batch: ReceiptIdentificationBatch; modelName: string; refresh: unknown[][]; onBack: () => void; onDone: (message: string) => void }) {
  const locations = useData(['receipt-batch', batch.id], () => get<ReceiptLocations>(`/inventory/receipt-batches/${batch.id}`));
  const warehouses = useWarehouses();
  const [warehouseId, setWarehouseId] = useState(batch.warehouseId);
  const available = locations.data?.data.locations ?? [];
  useEffect(() => {
    if (available.length && warehouseId && !available.some(location => location.warehouseId === warehouseId)) setWarehouseId('');
  }, [available, warehouseId]);
  const stock = useData(['stock', warehouseId], () => get<{ unidentifiedBatches: ReceiptIdentificationBatch[] }>(`/inventory/warehouses/${warehouseId}/inventory`), !!warehouseId);
  const global = locations.data?.data.batch ?? batch;
  const current = stock.data?.data.unidentifiedBatches.find(row => row.id === batch.id);
  const localAvailable = available.find(location => location.warehouseId === warehouseId)?.unidentifiedCount ?? current?.warehouseUnidentifiedCount ?? '0';
  const needsColors = !!current && (!current.exteriorColor || !current.interiorColor);
  const detail = useModelDetail(current?.modelId ?? global.modelId, needsColors);
  const spec = exactSpecification(detail.data, current?.modelSpecificationVersion ?? global.modelSpecificationVersion);
  const [colors, setColors] = useState({ exteriorColor: '', interiorColor: '', exteriorEdited: false, interiorEdited: false });
  const exteriorOptions = colorPalette(spec, 'exterior'); const interiorOptions = colorPalette(spec, 'interior');
  const exterior = current?.exteriorColor || (colors.exteriorEdited ? exteriorOptions.find(value => value === colors.exteriorColor) ?? '' : selectedColor(exteriorOptions, colors.exteriorColor));
  const interior = current?.interiorColor || (colors.interiorEdited ? interiorOptions.find(value => value === colors.interiorColor) ?? '' : selectedColor(interiorOptions, colors.interiorColor));
  const colorsReady = !needsColors || (!!spec && !detail.isFetching && !detail.isError && !!exterior && !!interior);
  const [rows, setRows] = useState<string[]>(['']); const [error, setError] = useState(''); const [busy, setBusy] = useState(false); const submitted = useRef(false); const reload = useRefresh();
  const cap = Math.min(1000, Number(localAvailable)); const complete = !!locations.data && Number(global.unidentifiedCount) === 0;
  const locationName = (id: string) => warehouses.data?.find(warehouse => warehouse.id === id)?.name ?? id;

  async function submit() {
    if (submitted.current || busy || !current || stock.isFetching || !warehouseId || current.modelId !== global.modelId) return;
    if (!colorsReady) { setError('Выберите цвета из точной версии спецификации партии'); return; }
    const valid = validateVins(rows, cap, true); if (!valid.valid) { setError(valid.errors.join(' · ')); return; }
    submitted.current = true; setBusy(true); setError('');
    try { await post(`/inventory/receipt-batches/${batch.id}/identifications`, { atomic: true, warehouseId, items: valid.vins.map(vin => ({ vin, modelId: current.modelId })), ...(!current.exteriorColor ? { exteriorColor: exterior } : {}), ...(!current.interiorColor ? { interiorColor: interior } : {}) }); }
    catch (caught) { submitted.current = false; setBusy(false); setError(caught instanceof ApiError ? [caught.message, ...Object.values(caught.fields)].join(' · ') : errorText(caught)); void Promise.all([stock.refetch(), locations.refetch()]); return; }
    let message = `VIN добавлены: ${valid.vins.length}. В выбранном складе осталось без VIN: ${Number(localAvailable) - valid.vins.length}.`;
    const keys = [...refresh, ['order'], ['orders'], ['stock'], ['vehicles'], ['warehouses'], ['receipt-batch', batch.id]];
    try { await reload(...[...new Map(keys.map(key => [JSON.stringify(key), key])).values()]); } catch { message += ' Не удалось обновить данные. Обновите заказ перед следующим действием.'; }
    onDone(message);
  }

  return <section className="receipt-identification kit-stack">
    <header className="receipt-identification-head"><h3>Ввести VIN — {modelName}</h3><p className="cell-sub">{vehicleColorsLabel(current ?? global)} · партия <span title={batch.id}>{shortId(batch.id)}</span></p></header>
    <div className="kit-grid receipt-identification-stats"><Stat label="Принято во всей партии" value={global.confirmedQuantity} /><Stat label="С VIN во всей партии" value={global.identifiedCount} /><Stat label="Без VIN во всей партии" value={global.unidentifiedCount} note={`в этом складе — ${localAvailable}`} /></div>
    {locations.isFetching && <p role="status">Загружаем размещения партии…</p>}
    {locations.error && <><Notice kind="danger">Не удалось загрузить размещения партии. {errorText(locations.error)}</Notice><Button onClick={() => void locations.refetch()}>Повторить загрузку партии</Button></>}
    {!locations.isFetching && !locations.error && !complete && <label className="receipt-identification-location">Склад для ввода VIN<select aria-label="Склад для ввода VIN" value={warehouseId} disabled={busy || !available.length} onChange={event => setWarehouseId(event.target.value)}><option value="">Выберите склад с остатком</option>{available.map(location => <option key={location.warehouseId} value={location.warehouseId}>{locationName(location.warehouseId)} · доступно без VIN {location.unidentifiedCount}</option>)}</select></label>}
    {stock.isFetching && <p role="status">Обновляем остаток выбранного склада…</p>}
    {stock.error && <><Notice kind="danger">Не удалось загрузить остаток склада. {errorText(stock.error)}</Notice><Button onClick={() => void stock.refetch()}>Повторить загрузку остатка</Button></>}
    {error && <Notice kind="danger">{error}</Notice>}
    {!locations.isFetching && !locations.error && (complete ? <Notice kind="success">Во всей партии больше нет автомобилей, ожидающих VIN.</Notice> : !available.length ? <Notice kind="warning">Партия ещё ожидает VIN, но доступного остатка нет. Обновите данные или обратитесь к администратору.</Notice> : !warehouseId ? <Notice>Выберите склад, где сейчас находится остаток партии.</Notice> : !stock.isFetching && !stock.error && (current?.modelId !== global.modelId ? <Notice kind="danger">Модель партии изменилась. Вернитесь в заказ и обновите данные.</Notice> : !current || Number(localAvailable) === 0 ? <Notice kind="warning">В выбранном складе больше нет автомобилей без VIN. Выберите другой склад.</Notice> : <>
      {needsColors && <fieldset disabled={busy || detail.isFetching}><legend>Уточнить неизвестные цвета партии</legend>{detail.isFetching && <p role="status">Загрузка спецификации…</p>}{(detail.isError || (!detail.isFetching && !spec)) && <Notice kind="danger">Точная версия спецификации партии недоступна. <Button onClick={() => void detail.refetch()}>Повторить загрузку спецификации</Button></Notice>}{!current.exteriorColor && <label>Цвет кузова<Autocomplete aria-label="Цвет кузова" required value={exterior} onChange={value => setColors(previous => ({ ...previous, exteriorColor: value, exteriorEdited: true }))} options={[{ value: '', label: 'Выберите цвет' }, ...exteriorOptions.map(value => ({ value, label: value }))]} /></label>}{!current.interiorColor && <label>Цвет салона<Autocomplete aria-label="Цвет салона" required value={interior} onChange={value => setColors(previous => ({ ...previous, interiorColor: value, interiorEdited: true }))} options={[{ value: '', label: 'Выберите цвет' }, ...interiorOptions.map(value => ({ value, label: value }))]} /></label>}</fieldset>}
      <VinEditor rows={rows} onChange={setRows} cap={cap} required disabled={busy} /><p className="cell-sub">VIN вводятся из остатка выбранного склада; общая партия сохраняет исходное поступление.</p>
    </>))}
    <div className="receipt-identification-footer"><Button icon="back" disabled={busy} onClick={onBack}>Назад</Button>{!locations.isFetching && !locations.error && !stock.isFetching && !stock.error && !complete && !!current && Number(localAvailable) > 0 && current.modelId === global.modelId && <Button variant="primary" busy={busy} disabled={!colorsReady} onClick={() => void submit()}>Сохранить VIN</Button>}</div>
  </section>;
}
