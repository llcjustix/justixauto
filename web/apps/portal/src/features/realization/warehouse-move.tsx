import { useMemo, useRef, useState } from 'react';
import { ApiError, Button, Notice, date, errorText, get, post, useData, useRefresh, useSession } from '@justixauto/kit';
import { useWarehouses } from './data';
import type { Vehicle, Warehouse } from './data';
import { vehicleColorsLabel } from './vehicle-colors';
import './warehouse-move.css';

export interface WarehouseMoveBatch {
  id: string;
  modelId: string;
  modelSpecificationVersion: string;
  confirmedQuantity: string;
  identifiedCount: string;
  unidentifiedCount: string;
  warehouseUnidentifiedCount: string;
  receivedAt: string;
  revision: string;
  exteriorColor?: string;
  interiorColor?: string;
}

interface Stock { warehouse: Warehouse; vehicles: Vehicle[]; unidentifiedBatches: WarehouseMoveBatch[] }

const limit = 1000;
const positiveInteger = (value: string) => /^[1-9]\d*$/.test(value) && Number.isSafeInteger(Number(value));
const localDateTime = () => { const now = new Date(); const offset = now.getTimezoneOffset() * 60000; return new Date(now.getTime() - offset).toISOString().slice(0, 16); };
const byVin = (a: Vehicle, b: Vehicle) => a.vin.localeCompare(b.vin) || a.id.localeCompare(b.id);

