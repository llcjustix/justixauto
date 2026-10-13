import { useRef, useState } from 'react';
import { ApiError, Autocomplete, Button, Notice, Stat, errorText, get, post, useData, useRefresh } from '@justixauto/kit';
import { shortId } from './order-workspace';
import { VinEditor, validateVins } from './vin-editor';
import { useModelDetail } from './data';
import { colorPalette, exactSpecification, selectedColor, vehicleColorsLabel } from './vehicle-colors';

export interface ReceiptIdentificationBatch {
  id: string;
  modelId: string;
  warehouseId: string;
  confirmedQuantity: string;
  identifiedCount: string;
  unidentifiedCount: string;
  revision: string;
  modelSpecificationVersion?: string;
  exteriorColor?: string;
  interiorColor?: string;
}

/** Fresh inventory is authoritative; a batch absent from pending stock cannot be edited. */
export function ReceiptIdentification({ batch, modelName, refresh, onBack, onDone }: {
  batch: ReceiptIdentificationBatch;
  modelName: string;
  refresh: unknown[][];
  onBack: () => void;
  onDone: (message: string) => void;
}) {
  const stock = useData(['stock', batch.warehouseId], () => get<{ unidentifiedBatches: ReceiptIdentificationBatch[] }>(
    `/inventory/warehouses/${batch.warehouseId}/inventory`));
  const current = stock.data?.data.unidentifiedBatches.find((b) => b.id === batch.id);
  const needsColors = !!current && (!current.exteriorColor || !current.interiorColor);
  const detail = useModelDetail(current?.modelId ?? '', needsColors);
  const spec = exactSpecification(detail.data, current?.modelSpecificationVersion);
  const [colors, setColors] = useState({ exteriorColor: '', interiorColor: '', exteriorEdited: false, interiorEdited: false });
  const exteriorOptions = colorPalette(spec, 'exterior');
  const interiorOptions = colorPalette(spec, 'interior');
  const exterior = current?.exteriorColor || (colors.exteriorEdited ? exteriorOptions.find(value => value === colors.exteriorColor) ?? '' : selectedColor(exteriorOptions, colors.exteriorColor));
  const interior = current?.interiorColor || (colors.interiorEdited ? interiorOptions.find(value => value === colors.interiorColor) ?? '' : selectedColor(interiorOptions, colors.interiorColor));
  const colorsReady = !needsColors || (!!spec && !detail.isFetching && !detail.isError && !!exterior && !!interior);
  const [rows, setRows] = useState<string[]>(['']);
  const [error, setError] = useState('');
  const [busy, setBusy] = useState(false);
  const submitted = useRef(false);
  const reload = useRefresh();
  const cap = Math.min(1000, Number(current?.unidentifiedCount ?? 0));
  const complete = Number(batch.unidentifiedCount) === 0 || (!!stock.data && !current);

  async function submit() {
    if (submitted.current || busy || !current || stock.isFetching || current.modelId !== batch.modelId) return;
    if (!colorsReady) { setError('Выберите цвета из точной версии спецификации партии'); return; }
    const valid = validateVins(rows, cap, true);
    if (!valid.valid) { setError(valid.errors.join(' · ')); return; }
    submitted.current = true;
    setBusy(true);
    setError('');
    try {
      await post(`/inventory/receipt-batches/${batch.id}/identifications`, {
        atomic: true, items: valid.vins.map((vin) => ({ vin, modelId: current.modelId })),
        ...(!current.exteriorColor ? { exteriorColor: exterior } : {}),
        ...(!current.interiorColor ? { interiorColor: interior } : {}),
      });
    } catch (e) {
      submitted.current = false;
      setBusy(false);
      setError(e instanceof ApiError ? [e.message, ...Object.values(e.fields)].join(' · ') : errorText(e));
      // A conflict refreshes facts, never the user's draft or the command itself.
      void stock.refetch();
      return;
    }
    let message = `VIN добавлены: ${valid.vins.length}. Осталось без VIN: ${Number(current.unidentifiedCount) - valid.vins.length}. Количество партии и занятость склада не изменились.`;
    const keys = [...refresh, ['order'], ['orders'], ['stock'], ['vehicles'], ['warehouses']];
    const unique = [...new Map(keys.map((key) => [JSON.stringify(key), key])).values()];
    try { await reload(...unique); }
    catch { message += ' Не удалось обновить данные. Обновите заказ перед следующим действием.'; }
    onDone(message);
  }

  return <section className="receipt-identification kit-stack">
    <header className="receipt-identification-head">
      <h3>Ввести VIN — {modelName}</h3>
      <p className="cell-sub">{vehicleColorsLabel(current ?? batch)} · партия <span title={batch.id}>{shortId(batch.id)}</span></p>
    </header>
    <div className="kit-grid receipt-identification-stats">
      <Stat label="Принято" value={current?.confirmedQuantity ?? batch.confirmedQuantity} />
      <Stat label="С VIN" value={current?.identifiedCount ?? batch.identifiedCount} />
      <Stat label="Без VIN" value={current?.unidentifiedCount ?? batch.unidentifiedCount} note={`за один раз — до ${cap}`} />
    </div>
    {stock.isFetching && <p role="status">Обновляем остаток партии…</p>}
    {stock.error && <><Notice kind="danger">Не удалось загрузить партию. {errorText(stock.error)}</Notice>
      <Button onClick={() => void stock.refetch()}>Повторить загрузку партии</Button></>}
    {error && <Notice kind="danger">{error}</Notice>}
    {!stock.isFetching && !stock.error && (complete
      ? <Notice kind="success">В партии больше нет автомобилей, ожидающих VIN.</Notice>
      : current?.modelId !== batch.modelId ? <Notice kind="danger">Модель партии изменилась. Вернитесь в заказ и обновите данные.</Notice>
      : current && <>
        {needsColors && <fieldset disabled={busy || detail.isFetching}>
          <legend>Уточнить неизвестные цвета партии</legend>
          {detail.isFetching && <p role="status">Загрузка спецификации…</p>}
          {(detail.isError || (!detail.isFetching && !spec)) && <Notice kind="danger">Точная версия спецификации партии недоступна. <Button onClick={() => void detail.refetch()}>Повторить загрузку спецификации</Button></Notice>}
          {!current.exteriorColor && <label>Цвет кузова<Autocomplete aria-label="Цвет кузова" required value={exterior} onChange={value => setColors(previous => ({ ...previous, exteriorColor: value, exteriorEdited: true }))} options={[{ value: '', label: 'Выберите цвет' }, ...exteriorOptions.map(value => ({ value, label: value }))]} /></label>}
          {!current.interiorColor && <label>Цвет салона<Autocomplete aria-label="Цвет салона" required value={interior} onChange={value => setColors(previous => ({ ...previous, interiorColor: value, interiorEdited: true }))} options={[{ value: '', label: 'Выберите цвет' }, ...interiorOptions.map(value => ({ value, label: value }))]} /></label>}
        </fieldset>}
        <VinEditor rows={rows} onChange={setRows} cap={cap} required disabled={busy} />
        <p className="cell-sub">Можно заполнить часть партии и вернуться к остальным VIN позже.</p>
      </>)}
    <div className="receipt-identification-footer">
      <Button icon="back" disabled={busy} onClick={onBack}>Назад</Button>
      {!stock.isFetching && !stock.error && !complete && current?.modelId === batch.modelId && current &&
        <Button variant="primary" busy={busy} disabled={!colorsReady} onClick={() => void submit()}>Сохранить VIN</Button>}
    </div>
  </section>;
}
