import { useRef, useState } from 'react';
import { Button, Modal, Notice, Table, errorText, money, post, put, toMinor, useCommercialAccess, useRefresh, useSession } from '@justixauto/kit';
import { modelName, routeLabel, useEligibleVehicles, useModelDetail, useOrderModels, useReservationAllocations, useReservationPartners, useReservationDetail, useReservationStock, useReservations, useSupplyLots } from './data';
import type { Reservation, ReservationContinuation, ReservationLine, ReservationStockVehicle, ReservationTerms, SupplyLot, Terms } from './data';
import { colorOptions, exactSpecification } from './vehicle-colors';
import { ReservationPayment } from './reservation-payment';
import { PaymentScheduleEditor, TermsView } from './shared';

const quantityOK = (s: string) => /^[1-9]\d*$/.test(s) && BigInt(s) <= 2147483647n;
const iso = (s: string) => s ? new Date(s).toISOString() : undefined;
const localDate = (s?: string | null) => {
  if (!s) return '';
  const d = new Date(s);
  const pad = (n: number) => String(n).padStart(2, '0');
  return `${d.getFullYear()}-${pad(d.getMonth() + 1)}-${pad(d.getDate())}T${pad(d.getHours())}:${pad(d.getMinutes())}`;
};
const amountInput = (s?: { amountMinor: string }) => {
  if (!s) return '';
  const digits = s.amountMinor.padStart(3, '0');
  return digits.slice(0, -2) + '.' + digits.slice(-2);
};
const allowed = (session: ReturnType<typeof useSession>, access: ReturnType<typeof useCommercialAccess>) =>
  !!session.company && access.allowed && session.can('commerce.trade');

function Load({ loading, error, retry, empty }: { loading: boolean; error: unknown; retry: () => unknown; empty?: boolean }) {
  if (loading) return <Notice>Загрузка…</Notice>;
  if (error) return <Notice kind="error">{errorText(error)} <Button onClick={() => void retry()}>Повторить</Button></Notice>;
  return empty ? <Notice>Данных пока нет</Notice> : null;
}

export function ReservationWorkspace({ onClose, onOpenOrder }: { onClose: () => void; onOpenOrder?: (id: string) => void }) {
  const session = useSession(); const access = useCommercialAccess();
  const reservations = useReservations(); const lots = useSupplyLots(); const models = useOrderModels();
  const [mode, setMode] = useState<'list' | 'supply' | 'request'>('list');
  const [selectedId, setSelectedId] = useState(''); const [edit, setEdit] = useState<SupplyLot | null>(null);
  const type = session.company?.userType;
  if (selectedId) return <ReservationDetail id={selectedId} onClose={() => setSelectedId('')} onOpenOrder={onOpenOrder} />;
  return <Modal title="Резервы и поставки" size="wide" onClose={onClose} footer={<Button onClick={onClose}>Закрыть</Button>}>
    {!access.allowed && <Notice kind="warning">{access.reason}</Notice>}
    <div className="kit-row">
      {type === 'dealer' && <Button disabled={!allowed(session, access)} onClick={() => setMode('request')}>Запросить резерв</Button>}
      {type === 'distributor' && <Button disabled={!allowed(session, access)} onClick={() => { setEdit(null); setMode('supply'); }}>Добавить будущую поставку</Button>}
      {mode !== 'list' && <Button onClick={() => setMode('list')}>К списку</Button>}
    </div>
    {mode === 'supply' && type === 'distributor' && <SupplyForm key={edit?.id ?? 'new'} lot={edit} onDone={() => setMode('list')} />}
    {mode === 'request' && type === 'dealer' && <RequestForm onDone={id => { setMode('list'); setSelectedId(id); }} />}
    {mode === 'list' && <>
      <h3>Резервы</h3>
      <Table rows={reservations.data} loading={reservations.isLoading} error={reservations.error} rowKey={r => r.id}
        onRowClick={r => setSelectedId(r.id)} empty="Резервов пока нет" columns={[
          { title: 'Покупатель', render: r => r.buyer.name }, { title: 'Поставщик', render: r => r.supplier.name },
          { title: 'Статус', render: r => r.status }, { title: 'Количество', render: r => r.requested.lines.map(l => l.quantity).join(', ') },
          { title: 'Действие', render: r => <Button size="sm" onClick={() => setSelectedId(r.id)}>Открыть</Button> },
        ]} />
      {type === 'distributor' && <><h3>Будущие поставки</h3>
        <Table rows={lots.data?.filter(l => l.supplierCompanyId === session.company?.id)} loading={lots.isLoading} error={lots.error} rowKey={l => l.id}
          empty="Будущих поставок пока нет" columns={[
            { title: 'Модель', render: l => modelName(models.data?.find(m => m.id === l.modelId)) }, { title: 'Доступно', render: l => `${l.availableQuantity} из ${l.totalQuantity}` },
            { title: 'Прибытие', render: l => l.expectedArrival ? new Date(l.expectedArrival).toLocaleString() : '—' },
            { title: 'Действие', render: l => <Button size="sm" disabled={!allowed(session, access)} onClick={() => { setEdit(l); setMode('supply'); }}>Изменить</Button> },
          ]} />
      </>}
    </>}
  </Modal>;
}

