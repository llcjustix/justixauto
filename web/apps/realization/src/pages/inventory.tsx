import { useRef, useState } from 'react';
import { ReceiptIdentification } from '../receipt-identification';
import {
  ActionButton,
  Autocomplete,
  Button,
  Notice,
  ApiError,
  Details,
  Modal,
  Panel,
  Stat,
  Table,
  date,
  dateTime,
  get,
  patch,
  post,
  useData,
  useRefresh,
} from '@justixauto/kit';
import {
  modelName,
  useModelName,
  useModels,
  useModelDetail,
  useOrderModels,
  useWarehouses,
  useBranches,
  warehouseFields,
  warehouseInput,
} from '../data';
import type { Model, Vehicle, Warehouse } from '../data';
import { colorPalette, exactSpecification, selectedColor, vehicleColorsLabel } from '../vehicle-colors';
import { validateVins, VinEditor } from '../vin-editor';

interface Batch {
  id: string;
  modelId: string;
  modelSpecificationVersion: string;
  confirmedQuantity: string;
  identifiedCount: string;
  unidentifiedCount: string;
  receivedAt: string;
  revision: string;
  exteriorColor?: string;
  interiorColor?: string;
}
interface Stock {
  warehouse: Warehouse;
  vehicles: Vehicle[];
  unidentifiedBatches: Batch[];
}

export function WarehouseDialog({ id, onClose }: { id: string; onClose: () => void }) {
  const q = useData(['stock', id], () => get<Stock>(`/inventory/warehouses/${id}/inventory`));
  const models = useModels();
  const orderModels = useOrderModels();
  const name = (modelId: string) => modelName(orderModels.data?.find((m) => m.id === modelId));
  const [identifying, setIdentifying] = useState<Batch | null>(null);
  const [receiving, setReceiving] = useState(false);
  const [receivingBusy, setReceivingBusy] = useState(false);
  const [success, setSuccess] = useState('');
  const s = q.data?.data;
  const w = s?.warehouse;
  const refresh = [['stock', id], ['warehouses'], ['vehicles']];
  const branches = useBranches();
  return (
    <Modal
      title={w?.name ?? 'Склад'}
      onClose={() => { if (!receivingBusy) onClose(); }}
      size="wide"
      footer={
        w && !identifying && !receiving && (
          <>
            <Button variant="primary" onClick={() => { setSuccess(''); setReceiving(true); }}>Принять автомобили</Button>
            <ActionButton
              label="Изменить вместимость"
              refresh={refresh}
              fields={[
                { name: 'capacity', label: 'Новая вместимость', type: 'number', required: true, initial: w.capacity },
                { name: 'reason', label: 'Основание', type: 'textarea', required: true },
              ]}
              onSubmit={(v) => post(`/inventory/warehouses/${id}/capacity-changes`, v, { ifMatch: w.revision })}
            />
            <ActionButton
              label="Реквизиты"
              fields={warehouseFields(w)}
              refresh={refresh}
              onSubmit={(v) => patch(`/inventory/warehouses/${id}`, warehouseInput(v), { ifMatch: w.revision })}
            />
            <ActionButton
              label="Филиал"
              refresh={refresh}
              intro={
                <p>У филиала может быть один основной склад. Отвязка сохраняет склад, вместимость и автомобили.</p>
              }
              fields={[
                {
                  name: 'branchId',
                  label: 'Основной склад филиала',
                  type: 'select',
                  initial: w.branchId ?? '',
                  options: (branches.data ?? []).map((b) => [b.id, b.name]),
                },
              ]}
              onSubmit={(v) =>
                post(
                  `/inventory/warehouses/${id}/branch-attachment`,
                  { branchId: v.branchId || null },
                  { ifMatch: w.revision },
                )
              }
            />
          </>
        )
      }
    >
      {success && <Notice kind="success">{success}</Notice>}
      {receiving && w ? <ManualReceipt warehouse={w} models={models.data ?? []} modelsLoading={models.isLoading} modelsError={!!models.error} retryModels={() => void models.refetch()} refresh={refresh} onBusy={setReceivingBusy} onBack={() => setReceiving(false)} onDone={message => { setSuccess(message); setReceiving(false); }} /> : identifying ? <ReceiptIdentification key={identifying.id} batch={{ ...identifying, warehouseId: id }} modelName={name(identifying.modelId)} refresh={refresh}
        onBack={() => setIdentifying(null)} onDone={(message) => { setSuccess(message); setIdentifying(null); }} /> : <>
      {w && (
        <div className="kit-grid">
          <Stat label="Вместимость" value={w.capacity} />
          <Stat label="Занято" value={w.occupied} />
          <Stat label="Свободно" value={w.free} />
        </div>
      )}
      <Panel title="Автомобили">
        <Table
          rows={s?.vehicles}
          rowKey={(v) => v.id}
          columns={[
            { title: 'VIN', render: (v) => <code>{v.vin}</code> },
            { title: 'Модель', render: (v) => name(v.modelId) },
            { title: 'Цвета', render: (v) => vehicleColorsLabel(v) },
            { title: 'Размещён', render: (v) => date(v.placement?.placedAt) },
          ]}
          empty="На складе нет автомобилей с VIN"
        />
      </Panel>
      <Panel title="Ожидают ввода VIN">
        <Table
          rows={s?.unidentifiedBatches}
          rowKey={(b) => b.id}
          columns={[
            { title: 'Модель', render: (b) => name(b.modelId) },
            { title: 'Цвета', render: (b) => vehicleColorsLabel(b) },
            { title: 'Принято', render: (b) => b.confirmedQuantity },
            { title: 'Без VIN', render: (b) => b.unidentifiedCount },
            { title: 'Дата', render: (b) => date(b.receivedAt) },
            {
              title: '',
              render: (b) => (
                <div className="kit-row">
                  <Button onClick={() => { setSuccess(''); setIdentifying(b); }}>Ввести VIN</Button>
                  <ActionButton
                    label="Исправить кол-во"
                    refresh={refresh}
                    intro={
                      <p>
                        Пересчёт партии фиксируется в истории склада. Нельзя указать меньше, чем уже введено VIN (
                        {b.identifiedCount}).
                      </p>
                    }
                    fields={[
                      {
                        name: 'quantity',
                        label: 'Верное количество',
                        type: 'number',
                        required: true,
                        initial: b.confirmedQuantity,
                      },
                      { name: 'reason', label: 'Основание', type: 'textarea', required: true },
                    ]}
                    onSubmit={(v) =>
                      post(`/inventory/receipt-batches/${b.id}/quantity-corrections`, v, { ifMatch: b.revision })
                    }
                  />
                </div>
              ),
            },
          ]}
          empty="Нет партий без VIN"
        />
      </Panel>
      </>}
    </Modal>
  );
}