export function WarehouseMove({ source, initialVehicles, initialBatches, modelName, onBack, onDone, onBusy, onVehicle }: {
  source: Warehouse;
  initialVehicles: Vehicle[];
  initialBatches: WarehouseMoveBatch[];
  modelName: (modelId: string) => string;
  onBack: () => void;
  onDone: (message: string) => void;
  onBusy: (busy: boolean) => void;
  onVehicle: (id: string) => void;
}) {
  const session = useSession();
  const warehouses = useWarehouses();
  const fresh = useData(['stock', source.id, 'move-facts'], () => get<Stock>(`/inventory/warehouses/${source.id}/inventory`));
  const reload = useRefresh();
  const [destinationId, setDestinationId] = useState('');
  const [occurredAt, setOccurredAt] = useState(localDateTime);
  const [selected, setSelected] = useState<string[]>([]);
  const [quantities, setQuantities] = useState<Record<string, string>>({});
  const [firstN, setFirstN] = useState('');
  const [error, setError] = useState('');
  const [needsReconfirm, setNeedsReconfirm] = useState(false);
  const [busy, setBusy] = useState(false);
  const sending = useRef(false);
  const [snapshot, setSnapshot] = useState(() => capture(source, initialVehicles, initialBatches, warehouses.data ?? []));
  const facts = fresh.data?.data;
  const vehicles = facts?.vehicles ?? initialVehicles;
  const batches = facts?.unidentifiedBatches ?? initialBatches;
  const destinations = (warehouses.data ?? []).filter(warehouse => warehouse.id !== source.id);
  const selectedVehicles = useMemo(() => selected.map(id => {
    const vehicle = vehicles.find(item => item.id === id) ?? snapshot.vehicles.get(id);
    return vehicle ? { vehicle, unavailable: !vehicles.some(current => current.id === vehicle.id) } : { vehicle: { id, vin: id, modelId: '', modelSpecificationVersion: '', reserved: false, ownerCompanyId: null, custodianCompanyId: null, revision: '', placement: null }, unavailable: true };
  }), [selected, vehicles, snapshot.vehicles]);
  // Keep an explicit draft intact across a stale refresh; it is only resnapshotted by reconfirm().
  const selectedBatches = Object.entries(quantities).filter(([, quantity]) => positiveInteger(quantity)).map(([id]) => batches.find(batch => batch.id === id) ?? snapshot.batches.get(id)).filter((batch): batch is WarehouseMoveBatch => !!batch);
  const total = selectedVehicles.length + selectedBatches.reduce((sum, batch) => sum + Number(quantities[batch.id]), 0);
  const canMove = session.can('inventory.vehicles.move');

  function select(id: string) {
    setSelected(previous => previous.includes(id) ? previous.filter(item => item !== id) : [...previous, id]);
    setError('');
  }
  function selectAll() { setSelected(vehicles.map(vehicle => vehicle.id)); setError(''); }
  function clearAll() { setSelected([]); setError(''); }
  function chooseFirstN() {
    if (!positiveInteger(firstN)) { setError('Введите положительное целое число автомобилей с VIN.'); return; }
    const count = Number(firstN);
    if (count > vehicles.length || count > limit) { setError(`Можно выбрать не больше ${Math.min(vehicles.length, limit)} загруженных VIN.`); return; }
    setSelected(vehicles.slice().sort(byVin).slice(0, count).map(vehicle => vehicle.id));
    setError('');
  }
  function setQuantity(batchId: string, value: string) {
    setQuantities(previous => ({ ...previous, [batchId]: value }));
    setError('');
  }
  async function reconfirm() {
    setError('');
    try {
      const [stock, warehouseResult] = await Promise.all([fresh.refetch(), warehouses.refetch()]);
      const next = stock.data?.data;
      if (!next) { setError('Не удалось получить обновлённые остатки. Повторите загрузку.'); return; }
      const available = new Set(next.vehicles.map(vehicle => vehicle.id));
      const availableBatches = new Set(next.unidentifiedBatches.map(batch => batch.id));
      if (selected.some(id => !available.has(id)) || Object.entries(quantities).some(([id, quantity]) => positiveInteger(quantity) && !availableBatches.has(id))) {
        setError('Состав выбранного запаса изменился. Уберите недоступные позиции или вернитесь к обзору.'); return;
      }
      setSnapshot(capture(next.warehouse, next.vehicles, next.unidentifiedBatches, warehouseResult.data ?? []));
      setNeedsReconfirm(false);
    } catch (caught) { setError(`Не удалось обновить факты: ${errorText(caught)}`); }
  }
  async function submit() {
    if (sending.current || busy) return;
    if (!canMove) { setError('Недостаточно прав для перемещения автомобилей.'); return; }
    if (needsReconfirm) { setError('Подтвердите обновлённые остатки перед повторной отправкой.'); return; }
    if (!destinationId) { setError('Выберите склад назначения.'); return; }
    if (!occurredAt || !Number.isFinite(new Date(occurredAt).getTime())) { setError('Укажите дату и время перемещения.'); return; }
    const enteredBatches = Object.entries(quantities).filter(([, quantity]) => quantity.trim());
    if (selected.length > limit || enteredBatches.length > limit || total > 10000) { setError('Выбранный объём превышает лимит одного перемещения. Уменьшите выбор.'); return; }
    for (const [batchId, quantity] of enteredBatches) {
      const batch = batches.find(item => item.id === batchId) ?? snapshot.batches.get(batchId);
      if (!positiveInteger(quantity) || !batch || Number(quantity) > Number(batch.warehouseUnidentifiedCount)) { setError('Для каждой партии укажите положительное целое количество в пределах остатка на этом складе.'); return; }
    }
    if (!total) { setError('Выберите хотя бы один VIN или количество партии.'); return; }
    const destination = snapshot.warehouses.get(destinationId);
    if (!destination) { setError('Данные склада назначения устарели. Подтвердите обновлённые остатки.'); setNeedsReconfirm(true); return; }
    const missing = selected.filter(id => !snapshot.vehicles.has(id));
    if (missing.length) { setError('Выбранные VIN изменились. Подтвердите обновлённые остатки.'); setNeedsReconfirm(true); return; }
    sending.current = true; setBusy(true); onBusy(true); setError('');
    try {
      await post(`/inventory/warehouses/${source.id}/stock-moves`, {
        toWarehouseId: destinationId,
        toWarehouseRevision: destination.revision,
        occurredAt: new Date(occurredAt).toISOString(),
        vehicleIds: selected,
        unidentified: selectedBatches.map(batch => ({ receiptBatchId: batch.id, quantity: quantities[batch.id], revision: snapshot.batches.get(batch.id)?.revision ?? batch.revision })),
      }, { ifMatch: snapshot.sourceRevision });
    } catch (caught) {
      sending.current = false; setBusy(false); onBusy(false);
      if (caught instanceof ApiError && [409, 412].includes(caught.status)) {
        setNeedsReconfirm(true);
        setError(`${caught.message}. Остатки обновляются; проверьте выбранные позиции и подтвердите их перед повторной отправкой.`);
        void Promise.all([fresh.refetch(), warehouses.refetch()]);
      } else if (caught instanceof ApiError && caught.status < 500) {
        setError(caught.message);
      } else {
        setError(`${errorText(caught)}. Статус перемещения неизвестен; остатки обновляются. Не повторяйте отправку автоматически.`);
        setNeedsReconfirm(true);
        void Promise.all([fresh.refetch(), warehouses.refetch()]);
      }
      return;
    }
    let message = `Перемещено единиц: ${total}.`;
    try { await reload(['stock'], ['warehouses'], ['vehicles'], ['vehicle'], ['receipt-batch'], ['order'], ['orders']); }
    catch { message += ' Команда выполнена, но обновить списки не удалось.'; }
    onDone(message);
  }

  return <section className="warehouse-move kit-stack">
    <header className="warehouse-move-head"><h3>Переместить автомобили</h3><p>Переместите VIN и партии без VIN между вашими складами за один раз.</p></header>
    {!canMove && <Notice kind="warning">У вас нет разрешения на перемещение автомобилей.</Notice>}
    {error && <Notice kind="danger">{error}</Notice>}
    {needsReconfirm && <Notice kind="warning">Данные изменились. Выбор сохранён, но перед отправкой подтвердите обновлённые остатки.</Notice>}
    {fresh.error && <Notice kind="danger">Не удалось обновить остатки: {errorText(fresh.error)} <Button onClick={() => void fresh.refetch()}>Повторить</Button></Notice>}
    {warehouses.error && <Notice kind="danger">Не удалось загрузить склады: {errorText(warehouses.error)} <Button onClick={() => void warehouses.refetch()}>Повторить</Button></Notice>}
    <fieldset disabled={busy || !canMove}>
      <div className="warehouse-move-fields">
        <label>Склад отправления<span>{source.name}</span></label>
        <label>Склад назначения<select aria-label="Склад назначения" value={destinationId} onChange={event => { const id = event.target.value; const warehouse = destinations.find(item => item.id === id); setDestinationId(id); if (warehouse) setSnapshot(previous => ({ ...previous, warehouses: new Map(previous.warehouses).set(id, warehouse) })); setError(''); }}><option value="">Выберите склад</option>{destinations.map(warehouse => <option key={warehouse.id} value={warehouse.id}>{warehouse.name} · свободно {warehouse.free}</option>)}</select></label>
        <label>Дата перемещения<input aria-label="Дата перемещения" type="datetime-local" value={occurredAt} onChange={event => setOccurredAt(event.target.value)} /></label>
        <label>Всего единиц<span>{total}</span></label>
      </div>
      {!warehouses.isLoading && !warehouses.error && !destinations.length && <Notice>Нет другого доступного склада назначения.</Notice>}
      <section className="warehouse-move-section">
        <div className="warehouse-move-section-head"><div><h4>Автомобили с VIN</h4><p>Загружено {vehicles.length} VIN. Выбор не меняется сам при обновлении.</p></div><div className="kit-row"><Button onClick={selectAll} disabled={!vehicles.length}>Выбрать все загруженные</Button><Button onClick={clearAll} disabled={!selected.length}>Очистить</Button></div></div>
        <div className="warehouse-move-first-n"><label>Первые N VIN по возрастанию<input aria-label="Количество первых VIN" inputMode="numeric" value={firstN} onChange={event => setFirstN(event.target.value)} /></label><Button onClick={chooseFirstN}>Выбрать N</Button></div>
        {!vehicles.length ? <p className="cell-sub">На этом складе нет загруженных VIN.</p> : <div className="warehouse-move-list">{vehicles.slice().sort(byVin).map(vehicle => <label className="warehouse-move-item" key={vehicle.id}><input type="checkbox" aria-label={`Выбрать ${vehicle.vin}`} checked={selected.includes(vehicle.id)} onChange={() => select(vehicle.id)} /><code>{vehicle.vin}</code><span>{modelName(vehicle.modelId)}</span><small>{vehicleColorsLabel(vehicle)}</small></label>)}</div>}
      </section>
      <section className="warehouse-move-section">
        <div className="warehouse-move-section-head"><div><h4>Партии без VIN</h4><p>Количество ограничено остатком на этом складе.</p></div></div>
        {!batches.length ? <p className="cell-sub">Нет партий без VIN на этом складе.</p> : <div className="warehouse-move-list">{batches.map(batch => <label className="warehouse-move-batch" key={batch.id}><span><strong>{modelName(batch.modelId)}</strong><small>{vehicleColorsLabel(batch)} · партия {batch.id.slice(0, 8)} · принято {batch.confirmedQuantity}</small></span><span>Здесь доступно {batch.warehouseUnidentifiedCount}</span><input aria-label={`Количество партии ${batch.id}`} inputMode="numeric" value={quantities[batch.id] ?? ''} onChange={event => setQuantity(batch.id, event.target.value)} /></label>)}</div>}
      </section>
    </fieldset>
    <section className="warehouse-move-selected"><h4>Выбрано для перемещения</h4>{!selectedVehicles.length && !selectedBatches.length ? <p className="cell-sub">Выберите VIN или укажите количество партии.</p> : <ul>{selectedVehicles.map(({ vehicle, unavailable }) => <li key={vehicle.id}><Button onClick={() => onVehicle(vehicle.id)} disabled={unavailable || busy}>{vehicle.vin}</Button> · {modelName(vehicle.modelId)} · {vehicleColorsLabel(vehicle)} {unavailable && <span className="warehouse-move-unavailable">больше нет в текущих остатках</span>} <Button onClick={() => select(vehicle.id)} disabled={busy}>Убрать</Button></li>)}{selectedBatches.map(batch => <li key={batch.id}>{modelName(batch.modelId)} · {vehicleColorsLabel(batch)} · {quantities[batch.id]} шт. <Button onClick={() => setQuantity(batch.id, '')} disabled={busy}>Убрать</Button></li>)}</ul>}</section>
    <div className="warehouse-move-footer"><Button icon="back" disabled={busy} onClick={onBack}>Назад</Button>{needsReconfirm ? <Button variant="primary" disabled={busy} onClick={() => void reconfirm()}>Подтвердить обновлённые остатки</Button> : <Button variant="primary" busy={busy} disabled={busy || !canMove || !destinations.length} onClick={() => void submit()}>Переместить {total || ''}</Button>}</div>
  </section>;
}

function capture(source: Warehouse, vehicles: Vehicle[], batches: WarehouseMoveBatch[], warehouses: Warehouse[]) {
  return { sourceRevision: source.revision, vehicles: new Map(vehicles.map(vehicle => [vehicle.id, vehicle])), batches: new Map(batches.map(batch => [batch.id, batch])), warehouses: new Map(warehouses.map(warehouse => [warehouse.id, warehouse])) };
}