function SupplyForm({ lot, onDone }: { lot: SupplyLot | null; onDone: () => void }) {
  const session = useSession(); const access = useCommercialAccess(); const refresh = useRefresh();
  const models = useOrderModels();
  const [modelId, setModelId] = useState(lot?.modelId ?? ''); const detail = useModelDetail(modelId);
  const [version, setVersion] = useState(lot?.modelSpecificationVersion ?? '');
  const [exterior, setExterior] = useState(lot?.exteriorColor ?? ''); const [interior, setInterior] = useState(lot?.interiorColor ?? '');
  const [quantity, setQuantity] = useState(lot?.totalQuantity ?? ''); const [arrival, setArrival] = useState(localDate(lot?.expectedArrival));
  const [error, setError] = useState(''); const [busy, setBusy] = useState(false); const sending = useRef(false);
  const locked = !!lot && (BigInt(lot.heldQuantity) > 0n || BigInt(lot.materializedQuantity) > 0n);
  const spec = exactSpecification(detail.data, version);
  const exteriorChoices = colorOptions(spec, 'exterior'); const interiorChoices = colorOptions(spec, 'interior');
  async function submit() {
    if (sending.current || !allowed(session, access) || session.company?.userType !== 'distributor') return;
    if (models.isLoading || models.error || detail.isLoading || detail.error || !models.data?.some(m => m.id === modelId) || !spec ||
      !exteriorChoices.includes(exterior) || !interiorChoices.includes(interior) || !quantityOK(quantity) || !arrival) {
      setError('Выберите модель, точную спецификацию, оба цвета, количество и дату прибытия.'); return;
    }
    sending.current = true; setBusy(true); setError('');
    const body = { modelId, modelSpecificationVersion: version, exteriorColor: exterior, interiorColor: interior,
      totalQuantity: quantity, expectedArrival: iso(arrival) };
    try {
      if (lot) await put(`/commerce/supply-lots/${lot.id}`, body, { ifMatch: lot.revision });
      else await post('/commerce/supply-lots', body);
      await refresh(['supply-lots', session.company.id]); onDone();
    } catch (e) {
      setError(errorText(e)); sending.current = false; setBusy(false);
      if (lot) await refresh(['supply-lots', session.company.id]);
    }
  }
  return <section><h3>{lot ? 'Изменить будущую поставку' : 'Будущая поставка'}</h3>
    {error && <Notice kind="error">{error}</Notice>}
    <Load loading={models.isLoading} error={models.error} retry={models.refetch} empty={models.data?.length === 0} />
    <label>Модель<select aria-label="Модель" value={modelId} disabled={locked || !!models.error} onChange={e => { setModelId(e.target.value); setVersion(''); setExterior(''); setInterior(''); }}>
      <option value="">Выберите</option>{(models.data ?? []).map(m => <option key={m.id} value={m.id}>{modelName(m)}</option>)}</select></label>
    {modelId && <Load loading={detail.isLoading} error={detail.error} retry={detail.refetch} />}
    <label>Версия спецификации<select aria-label="Версия спецификации" value={version} disabled={locked || !detail.data} onChange={e => { setVersion(e.target.value); setExterior(''); setInterior(''); }}>
      <option value="">Выберите</option>{(detail.data?.versions ?? []).map(s => <option key={s.version} value={s.version}>{s.version} · {s.make} {s.model} {s.variant}</option>)}</select></label>
    <label>Цвет кузова<select aria-label="Цвет кузова" value={exterior} disabled={locked || !spec} onChange={e => setExterior(e.target.value)}>
      <option value="">Выберите</option>{exteriorChoices.map(c => <option key={c}>{c}</option>)}</select></label>
    <label>Цвет салона<select aria-label="Цвет салона" value={interior} disabled={locked || !spec} onChange={e => setInterior(e.target.value)}>
      <option value="">Выберите</option>{interiorChoices.map(c => <option key={c}>{c}</option>)}</select></label>
    <label>Общее количество<input aria-label="Общее количество" inputMode="numeric" value={quantity} onChange={e => setQuantity(e.target.value)} /></label>
    {lot && <p>Доступно: {lot.availableQuantity}; удержано: {lot.heldQuantity}; получено: {lot.materializedQuantity}</p>}
    <label>Ожидаемое прибытие<input aria-label="Ожидаемое прибытие" type="datetime-local" value={arrival} onChange={e => setArrival(e.target.value)} /></label>
    <Button variant="primary" disabled={busy || !allowed(session, access) || !!detail.error} onClick={() => void submit()}>{busy ? 'Сохранение…' : 'Сохранить поставку'}</Button>
  </section>;
}

type TermDraft = { price: string; currency: string; expiry: string; advance: string; advanceDue: string };
const blankTerms = (): TermDraft => ({ price: '', currency: 'USD', expiry: '', advance: '', advanceDue: '' });
const fromTerms = (t: ReservationTerms): TermDraft => ({ price: amountInput(t.unitPrice), currency: t.unitPrice?.currency ?? 'USD',
  expiry: localDate(t.expiresAt), advance: amountInput(t.advance), advanceDue: localDate(t.advanceDueAt) });