function ManualReceipt({ warehouse, models, modelsLoading, modelsError, retryModels, refresh, onBusy, onBack, onDone }: {
  warehouse: Warehouse; models: Model[]; modelsLoading: boolean; modelsError: boolean; retryModels: () => void;
  refresh: unknown[][]; onBusy: (busy: boolean) => void; onBack: () => void; onDone: (message: string) => void;
}) {
  const [modelId, setModelId] = useState('');
  const [version, setVersion] = useState('');
  const [colors, setColors] = useState({ exteriorColor: '', interiorColor: '', exteriorEdited: false, interiorEdited: false });
  const [mode, setMode] = useState('identified');
  const [rows, setRows] = useState(['']);
  const [quantity, setQuantity] = useState('');
  const [receivedAt, setReceivedAt] = useState('');
  const [busy, setBusy] = useState(false);
  const [error, setError] = useState('');
  const sending = useRef(false);
  const refreshQueries = useRefresh();
  const detail = useModelDetail(modelId);
  const spec = exactSpecification(detail.data, version);
  const exteriorOptions = colorPalette(spec, 'exterior');
  const interiorOptions = colorPalette(spec, 'interior');
  const exterior = colors.exteriorEdited ? exteriorOptions.find(value => value === colors.exteriorColor) ?? '' : selectedColor(exteriorOptions, colors.exteriorColor);
  const interior = colors.interiorEdited ? interiorOptions.find(value => value === colors.interiorColor) ?? '' : selectedColor(interiorOptions, colors.interiorColor);
  const ready = !!modelId && !!spec && !!exterior && !!interior && !detail.isFetching && !detail.isError;

  async function submit(event: React.FormEvent) {
    event.preventDefault();
    if (sending.current) return;
    if (!ready) { setError('Выберите модель и оба цвета из точной спецификации'); return; }
    if (mode !== 'identified' && mode !== 'unidentified') { setError('Выберите режим приёмки'); return; }
    if (!receivedAt || !Number.isFinite(new Date(receivedAt).getTime())) { setError('Укажите дату приёмки'); return; }
    const checked = validateVins(rows, 1000, true);
    if (mode === 'identified' && !checked.valid) { setError(checked.errors.join(' · ')); return; }
    if (mode !== 'identified' && (!/^[1-9]\d*$/.test(quantity) || !Number.isSafeInteger(Number(quantity)))) { setError('Введите целое количество'); return; }
    sending.current = true; setBusy(true); onBusy(true); setError('');
    try {
      await post(`/inventory/warehouses/${warehouse.id}/receipt-batches`, {
        modelId, modelSpecificationVersion: version, exteriorColor: exterior, interiorColor: interior,
        stock: mode === 'identified' ? { mode, vins: checked.vins } : { mode, quantity },
        receivedAt: new Date(receivedAt).toISOString(),
      }, { ifMatch: warehouse.revision });
    } catch (caught) {
      setError(caught instanceof ApiError ? [caught.message, ...Object.values(caught.fields)].join(' · ') : caught instanceof Error ? caught.message : 'Не удалось принять автомобили');
      if (caught instanceof ApiError && [409, 412].includes(caught.status)) {
        try { await refreshQueries(...refresh); } catch { setError(previous => `${previous}. Обновление не удалось`); }
      }
      sending.current = false; setBusy(false); onBusy(false); return;
    }
    let message = `Автомобили приняты. ${vehicleColorsLabel({ exteriorColor: exterior, interiorColor: interior, modelSpecificationVersion: version })}.`;
    try { await refreshQueries(...refresh); } catch { message += ' Не удалось обновить списки; обновите склад.'; }
    onBusy(false); onDone(message);
  }

  return <form className="kit-stack" onSubmit={submit} noValidate>
    <h3>Принять автомобили</h3>
    {error && <Notice kind="danger">{error}</Notice>}
    <fieldset disabled={busy}>
      <label>Модель<Autocomplete aria-label="Модель" required value={modelId} onChange={id => {
        setModelId(id); setVersion(String(models.find(model => model.id === id)?.specification.version ?? '')); setColors({ exteriorColor: '', interiorColor: '', exteriorEdited: false, interiorEdited: false });
      }} options={[{ value: '', label: 'Выберите модель' }, ...models.map(model => ({ value: model.id, label: modelName(model) }))]} /></label>
      {modelsLoading && <p role="status">Загрузка моделей…</p>}
      {modelsError && <Notice kind="danger">Не удалось загрузить модели. <Button onClick={retryModels}>Повторить загрузку моделей</Button></Notice>}
      {modelId && <>
        <p>Версия спецификации: {version || 'Не указана'}. Одна партия — один цвет кузова и салона.</p>
        {detail.isFetching && <p role="status">Загрузка спецификации…</p>}
        {(detail.isError || (!detail.isFetching && !spec)) && <Notice kind="danger">Точная версия спецификации недоступна. <Button onClick={() => void detail.refetch()}>Повторить загрузку спецификации</Button></Notice>}
        <label>Цвет кузова<Autocomplete aria-label="Цвет кузова" required disabled={detail.isFetching} value={exterior} onChange={value => setColors(previous => ({ ...previous, exteriorColor: value, exteriorEdited: true }))} options={[{ value: '', label: 'Выберите цвет' }, ...exteriorOptions.map(value => ({ value, label: value }))]} /></label>
        <label>Цвет салона<Autocomplete aria-label="Цвет салона" required disabled={detail.isFetching} value={interior} onChange={value => setColors(previous => ({ ...previous, interiorColor: value, interiorEdited: true }))} options={[{ value: '', label: 'Выберите цвет' }, ...interiorOptions.map(value => ({ value, label: value }))]} /></label>
      </>}
      <label>Приёмка<Autocomplete aria-label="Приёмка" required value={mode} onChange={setMode} options={[{ value: 'identified', label: 'С VIN' }, { value: 'unidentified', label: 'Количество, VIN позже' }]} /></label>
      {mode === 'identified' ? <VinEditor rows={rows} onChange={setRows} cap={1000} required disabled={busy} /> : mode === 'unidentified' && <label>Количество (без VIN)<input inputMode="numeric" value={quantity} onChange={event => setQuantity(event.target.value)} /></label>}
      <label>Дата приёмки<input type="datetime-local" value={receivedAt} onChange={event => setReceivedAt(event.target.value)} /></label>
      <p>{vehicleColorsLabel({ exteriorColor: exterior, interiorColor: interior, modelSpecificationVersion: version })}</p>
    </fieldset>
    <div className="kit-row"><Button icon="back" disabled={busy} onClick={onBack}>Назад</Button><Button type="submit" variant="primary" busy={busy} disabled={!ready}>Принять автомобили</Button></div>
  </form>;
}