function termPayload(d: TermDraft, required: boolean): ReservationTerms | string {
  const price = d.price ? toMinor(d.price) : null; const advance = d.advance ? toMinor(d.advance) : null;
  if (required && (!price || BigInt(price) <= 0n || !d.expiry)) return 'Укажите положительную цену и срок действия.';
  if (d.price && (!price || BigInt(price) <= 0n)) return 'Цена должна быть положительной точной суммой.';
  if (d.advance && (!advance || BigInt(advance) <= 0n || !d.advanceDue)) return 'Для аванса укажите положительную сумму и срок оплаты.';
  if (!d.advance && d.advanceDue) return 'Срок аванса требует сумму аванса.';
  const expiry = iso(d.expiry); const advanceDue = iso(d.advanceDue);
  if (expiry && new Date(expiry).getTime() <= Date.now()) return 'Срок действия должен быть в будущем.';
  if (advanceDue && (new Date(advanceDue).getTime() <= Date.now() || (expiry && advanceDue > expiry))) return 'Срок аванса должен быть в будущем и не позже срока действия.';
  return { ...(price ? { unitPrice: { amountMinor: price, currency: d.currency } } : {}),
    ...(expiry ? { expiresAt: expiry } : {}), ...(advance ? { advance: { amountMinor: advance, currency: d.currency }, advanceDueAt: advanceDue } : {}) };
}
function TermsFields({ value, onChange, desired = false }: { value: TermDraft; onChange: (next: TermDraft) => void; desired?: boolean }) {
  const change = (key: keyof TermDraft, text: string) => onChange({ ...value, [key]: text });
  return <fieldset><legend>{desired ? 'Желаемые условия (можно оставить пустыми)' : 'Предложенные условия'}</legend>
    <label>Цена за единицу<input aria-label="Цена за единицу" inputMode="decimal" value={value.price} onChange={e => change('price', e.target.value)} /></label>
    <label>Валюта<select aria-label="Валюта" value={value.currency} onChange={e => change('currency', e.target.value)}>{['USD', 'UZS', 'EUR'].map(c => <option key={c}>{c}</option>)}</select></label>
    <label>Срок действия<input aria-label="Срок действия" type="datetime-local" value={value.expiry} onChange={e => change('expiry', e.target.value)} /></label>
    <label>Аванс<input aria-label="Аванс" inputMode="decimal" value={value.advance} onChange={e => change('advance', e.target.value)} /></label>
    <label>Срок аванса<input aria-label="Срок аванса" type="datetime-local" value={value.advanceDue} onChange={e => change('advanceDue', e.target.value)} /></label>
  </fieldset>;
}

function RequestForm({ onDone }: { onDone: (id: string) => void }) {
  const session = useSession(); const access = useCommercialAccess(); const refresh = useRefresh();
  const partners = useReservationPartners(); const lots = useSupplyLots(); const models = useOrderModels();
  const [supplier, setSupplier] = useState(''); const [modelId, setModelId] = useState(''); const [offset, setOffset] = useState(0);
  const stock = useReservationStock(supplier, modelId, offset);
  const [vehicleId, setVehicleId] = useState(''); const [selectedVehicles, setSelectedVehicles] = useState<ReservationStockVehicle[]>([]);
  const [lotId, setLotId] = useState(''); const [quantity, setQuantity] = useState('1');
  const [terms, setTerms] = useState(blankTerms); const [error, setError] = useState(''); const [busy, setBusy] = useState(false); const sending = useRef(false);
  const activePartners = (partners.data ?? []).filter(p => p.status === 'active');
  const visibleLots = (lots.data ?? []).filter(l => l.supplierCompanyId === supplier && BigInt(l.availableQuantity) > 0n);
  const vehicle = stock.data?.find(v => v.vehicleId === vehicleId); const lot = visibleLots.find(l => l.id === lotId);
  const vehicles = [...selectedVehicles, ...(vehicle && !selectedVehicles.some(v => v.vehicleId === vehicle.vehicleId) ? [vehicle] : [])];
  async function submit() {
    if (sending.current || !allowed(session, access) || session.company?.userType !== 'dealer') return;
    const desiredTerms = termPayload(terms, false);
    if (typeof desiredTerms === 'string') return setError(desiredTerms);
    if (partners.isLoading || partners.error || !activePartners.some(p => p.counterparty.id === supplier) ||
      (vehicles.length === 0 && (!lot || !quantityOK(quantity)))) return setError('Выберите активного поставщика и его VIN или будущую поставку с количеством.');
    const lines = vehicles.length ? vehicles.map(v => ({ vehicleId: v.vehicleId, quantity: '1' })) : [{ supplyLotId: lotId, quantity }];
    sending.current = true; setBusy(true); setError('');
    try {
      const result = await post<Reservation>('/commerce/reservations', { supplierCompanyId: supplier, lines, desiredTerms });
      await refresh(['reservations', session.company.id], ['supply-lots', session.company.id], ['reservation-stock', session.company.id]);
      onDone(result.data.id);
    } catch (e) { setError(errorText(e)); setBusy(false); sending.current = false; }
  }
  return <section><h3>Запрос резерва</h3><p>Запрос не удерживает автомобили или будущую поставку. Резерв возникнет после согласования условий.</p>
    {error && <Notice kind="error">{error}</Notice>}
    <Load loading={partners.isLoading} error={partners.error} retry={partners.refetch} empty={activePartners.length === 0 && !partners.isLoading} />
    <label>Поставщик<select aria-label="Поставщик" value={supplier} onChange={e => { setSupplier(e.target.value); setVehicleId(''); setSelectedVehicles([]); setLotId(''); setOffset(0); }}>
      <option value="">Выберите</option>{activePartners.map(p => <option key={p.counterparty.id} value={p.counterparty.id}>{p.counterparty.name}</option>)}</select></label>
    <Load loading={models.isLoading} error={models.error} retry={models.refetch} />
    <label>Модель для поиска VIN<select aria-label="Модель для поиска VIN" value={modelId} onChange={e => { setModelId(e.target.value); setVehicleId(''); setSelectedVehicles([]); setOffset(0); }}>
      <option value="">Все модели</option>{(models.data ?? []).map(m => <option key={m.id} value={m.id}>{modelName(m)}</option>)}</select></label>
    {supplier && <><h4>Физический остаток партнёра</h4>
      <Load loading={stock.isLoading} error={stock.error} retry={stock.refetch} empty={stock.data?.length === 0} />
      <label>VIN<select aria-label="VIN" value={vehicleId} disabled={stock.isLoading || !!stock.error} onChange={e => { setVehicleId(e.target.value); if (e.target.value) setLotId(''); }}>
        <option value="">Выберите VIN</option>{(stock.data ?? []).filter(v => !selectedVehicles.some(selected => selected.vehicleId === v.vehicleId)).map(v => <option key={v.vehicleId} value={v.vehicleId}>{v.vin} · {v.modelSpecificationVersion} · {v.exteriorColor}/{v.interiorColor}</option>)}</select></label>
      <Button disabled={!vehicle || selectedVehicles.length >= 99} onClick={() => { if (vehicle) { setSelectedVehicles([...selectedVehicles, vehicle]); setVehicleId(''); setLotId(''); } }}>Добавить VIN</Button>
      {selectedVehicles.length > 0 && <ul>{selectedVehicles.map(v => <li key={v.vehicleId}>{v.vin} <Button size="sm" onClick={() => setSelectedVehicles(selectedVehicles.filter(x => x.vehicleId !== v.vehicleId))}>Убрать {v.vin}</Button></li>)}</ul>}
      <div className="kit-row"><Button disabled={offset === 0} onClick={() => { setOffset(Math.max(0, offset - 50)); setVehicleId(''); }}>Предыдущие VIN</Button>
        <Button disabled={(stock.data?.length ?? 0) < 50} onClick={() => { setOffset(offset + 50); setVehicleId(''); }}>Следующие VIN</Button></div>
      <h4>Будущие поставки партнёра</h4><Load loading={lots.isLoading} error={lots.error} retry={lots.refetch} empty={visibleLots.length === 0 && !lots.isLoading} />
      <label>Будущая поставка<select aria-label="Будущая поставка" value={lotId} disabled={lots.isLoading || !!lots.error} onChange={e => { setLotId(e.target.value); if (e.target.value) { setVehicleId(''); setSelectedVehicles([]); } }}>
        <option value="">Выберите поставку</option>{visibleLots.map(l => <option key={l.id} value={l.id}>{modelName(models.data?.find(m => m.id === l.modelId))} · версия {l.modelSpecificationVersion} · {l.exteriorColor}/{l.interiorColor} · доступно {l.availableQuantity}</option>)}</select></label>
      {lotId && <label>Количество будущей поставки<input aria-label="Количество будущей поставки" inputMode="numeric" value={quantity} onChange={e => setQuantity(e.target.value)} /></label>}
    </>}
    <TermsFields value={terms} onChange={setTerms} desired />
    <Button variant="primary" disabled={busy || !allowed(session, access) || !!partners.error || !!stock.error || !!lots.error} onClick={() => void submit()}>{busy ? 'Отправка…' : 'Отправить запрос'}</Button>
  </section>;
}

function Snapshot({ title, snapshot, digest }: { title: string; snapshot?: Reservation['requested']; digest?: string }) {
  if (!snapshot) return null;
  const t = snapshot.terms;
  return <section><h3>{title}</h3><p>Отпечаток условий: <code>{digest || '—'}</code></p>
    <p>Цена за единицу: {money(t.unitPrice)} · Действует до: {t.expiresAt ? new Date(t.expiresAt).toLocaleString() : 'не указано'}</p>
    <p>Аванс: {money(t.advance)} · Срок аванса: {t.advanceDueAt ? new Date(t.advanceDueAt).toLocaleString() : 'не указан'}</p>
    <ul>{snapshot.lines.map((l: ReservationLine) => <li key={l.lineId}>{l.vehicleId ? `Автомобиль: ${l.vehicleId}` : `Будущая поставка: ${l.supplyLotId}`} · модель {l.modelId}, версия {l.modelSpecificationVersion}, кузов {l.exteriorColor}, салон {l.interiorColor}, количество {l.quantity}</li>)}</ul>
  </section>;
}

const reservationActionLabel: Record<string, string> = {
  confirm: 'подтвердить условия', accept: 'принять предложение', allocate: 'назначить VIN',
  extend: 'продлить', cancel: 'отменить', requestRefund: 'запросить возврат',
  proposeContinuation: 'предложить продолжение', acceptContinuation: 'принять продолжение', convert: 'создать заказ',
};