interface VehicleDetail {
  vehicle: Vehicle;
  specification: Model['specification'];
  history: { type: string; occurredAt: string; warehouseId: string | null; reason: string }[];
}

const factLabel: Record<string, string> = {
  'vehicle.received': 'Принят на склад',
  'vehicle.identified': 'VIN введён',
  'vehicle.moved': 'Перемещён',
  'vehicle.handed_over': 'Получен от поставщика',
  'vehicle.delivered_to_customer': 'Выдан клиенту',
};

export function VehicleDialog({ id, onClose }: { id: string; onClose: () => void }) {
  const q = useData(['vehicle', id], () => get<VehicleDetail>(`/inventory/vehicle-units/${id}`));
  const warehouses = useWarehouses();
  const d = q.data?.data;
  const wh = (x?: string | null) => warehouses.data?.find((w) => w.id === x)?.name ?? '—';
  return (
    <Modal
      title={d ? `VIN ${d.vehicle.vin}` : 'Автомобиль'}
      onClose={onClose}
      size="wide"
      footer={
        d?.vehicle.placement && (
          <ActionButton
            label="Переместить"
            refresh={[['vehicle', id], ['vehicles'], ['warehouses']]}
            fields={[
              {
                name: 'toWarehouseId',
                label: 'На склад',
                type: 'select',
                required: true,
                options: (warehouses.data ?? [])
                  .filter((w) => w.id !== d.vehicle.placement!.warehouseId)
                  .map((w) => [w.id, `${w.name} (свободно ${w.free})`]),
              },
              { name: 'occurredAt', label: 'Когда', type: 'datetime', required: true },
            ]}
            onSubmit={(v) =>
              post(`/inventory/vehicle-units/${id}/warehouse-moves`, {
                fromWarehouseId: d.vehicle.placement!.warehouseId,
                ...v,
              })
            }
          />
        )
      }
    >
      {d && (
        <>
          <Details
            items={[
              [
                'Модель',
                `${d.specification.make} ${d.specification.model} ${d.specification.variant} (${d.specification.year})`,
              ],
              [
                'Кузов / цвет',
                `${d.specification.bodyType} · ${vehicleColorsLabel(d.vehicle)}`,
              ],
              ['Двигатель / привод', `${d.specification.powertrain}, ${d.specification.drivetrain}`],
              ['Склад', wh(d.vehicle.placement?.warehouseId)],
            ]}
          />
          <Panel title="История" padded>
            <ul className="kit-timeline">
              {d.history.map((h, i) => (
                <li key={i}>
                  {dateTime(h.occurredAt)} — {factLabel[h.type] ?? h.type}
                  {h.warehouseId ? ` · ${wh(h.warehouseId)}` : ''}
                  {h.reason ? ` · ${h.reason}` : ''}
                </li>
              ))}
            </ul>
          </Panel>
        </>
      )}
    </Modal>
  );
}

/** The shared car catalog, read-only: the platform admin maintains it. */
export function ModelsPanel() {
  const q = useModels();
  return (
    <Panel title="Каталог моделей">
      <Table
        rows={q.data}
        loading={q.isLoading}
        error={q.error}
        rowKey={(m) => m.id}
        empty="Каталог пока пуст: модели добавляет администратор платформы"
        columns={[
          { title: 'Модель', render: (m) => <b>{modelName(m)}</b> },
          { title: 'Год', render: (m) => m.specification.year },
          { title: 'Версия', render: (m) => m.specification.version },
        ]}
      />
    </Panel>
  );
}