function AllocationSection({ reservation, busy, canWrite, onAllocate }: {
  reservation: Reservation; busy: boolean; canWrite: boolean; onAllocate: (lineId: string, vehicleId: string) => void;
}) {
  const session = useSession();
  const allocations = useReservationAllocations(reservation.id);
  const lots = useSupplyLots();
  const source = reservation.current?.lines.filter(line => !!line.supplyLotId) ?? [];
  const [lineId, setLineId] = useState('');
  const [vehicleId, setVehicleId] = useState('');
  const [offset, setOffset] = useState(0);
  const selected = source.find(line => line.lineId === lineId);
  const supplier = session.company?.id === reservation.supplier.id && session.company?.userType === 'distributor';
  const mayAllocate = supplier && canWrite && reservation.allowedActions.includes('allocate');
  const stock = useEligibleVehicles({ modelId: selected?.modelId ?? '', placement: 'warehouse', limit: 50, offset }, mayAllocate);
  const rows = allocations.data?.allocations ?? [];
  const current = rows.filter(row => row.current);
  const history = rows.filter(row => !row.current);
  const eligible = (stock.data ?? []).filter(v => !v.reserved && !!v.placement && v.modelId === selected?.modelId &&
    v.modelSpecificationVersion === selected?.modelSpecificationVersion &&
    v.exteriorColor === selected?.exteriorColor && v.interiorColor === selected?.interiorColor &&
    !current.some(row => row.vehicleId === v.id));
  return <section aria-label="VIN и будущая поставка"><h3>VIN и будущая поставка</h3>
    {source.map(line => {
      const lot = lots.data?.find(row => row.id === line.supplyLotId);
      return <p key={line.lineId}>Строка {line.lineId} · поставка {line.supplyLotId} · согласовано {line.quantity}
        {lot && <> · доступно {lot.availableQuantity}, удержано на будущее {lot.heldQuantity}, материализовано {lot.materializedQuantity} (счётчики поставки сервера)</>}
      </p>;
    })}
    {lots.error && <Notice kind="error">Не удалось обновить счётчики поставки. <Button onClick={() => void lots.refetch()}>Повторить</Button></Notice>}
    <Load loading={allocations.isLoading} error={allocations.error} retry={allocations.refetch} />
    {!allocations.error && <><h4>Текущие связи VIN</h4>{current.length ? <ul>{current.map(row => <li key={row.id}>Строка {row.lineId} · VIN {row.vehicleId} · держатель {row.holderType} {row.holderId} · источник {row.claimId}</li>)}</ul> : <p>Назначенных VIN пока нет.</p>}
      <h4>История замен и передач</h4>{history.length ? <ul>{history.map(row => <li key={row.id}>Строка {row.lineId} · VIN {row.vehicleId} · {new Date(row.createdAt).toLocaleString()} · источник {row.claimId}</li>)}</ul> : <p>Истории связей пока нет.</p>}</>}
    {mayAllocate && source.length > 0 && <div className="kit-stack"><h4>Назначить VIN из своего свободного склада</h4>
      <label>Согласованная строка<select aria-label="Согласованная строка" value={lineId} onChange={e => { setLineId(e.target.value); setVehicleId(''); setOffset(0); }}>
        <option value="">Выберите строку</option>{source.map(line => <option key={line.lineId} value={line.lineId}>{line.lineId} · {line.modelId} · {line.exteriorColor}/{line.interiorColor}</option>)}
      </select></label>
      {selected && <><Load loading={stock.isLoading} error={stock.error} retry={stock.refetch} empty={!stock.isLoading && !stock.error && eligible.length === 0} />
        <label>VIN<select aria-label="VIN для резерва" value={vehicleId} onChange={e => setVehicleId(e.target.value)} disabled={!!stock.error || stock.isLoading}>
          <option value="">Выберите VIN</option>{eligible.map(v => <option key={v.id} value={v.id}>{v.vin} · {v.exteriorColor}/{v.interiorColor}</option>)}
        </select></label>
        <div className="kit-row"><Button disabled={offset === 0 || busy} onClick={() => { setOffset(Math.max(0, offset - 50)); setVehicleId(''); }}>Назад по VIN</Button>
          <Button disabled={(stock.data?.length ?? 0) < 50 || busy} onClick={() => { setOffset(offset + 50); setVehicleId(''); }}>Далее по VIN</Button></div>
        <Button variant="primary" disabled={!vehicleId || busy || !!stock.error} onClick={() => onAllocate(lineId, vehicleId)}>Назначить VIN</Button></>}
    </div>}
  </section>;
}

function ContinuationSection({ reservation, busy, canWrite, onPropose, onAccept, onConvert }: {
  reservation: Reservation; busy: boolean; canWrite: boolean;
  onPropose: (value: ReservationContinuation) => void; onAccept: () => void; onConvert: () => void;
}) {
  const source = reservation.current!;
  const previous = reservation.continuationAccepted ?? reservation.continuationProposed;
  const currencySeed = previous?.terms.lines[0]?.unitPrice.currency ?? source.terms.unitPrice?.currency ?? 'USD';
  const [mode, setMode] = useState<'purchase' | 'consignment'>(previous?.mode ?? 'purchase');
  const [currency, setCurrency] = useState(currencySeed);
  const [prices, setPrices] = useState<Record<string, string>>(() => Object.fromEntries(source.lines.map(line => [line.lineId,
    amountInput(previous?.terms.lines.find(v => v.offerLineId === line.lineId)?.unitPrice ?? source.terms.unitPrice)])));
  const [route, setRoute] = useState(previous?.terms.route ?? '');
  const [delivery, setDelivery] = useState(previous?.terms.deliveryTerms ?? '');
  const [warranty, setWarranty] = useState(previous?.terms.warrantyTerms ?? '');
  const [service, setService] = useState(previous?.terms.serviceTerms ?? '');
  const [schedule, setSchedule] = useState<{ amount: string; dueDate: string }[]>(() => previous?.terms.paymentSchedule.map(v => ({ amount: amountInput(v.amount), dueDate: v.dueDate })) ?? []);
  const [partial, setPartial] = useState(false);
  const [minimum, setMinimum] = useState('');
  const [conditions, setConditions] = useState('');
  const [handover, setHandover] = useState(previous?.handoverTerms ?? '');
  const [error, setError] = useState('');
  function propose() {
    if (!route || !Object.hasOwn(routeLabel, route)) return setError('Выберите согласованный маршрут.');
    if (!/^[A-Z]{3}$/.test(currency)) return setError('Укажите трёхбуквенный код валюты.');
    if (!handover.trim()) return setError('Укажите условия передачи.');
    const lines: Terms['lines'] = [];
    for (const line of source.lines) {
      const amountMinor = toMinor(prices[line.lineId] ?? '');
      if (!amountMinor || BigInt(amountMinor) <= 0n) return setError(`Укажите положительную цену для строки ${line.lineId}.`);
      lines.push({ offerLineId: line.lineId, modelId: line.modelId, modelSpecificationVersion: line.modelSpecificationVersion,
        exteriorColor: line.exteriorColor, interiorColor: line.interiorColor, quantity: line.quantity,
        unitPrice: { amountMinor, currency } });
    }
    const payments: Terms['paymentSchedule'] = [];
    for (const item of schedule) {
      const amountMinor = toMinor(item.amount);
      if (!amountMinor || BigInt(amountMinor) <= 0n || !/^\d{4}-\d{2}-\d{2}$/.test(item.dueDate)) return setError('Заполните сумму и дату каждого согласованного платежа.');
      payments.push({ amount: { amountMinor, currency }, dueDate: item.dueDate });
    }
    const total = lines.reduce((sum, line) => sum + BigInt(line.unitPrice.amountMinor) * BigInt(line.quantity), 0n);
    if (payments.length && payments.reduce((sum, item) => sum + BigInt(item.amount.amountMinor), 0n) !== total)
      return setError('Сумма согласованного графика должна равняться полной цене заказа.');
    const minimumMinor = minimum ? toMinor(minimum) : null;
    if (partial && minimum && (minimumMinor === null || BigInt(minimumMinor) > total)) return setError('Согласованный минимум должен быть точной суммой в пределах цены заказа.');
    setError('');
    onPropose({ mode, terms: { lines, route, deliveryTerms: delivery, warrantyTerms: warranty, serviceTerms: service, paymentSchedule: payments },
      allowPartialPurchaseHandover: mode === 'purchase' && partial,
      ...(mode === 'purchase' && partial && minimumMinor !== null ? { minimumPaid: { amountMinor: minimumMinor, currency } } : {}),
      ...(mode === 'purchase' && partial && conditions.trim() ? { handoverConditions: conditions.trim() } : {}), handoverTerms: handover.trim() });
  }
  const proposed = reservation.continuationProposed;
  const accepted = reservation.continuationAccepted;
  return <section aria-label="Продолжение резерва"><h3>Продолжение в заказ</h3>
    {proposed && <section><h4>Предложено {reservation.continuationProposedBy}</h4><p>Отпечаток: <code>{reservation.continuationProposedDigest}</code> · ревизия {reservation.continuationProposedRevision}</p>
      <p>Форма: {proposed.mode === 'purchase' ? 'покупка' : 'консигнация'} · передача до полной оплаты: {proposed.allowPartialPurchaseHandover ? 'согласована' : 'не согласована'}</p>
      {proposed.minimumPaid && <p>Согласованный минимум: {money(proposed.minimumPaid)}</p>}
      {proposed.handoverConditions && <p>Условия частичной передачи: {proposed.handoverConditions}</p>}
      <p>Условия передачи: {proposed.handoverTerms}</p><TermsView terms={proposed.terms} modelNameOf={id => id} />
    </section>}
    {accepted && <Notice kind="success">Обе стороны приняли точные условия <code>{reservation.continuationAcceptedDigest}</code> · ревизия {reservation.continuationAcceptedRevision}. Зачёт принятого аванса показан в учёте и не прибавляется к оплате повторно.</Notice>}
    {canWrite && reservation.allowedActions.includes('acceptContinuation') && proposed && !accepted && <Button variant="primary" disabled={busy || !reservation.continuationProposedDigest || !reservation.continuationProposedRevision} onClick={onAccept}>Принять точное предложение</Button>}
    {canWrite && reservation.allowedActions.includes('convert') && accepted && <Button variant="primary" disabled={busy || !reservation.continuationAcceptedDigest || !reservation.continuationAcceptedRevision} onClick={onConvert}>Создать согласованный заказ</Button>}
    {canWrite && reservation.allowedActions.includes('proposeContinuation') && <div className="kit-stack"><h4>Предложить новые условия продолжения</h4>
      <label>Форма<select aria-label="Форма продолжения" value={mode} onChange={e => { const next = e.target.value as 'purchase' | 'consignment'; setMode(next); if (next === 'consignment') setPartial(false); }}><option value="purchase">Покупка</option><option value="consignment">Консигнация</option></select></label>
      <label>Валюта<input aria-label="Валюта продолжения" value={currency} onChange={e => setCurrency(e.target.value.toUpperCase())} /></label>
      {source.lines.map(line => <label key={line.lineId}>Цена за единицу · {line.lineId}<input aria-label={`Цена строки ${line.lineId}`} value={prices[line.lineId] ?? ''} inputMode="decimal" onChange={e => setPrices({ ...prices, [line.lineId]: e.target.value })} /></label>)}
      <label>Маршрут<select aria-label="Маршрут продолжения" value={route} onChange={e => setRoute(e.target.value)}><option value="">Выберите</option>{Object.entries(routeLabel).map(([value, label]) => <option key={value} value={value}>{label}</option>)}</select></label>
      <label>Условия поставки<textarea aria-label="Условия поставки продолжения" value={delivery} onChange={e => setDelivery(e.target.value)} /></label>
      <label>Гарантия<textarea aria-label="Гарантия продолжения" value={warranty} onChange={e => setWarranty(e.target.value)} /></label>
      <label>Сервис<textarea aria-label="Сервис продолжения" value={service} onChange={e => setService(e.target.value)} /></label>
      <p>График оплаты согласуется явно. Пустой график означает, что платежи по датам не согласованы.</p>
      <PaymentScheduleEditor schedule={schedule} setSchedule={setSchedule} />
      {mode === 'purchase' && <><label><input type="checkbox" checked={partial} onChange={e => setPartial(e.target.checked)} /> Стороны явно разрешают передачу автомобиля до полной оплаты</label>
        {partial && <><label>Согласованный минимум оплаты (необязательно)<input aria-label="Согласованный минимум оплаты" inputMode="decimal" value={minimum} onChange={e => setMinimum(e.target.value)} /></label>
          <label>Условия передачи до полной оплаты<textarea aria-label="Условия частичной передачи" value={conditions} onChange={e => setConditions(e.target.value)} /></label></>}</>}
      <label>Согласованные условия передачи<textarea aria-label="Согласованные условия передачи" value={handover} onChange={e => setHandover(e.target.value)} /></label>
      {error && <Notice kind="error">{error}</Notice>}
      <Button variant="primary" disabled={busy} onClick={propose}>Отправить предложение</Button>
    </div>}
  </section>;
}

function ReservationDetail({ id, onClose, onOpenOrder }: { id: string; onClose: () => void; onOpenOrder?: (id: string) => void }) {
  const session = useSession(); const access = useCommercialAccess(); const refresh = useRefresh(); const detail = useReservationDetail(id);
  const [error, setError] = useState(''); const [busy, setBusy] = useState(false); const [reconcile, setReconcile] = useState(false); const sending = useRef(false);
  const [terms, setTerms] = useState<TermDraft | null>(null); const [quantities, setQuantities] = useState<Record<string, string>>({});
  const [extensionExpiry, setExtensionExpiry] = useState(''); const [extensionAdvance, setExtensionAdvance] = useState('');
  const [extensionReason, setExtensionReason] = useState(''); const [cancelReason, setCancelReason] = useState('');
  const current = detail.data;
  const companyId = session.company?.id ?? '';
  const canWrite = !!session.company && session.can('commerce.trade') && !reconcile && !detail.error;
  const canStart = allowed(session, access) && !reconcile && !detail.error;
  async function reload() {
    await refresh(['reservations', companyId], ['supply-lots', companyId], ['eligible-vehicles'],
      ['orders'], ['order'], ['order-invoices'], ['billing', 'reservations', companyId]);
    const result = await detail.refetch();
    if (result.error) throw result.error;
  }
  async function retry() {
    try { await reload(); setReconcile(false); setError(''); }
    catch (e) { setError(errorText(e)); }
  }
  async function mutate(path: string, body: unknown, orderResult = false) {
    if (sending.current || !current || !canWrite) return;
    sending.current = true; setBusy(true); setError('');
    try {
      const result = await post<{ orderId: string }>(`/commerce/reservations/${encodeURIComponent(id)}/${path}`, body, { ifMatch: current.revision });
      setReconcile(true);
      try { await reload(); setReconcile(false); }
      catch (e) { setError(`Действие отправлено. Обновите данные перед следующим действием: ${errorText(e)}`); }
      setTerms(null); setQuantities({});
      if (orderResult && result.data.orderId && onOpenOrder) onOpenOrder(result.data.orderId);
    } catch (e) {
      setError(errorText(e)); setReconcile(true);
      try { await reload(); setReconcile(false); }
      catch { /* The explicit retry keeps writes locked until an authoritative read succeeds. */ }
    }
    finally { sending.current = false; setBusy(false); }
  }
  function confirm() {
    if (!current || !canStart || current.supplier.id !== session.company?.id || session.company.userType !== 'distributor' || !current.allowedActions.includes('confirm')) return;
    const offered = termPayload(terms ?? fromTerms(current.requested.terms), true);
    if (typeof offered === 'string') return setError(offered);
    const lines = current.requested.lines.map(l => ({ lineId: l.lineId, quantity: quantities[l.lineId] ?? l.quantity }));
    if (lines.some(l => !quantityOK(l.quantity))) return setError('Количество в каждой строке должно быть положительным целым.');
    void mutate('supplier-confirmations', { terms: offered, lines });
  }
  function accept() {
    if (!current || !canStart || current.buyer.id !== session.company?.id || session.company.userType !== 'dealer' || !current.allowedActions.includes('accept') || !current.offeredDigest) return;
    void mutate('acceptances', { digest: current.offeredDigest });
  }
  function extend() {
    if (!current?.allowedActions.includes('extend') || current.supplier.id !== companyId) return;
    const expiresAt = iso(extensionExpiry);
    if (!expiresAt || !extensionReason.trim()) return setError('Укажите новый срок и причину продления.');
    void mutate('extensions', { expiresAt, ...(extensionAdvance ? { advanceDueAt: iso(extensionAdvance) } : {}), reason: extensionReason.trim() });
  }
  function cancel() {
    if (!current?.allowedActions.includes('cancel') || !cancelReason.trim()) return setError('Укажите причину отмены.');
    void mutate('cancellations', { reason: cancelReason.trim() });
  }
  const actionBusy = busy || reconcile;
  return <Modal title="Условия резерва" size="wide" onClose={onClose} footer={<Button onClick={onClose}>Закрыть</Button>}>
    <Load loading={detail.isLoading} error={detail.error} retry={detail.refetch} />
    {current && <><p>{current.buyer.name} → {current.supplier.name} · статус: {current.status} · ревизия: {current.revision}</p>
      <p>Доступные действия по данным сервера: {current.allowedActions.length ? current.allowedActions.map(action => reservationActionLabel[action] ?? action).join(', ') : 'нет'}.</p>
      {!access.allowed && <Notice kind="warning">{access.reason}</Notice>}
      {error && <Notice kind="error">{error} <Button onClick={() => void retry()}>Обновить данные</Button></Notice>}
      {current.expiresAt && <p>Текущий согласованный срок: {new Date(current.expiresAt).toLocaleString()}{current.advanceDueAt ? ` · срок обязательного аванса: ${new Date(current.advanceDueAt).toLocaleString()}` : ''}</p>}
      {current.extendedAt && <p>Продлено {new Date(current.extendedAt).toLocaleString()} · участник {current.extendedBy}</p>}
      {current.releasedAt && <Notice>Резерв освобождён {new Date(current.releasedAt).toLocaleString()}. Причина: {current.releaseReason || 'не указана сервером'}.</Notice>}
      {current.cancellationState === 'awaitingEvidenceDecision' && <Notice kind="warning">До отмены бухгалтер должен решить вопрос с отправленными документами оплаты.</Notice>}
      {current.cancellationState === 'awaitingRefund' && <Notice kind="warning">До отмены нужен полный подтверждённый бухгалтером возврат аванса. Запрос возврата сам по себе не освобождает резерв.</Notice>}
      <Snapshot title="Запрошено" snapshot={current.requested} digest={current.requestedDigest} />
      <Snapshot title="Предложено поставщиком" snapshot={current.offered} digest={current.offeredDigest} />
      <Snapshot title="Согласовано" snapshot={current.current} digest={current.currentDigest} />
      {current.current && <ReservationPayment reservation={current} />}
      {current.current && <AllocationSection reservation={current} busy={actionBusy} canWrite={canWrite && access.allowed}
        onAllocate={(lineId, vehicleId) => void mutate('allocations', { items: [{ lineId, vehicleId }] })} />}
      {current.orderId && <section><h3>Созданный заказ</h3><p>Резерв продолжен заказом {current.orderId}. История резерва остаётся доступной.</p>
        {onOpenOrder && <Button onClick={() => onOpenOrder(current.orderId!)}>Открыть заказ</Button>}</section>}
      {current.current && !current.orderId && <ContinuationSection key={`${current.id}:${current.continuationProposedDigest ?? ''}`} reservation={current} busy={actionBusy} canWrite={canWrite}
        onPropose={proposal => void mutate('continuation-proposals', proposal)}
        onAccept={() => void mutate('continuation-acceptances', { digest: current.continuationProposedDigest, proposedRevision: current.continuationProposedRevision })}
        onConvert={() => void mutate('conversions', { digest: current.continuationAcceptedDigest, acceptedRevision: current.continuationAcceptedRevision }, true)} />}
      {current.allowedActions.includes('extend') && current.supplier.id === companyId && session.company?.userType === 'distributor' && <section><h3>Продлить резерв</h3>
        <p>Продление срока аванса указывается отдельно. Неизменённый срок аванса останется прежним.</p>
        <label>Новый срок резерва<input type="datetime-local" aria-label="Новый срок резерва" value={extensionExpiry} onChange={e => setExtensionExpiry(e.target.value)} /></label>
        <label>Новый срок аванса (необязательно)<input type="datetime-local" aria-label="Новый срок аванса" value={extensionAdvance} onChange={e => setExtensionAdvance(e.target.value)} /></label>
        <label>Причина продления<textarea aria-label="Причина продления" value={extensionReason} onChange={e => setExtensionReason(e.target.value)} /></label>
        <Button disabled={actionBusy || !canWrite} onClick={extend}>Продлить</Button>
      </section>}
      {current.allowedActions.includes('cancel') && <section><h3>Отменить резерв</h3>
        <label>Причина отмены<textarea aria-label="Причина отмены резерва" value={cancelReason} onChange={e => setCancelReason(e.target.value)} /></label>
        <Button disabled={actionBusy || !canWrite} onClick={cancel}>Отменить резерв</Button>
      </section>}
      {current.allowedActions.includes('confirm') && current.supplier.id === session.company?.id && session.company.userType === 'distributor' && <section>
        <h3>Подтвердить или предложить условия</h3>
        <TermsFields value={terms ?? fromTerms(current.requested.terms)} onChange={setTerms} />
        {current.requested.lines.map(l => <label key={l.lineId}>Количество · {l.modelId} / {l.modelSpecificationVersion}
          <input aria-label={`Количество ${l.lineId}`} inputMode="numeric" disabled={!!l.vehicleId} value={quantities[l.lineId] ?? l.quantity} onChange={e => setQuantities({ ...quantities, [l.lineId]: e.target.value })} /></label>)}
        <Button variant="primary" disabled={actionBusy || !canStart} onClick={confirm}>{busy ? 'Сохранение…' : 'Отправить подтверждение'}</Button>
      </section>}
      {current.allowedActions.includes('accept') && current.buyer.id === session.company?.id && session.company.userType === 'dealer' && current.offered && <section>
        <p>Примите точный отпечаток предложения: <code>{current.offeredDigest}</code></p>
        <Button variant="primary" disabled={actionBusy || !canStart} onClick={accept}>{busy ? 'Принятие…' : 'Принять предложенные условия'}</Button>
      </section>}
    </>}
  </Modal>;
}
