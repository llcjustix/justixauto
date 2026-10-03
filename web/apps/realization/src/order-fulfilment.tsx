import { useEffect, useRef, useState } from 'react';
import { ApiError, Autocomplete, Button, Notice, list, post, useRefresh } from '@justixauto/kit';
import { modelName, routeLabel, useEligibleVehicles, useOrderModelDetails, useOrderModels, useWarehouses } from './data';
import type { Order, Vehicle } from './data';
import { validateVins, VinEditor } from './vin-editor';
import { colorOptions, exactSpecification, selectedColor, vehicleColorsLabel } from './vehicle-colors';

export type OrderFulfilmentAction = { kind: 'allocate'; lineId: string } | { kind: 'ship-quantity' } | { kind: 'ship-allocated' };
export interface OrderFulfilmentPanelProps {
  order: Order;
  action: OrderFulfilmentAction;
  refresh: unknown[][];
  onDone: (message: string) => void;
  onBack: () => void;
}

export function remainingQuantity(order: Order, lineId: string) {
  const line = order.terms.lines.find((item) => item.lineId === lineId);
  const progress = order.lineProgress?.find((item) => item.orderLineId === lineId);
  const allocated = progress?.allocated ?? String(order.allocations.filter((item) => item.orderLineId === lineId && item.status === 'allocated').length);
  return Math.max(0, Number(line?.quantity ?? 0) - Number(progress?.shipped ?? 0) - Number(allocated));
}

/** Action/line switches intentionally start a new draft; order refreshes preserve the current draft. */
export function OrderFulfilmentPanel(props: OrderFulfilmentPanelProps) {
  return <FulfilmentDraft key={`${props.order.id}:${props.action.kind}:${props.action.kind === 'allocate' ? props.action.lineId : ''}`} {...props} />;
}

function FulfilmentDraft({ order, action, refresh, onDone, onBack }: OrderFulfilmentPanelProps) {
  const refreshQueries = useRefresh();
  const models = useOrderModels();
  const warehouses = useWarehouses();
  const name = (id: string) => modelName(models.data?.find((model) => model.id === id)) === '—' ? id : modelName(models.data?.find((model) => model.id === id));
  const line = action.kind === 'allocate' ? order.terms.lines.find((item) => item.lineId === action.lineId) : undefined;
  const [search, setSearch] = useState('');
  const [warehouse, setWarehouse] = useState('');
  const [offset, setOffset] = useState(0);
  const [selected, setSelected] = useState<Vehicle[]>([]);
  const [allocatedIds, setAllocatedIds] = useState<string[]>([]);
  const [quantities, setQuantities] = useState<Record<string, string>>(() => Object.fromEntries(order.terms.lines.map((item) => [item.lineId!, String(remainingQuantity(order, item.lineId!))])));
  const [vins, setVins] = useState<Record<string, string[]>>({});
  const [incoming, setIncoming] = useState<Record<string, { modelSpecificationVersion?: string; exteriorColor?: string; interiorColor?: string; exteriorEdited?: boolean; interiorEdited?: boolean }>>({});
  const historical = useOrderModelDetails(action.kind === 'ship-quantity' ? order.terms.lines.filter(item => !item.modelSpecificationVersion || !item.exteriorColor || !item.interiorColor).map(item => item.modelId) : []);
  const [route, setRoute] = useState(order.terms.route);
  const [error, setError] = useState('');
  const [pending, setPending] = useState(false);
  const [filling, setFilling] = useState(false);
  const [stockMessage, setStockMessage] = useState('');
  const sending = useRef(false);
  const errorRef = useRef<HTMLDivElement>(null);
  const stock = useEligibleVehicles({ modelId: line?.modelId ?? '', placement: 'warehouse', warehouseId: warehouse, search: search.trim().toUpperCase(), limit: 50, offset }, action.kind === 'allocate');
  const remaining = line ? remainingQuantity(order, line.lineId!) : 0;
  const cap = Math.min(remaining, 1000);
  const allowed = order.allowedActions.includes(action.kind === 'ship-allocated' ? 'ship' : action.kind);
  const allocated = order.allocations.filter((item) => item.status === 'allocated');
  // Defense in depth: only the server's owned/eligible endpoint is used, and rows must still match this exact line/filter.
  const matchesLine = (vehicle: Vehicle) => vehicle.modelId === line?.modelId && (!line?.exteriorColor || vehicle.exteriorColor?.toLowerCase() === line.exteriorColor.toLowerCase()) && (!line?.interiorColor || vehicle.interiorColor?.toLowerCase() === line.interiorColor.toLowerCase());
  const eligible = (vehicle: Vehicle) => matchesLine(vehicle) && !vehicle.reserved && !!vehicle.placement;
  const visibleStock = (stock.data ?? []).filter((vehicle) => eligible(vehicle) && (!warehouse || vehicle.placement?.warehouseId === warehouse) && vehicle.vin.toUpperCase().includes(search.trim().toUpperCase()));
  // Refreshes revalidate visible selections; off-page selections still have to match current ordered facts.
  useEffect(() => {
    setSelected(previous => {
      const next = previous.map(vehicle => stock.data?.find(fresh => fresh.id === vehicle.id) ?? vehicle).filter(eligible);
      return next.length === previous.length && next.every((vehicle, index) => vehicle === previous[index]) ? previous : next;
    });
  }, [selected, stock.data, line?.modelId, line?.exteriorColor, line?.interiorColor]);
  const quantityLines = order.terms.lines.map((item) => {
    const draft = incoming[item.lineId!] ?? {};
    const needsFacts = !item.modelSpecificationVersion || !item.exteriorColor || !item.interiorColor;
    const model = historical.data?.find(model => model.id === item.modelId);
    const version = item.modelSpecificationVersion || draft.modelSpecificationVersion || '';
    const spec = exactSpecification(model, version);
    const exteriorOptions = colorOptions(spec, 'exterior', item.exteriorColor);
    const interiorOptions = colorOptions(spec, 'interior', item.interiorColor);
    const exterior = item.exteriorColor || (draft.exteriorEdited ? exteriorOptions.find(value => value === draft.exteriorColor) ?? '' : selectedColor(exteriorOptions, draft.exteriorColor));
    const interior = item.interiorColor || (draft.interiorEdited ? interiorOptions.find(value => value === draft.interiorColor) ?? '' : selectedColor(interiorOptions, draft.interiorColor));
    const ready = !needsFacts || (!!spec && !historical.isFetching && !historical.isError && exteriorOptions.some(value => value.toLowerCase() === exterior.toLowerCase()) && interiorOptions.some(value => value.toLowerCase() === interior.toLowerCase()));
    return { item, quantity: quantities[item.lineId!] ?? '', checked: validateVins(vins[item.lineId!] ?? [], Number(quantities[item.lineId!]) || 0), needsFacts, model, version, spec, exteriorOptions, interiorOptions, exterior, interior, ready };
  });
  const total = quantityLines.reduce((sum, entry) => sum + (/^\d+$/.test(entry.quantity) ? Number(entry.quantity) : 0), 0);
  const identified = quantityLines.reduce((sum, entry) => sum + entry.checked.vins.length, 0);
  const title = action.kind === 'allocate' ? 'Назначить VIN со склада' : action.kind === 'ship-quantity' ? 'Отгрузить по количеству' : 'Отгрузить назначенные VIN';
  const fail = (message: string) => { setError(message); requestAnimationFrame(() => errorRef.current?.focus()); };
  const refreshAll = () => {
    const keys = [...refresh, ['eligible-vehicles'], ['vehicles'], ['warehouses'], ['stock']];
    return refreshQueries(...keys.filter((key, index) => keys.findIndex((other) => JSON.stringify(other) === JSON.stringify(key)) === index));
  };
  const pageSelection = visibleStock.filter((vehicle) => !selected.some((item) => item.id === vehicle.id)).slice(0, Math.max(0, cap - selected.length));

  async function fillSelection() {
    if (!line || filling || pending) return;
    setFilling(true);
    setStockMessage('');
    const found = new Map(selected.filter(eligible).map((vehicle) => [vehicle.id, vehicle]));
    try {
      for (let next = 0; found.size < cap; next += 100) {
        const params = new URLSearchParams({ modelId: line.modelId, placement: 'warehouse', eligible: 'true', limit: '100', offset: String(next) });
        if (warehouse) params.set('warehouseId', warehouse);
        if (search.trim()) params.set('search', search.trim().toUpperCase());
        const page = await list<Vehicle>(`/inventory/vehicle-units?${params}`);
        for (const vehicle of page) {
          if (found.size >= cap) break;
          if (eligible(vehicle) && (!warehouse || vehicle.placement?.warehouseId === warehouse) && vehicle.vin.toUpperCase().includes(search.trim().toUpperCase())) found.set(vehicle.id, vehicle);
        }
        if (page.length < 100) break;
      }
      setSelected([...found.values()]);
      setStockMessage(`Выбрано ${found.size} из ${cap}. Поиск выполнен по текущему складу и фильтру VIN; остальные результаты не включены.`);
    } catch {
      setSelected([...found.values()]);
      fail('Не удалось завершить подбор. Уже выбранные автомобили сохранены; повторите загрузку или исправьте выбор.');
    } finally { setFilling(false); }
  }

  async function submit(event: React.FormEvent) {
    event.preventDefault();
    if (sending.current || filling) return;
    if (!allowed) return fail('Действие больше недоступно. Обновите заказ и проверьте его состояние.');
    let payload: unknown;
    let summary: string;
    if (action.kind === 'allocate') {
      if (!line) return fail('Строка заказа больше недоступна');
      if (selected.length === 0) return fail('Выберите хотя бы один автомобиль');
      if (selected.length > cap) return fail(`Можно назначить не более ${cap} автомобилей за один запрос`);
      if (selected.some((vehicle) => !eligible(stock.data?.find(fresh => fresh.id === vehicle.id) ?? vehicle))) return fail('Модель или цвета автомобиля не соответствуют строке заказа либо VIN уже занят');
      payload = { items: selected.map((vehicle) => ({ orderLineId: line.lineId, vehicleId: vehicle.id })) };
      summary = `Назначено VIN: ${selected.length}.`;
    } else if (action.kind === 'ship-quantity') {
      if (!quantityLines.some((entry) => Number(entry.quantity) > 0)) return fail('Укажите количество хотя бы по одной строке');
      for (const entry of quantityLines) {
        if (!/^\d+$/.test(entry.quantity) || !Number.isSafeInteger(Number(entry.quantity)) || Number(entry.quantity) > remainingQuantity(order, entry.item.lineId!)) return fail(`${name(entry.item.modelId)}: введите целое количество от 0 до ${remainingQuantity(order, entry.item.lineId!)}`);
        if (!entry.checked.valid) return fail(`${name(entry.item.modelId)}: ${entry.checked.errors.join('. ')}`);
      }
      const allVins = quantityLines.flatMap((entry) => entry.checked.vins);
      if (new Set(allVins).size !== allVins.length) return fail('Повторяющийся VIN в разных строках заказа');
      if (quantityLines.some(entry => Number(entry.quantity) > 0 && !entry.ready)) return fail('Укажите точную версию спецификации и цвета для каждой отгружаемой строки');
      payload = { lines: quantityLines.filter((entry) => Number(entry.quantity) > 0).map((entry) => ({ orderLineId: entry.item.lineId, quantity: String(Number(entry.quantity)), vins: entry.checked.vins,
        ...(!entry.item.modelSpecificationVersion ? { modelSpecificationVersion: entry.version } : {}),
        ...(!entry.item.exteriorColor ? { exteriorColor: entry.exterior } : {}),
        ...(!entry.item.interiorColor ? { interiorColor: entry.interior } : {}),
      })), route };
      summary = `Отгружено: ${total}. С VIN: ${identified}. Без VIN: ${total - identified}.`;
    } else {
      if (!allocatedIds.length) return fail('Выберите хотя бы один автомобиль');
      if (allocatedIds.some((id) => !allocated.some((item) => item.vehicleId === id))) return fail('Некоторые VIN больше не доступны для отгрузки. Проверьте выбор.');
      payload = { vehicleIds: allocatedIds, route };
      summary = `Отгружено: ${allocatedIds.length}. С VIN: ${allocatedIds.length}. Без VIN: 0.`;
    }
    if (action.kind !== 'allocate' && !Object.hasOwn(routeLabel, route)) return fail('Выберите маршрут');
    sending.current = true;
    setPending(true);
    setError('');
    try {
      await post(`/commerce/orders/${order.id}/${action.kind === 'allocate' ? 'allocations' : 'shipments'}`, payload, { ifMatch: order.revision });
    } catch (caught) {
      const message = caught instanceof Error ? caught.message : 'Не удалось выполнить операцию';
      const fields = caught instanceof ApiError ? Object.entries(caught.fields).map(([key, value]) => `${key}: ${value}`).join('; ') : '';
      fail([message, fields, caught instanceof ApiError && caught.status === 412 ? 'Заказ изменён. Данные обновлены; проверьте сохранённый ввод перед повторной отправкой.' : ''].filter(Boolean).join(' · '));
      if (caught instanceof ApiError && [409, 412].includes(caught.status)) {
        try { await refreshAll(); } catch { setError((previous) => `${previous} Обновление не удалось; повторите загрузку.`); }
      }
      sending.current = false;
      setPending(false);
      return;
    }
    // A refresh failure must never turn a successful command into a second submission.
    try { await refreshAll(); } catch { summary += ' Не удалось обновить списки; обновите заказ.'; }
    onDone(summary);
  }

  return (
    <form className="order-fulfilment" onSubmit={submit} noValidate>
      <h2>{title}</h2>
      {error && <div ref={errorRef} tabIndex={-1}><Notice kind="error">{error}</Notice></div>}
      {!allowed && <Notice kind="warning">Действие больше недоступно для этого заказа.</Notice>}
      {models.isError && <p>Не удалось загрузить названия моделей. <Button onClick={() => void models.refetch()}>Повторить загрузку моделей</Button></p>}
      <fieldset disabled={pending || filling}>
        {action.kind === 'allocate' && <>
          <p>Назначаются свободные автомобили с VIN со склада. Если список пуст, откройте «Склады», примите автомобили с VIN или введите VIN в ранее принятой партии.</p>
          <h3>{line ? name(line.modelId) : 'Строка не найдена'}</h3>
          {line && <p>{vehicleColorsLabel(line)}</p>}
          <p>Заказано: {line?.quantity ?? 0} · Отгружено: {order.lineProgress?.find((item) => item.orderLineId === line?.lineId)?.shipped ?? 0} · Назначено: {order.lineProgress?.find((item) => item.orderLineId === line?.lineId)?.allocated ?? 0} · Осталось: {remaining}</p>
          <label>Поиск VIN<input value={search} onChange={(event) => { setSearch(event.target.value); setOffset(0); }} /></label>
          <label>Склад<Autocomplete aria-label="Склад" value={warehouse} onChange={(value) => { setWarehouse(value); setOffset(0); }}
            options={[{ value: '', label: 'Все склады' }, ...(warehouses.data ?? []).map(item => ({ value: item.id, label: item.name }))]} /></label>
          {warehouses.isLoading && <p>Загрузка складов…</p>}
          {warehouses.isError && <p>Не удалось загрузить склады. <Button onClick={() => void warehouses.refetch()}>Повторить загрузку складов</Button></p>}
          {stock.isLoading && <p role="status">Загрузка автомобилей…</p>}
          {stock.isError && <Notice kind="error">Не удалось загрузить автомобили. Повторите загрузку.</Notice>}
          <Button disabled={stock.isFetching} onClick={() => void stock.refetch()}>Обновить автомобили</Button>
          {!stock.isLoading && !stock.isError && visibleStock.length === 0 && <p>Нет подходящих автомобилей на этой странице.</p>}
          <table><thead><tr><th>Выбрать</th><th>VIN</th><th>Модель</th><th>Склад</th></tr></thead><tbody>
            {visibleStock.map((vehicle) => {
              const chosen = selected.some((item) => item.id === vehicle.id);
              return <tr key={vehicle.id}><td><input type="checkbox" aria-label={`Выбрать ${vehicle.vin}`} checked={chosen}
                disabled={stock.isFetching || (!chosen && selected.length >= cap)}
                onChange={() => setSelected((previous) => chosen ? previous.filter((item) => item.id !== vehicle.id) : [...previous, vehicle])} /></td>
                <td>{vehicle.vin}<br />{vehicleColorsLabel(vehicle)}</td><td>{name(vehicle.modelId)}</td><td>{warehouses.data?.find((item) => item.id === vehicle.placement?.warehouseId)?.name ?? vehicle.placement?.warehouseId}</td></tr>;
            })}
          </tbody></table>
          <div className="order-fulfilment-pages"><Button disabled={offset === 0 || stock.isFetching} onClick={() => setOffset(offset - 50)}>Предыдущая страница</Button>
            <span>Страница {offset / 50 + 1}</span><Button disabled={stock.isFetching || stock.isError || (stock.data?.length ?? 0) < 50} onClick={() => setOffset(offset + 50)}>Следующая страница</Button></div>
          <p>Выбрано: {selected.length} из {cap}. Осталось по строке: {remaining}.</p>
          <div className="order-fulfilment-pages">
            <Button disabled={!pageSelection.length || stock.isFetching} onClick={() => setSelected((previous) => [...previous, ...pageSelection])}>Выбрать на странице: {pageSelection.length}</Button>
            <Button disabled={selected.length >= cap} onClick={() => void fillSelection()}>Заполнить лимит из результатов: до {cap}</Button>
          </div>
          {stockMessage && <p role="status">{stockMessage}</p>}
          {remaining > 1000 && <p>За один запрос можно назначить до 1000 VIN. Оставшиеся назначьте следующей партией.</p>}
          {selected.length > cap && <Notice kind="error">Выбор превышает текущий лимит {cap}. Удалите лишние автомобили.</Notice>}
          <details open><summary>Выбранные VIN ({selected.length})</summary><ul>{selected.map((vehicle) => <li key={vehicle.id}>{vehicle.vin} <Button onClick={() => setSelected((previous) => previous.filter((item) => item.id !== vehicle.id))}>Убрать {vehicle.vin}</Button></li>)}</ul></details>
        </>}
        {action.kind === 'ship-quantity' && <>
          <p>Автомобили сразу поступят на склад покупателя. VIN можно указать частично или ввести позже. Вместимость проверяется при отгрузке.</p>
          <Button onClick={() => setQuantities(Object.fromEntries(order.terms.lines.map((item) => [item.lineId!, String(remainingQuantity(order, item.lineId!))])))}>Отгрузить весь остаток</Button>
          {quantityLines.map(({ item, quantity, checked, needsFacts, model, version, spec, exteriorOptions, interiorOptions, exterior, interior }) => {
            const left = remainingQuantity(order, item.lineId!);
            const changeIncoming = (values: typeof incoming[string]) => setIncoming(previous => ({ ...previous, [item.lineId!]: { ...previous[item.lineId!], ...values } }));
            return <section key={item.lineId} className="order-fulfilment-line"><h3>{name(item.modelId)}</h3>
              <p>Заказано: {vehicleColorsLabel(item)}</p>
              {needsFacts && <fieldset disabled={historical.isFetching}>
                <legend>Входящая партия — уточнение прежнего заказа</legend>
                <p>Указанные здесь сведения относятся к поступлению. Исторические условия заказа сохраняются.</p>
                {historical.isFetching && <p role="status">Загрузка спецификаций…</p>}
                {(historical.isError || (!historical.isFetching && (!model || (!!version && !spec)))) && <Notice kind="error">Точная спецификация недоступна. <Button onClick={() => void historical.refetch()}>Повторить загрузку спецификаций</Button></Notice>}
                {!item.modelSpecificationVersion && <label>Версия входящей спецификации<Autocomplete aria-label="Версия входящей спецификации" value={version} onChange={value => changeIncoming({ modelSpecificationVersion: value, exteriorColor: '', interiorColor: '', exteriorEdited: false, interiorEdited: false })} options={[{ value: '', label: 'Выберите версию' }, ...(model?.versions ?? []).map(spec => ({ value: spec.version, label: `Версия ${spec.version}` }))]} /></label>}
                {!item.exteriorColor && <label>Цвет кузова<Autocomplete aria-label="Цвет кузова" value={exterior} onChange={value => changeIncoming({ exteriorColor: value, exteriorEdited: true })} options={[{ value: '', label: 'Выберите цвет' }, ...exteriorOptions.map(value => ({ value, label: value }))]} /></label>}
                {!item.interiorColor && <label>Цвет салона<Autocomplete aria-label="Цвет салона" value={interior} onChange={value => changeIncoming({ interiorColor: value, interiorEdited: true })} options={[{ value: '', label: 'Выберите цвет' }, ...interiorOptions.map(value => ({ value, label: value }))]} /></label>}
                <p>Поступление: {vehicleColorsLabel({ exteriorColor: exterior, interiorColor: interior, modelSpecificationVersion: version })}</p>
              </fieldset>}
              <label>{name(item.modelId)} · {vehicleColorsLabel(item)} — количество (осталось {left})<input type="text" inputMode="numeric" value={quantity} onChange={(event) => setQuantities((previous) => ({ ...previous, [item.lineId!]: event.target.value }))} /></label>
              <p>Укажите 0, чтобы не включать строку в отгрузку.</p>
              <VinEditor rows={vins[item.lineId!] ?? []} onChange={(rows) => setVins((previous) => ({ ...previous, [item.lineId!]: rows }))} cap={/^\d+$/.test(quantity) ? Number(quantity) : 0} label={`VIN — ${name(item.modelId)}`} disabled={pending} />
              <p>К отгрузке: {quantity || '0'} · С VIN: {checked.vins.length} · Без VIN: {Math.max(0, (Number(quantity) || 0) - checked.vins.length)}</p>
            </section>;
          })}
          <p aria-live="polite">Итого к отгрузке: {total} · С VIN: {identified} · Без VIN: {Math.max(0, total - identified)}</p>
        </>}
        {action.kind === 'ship-allocated' && <>
          <p>{order.hasReceivingWarehouse ? 'Назначенные автомобили сразу поступят на склад покупателя.' : 'Для прежней отгрузки без склада получения сохранена ручная приёмка покупателем.'}</p>
          {allocated.length === 0 && <p>Нет назначенных автомобилей для отгрузки.</p>}
          {allocated.map((item) => <label key={item.vehicleId}><input type="checkbox" checked={allocatedIds.includes(item.vehicleId)} onChange={() => setAllocatedIds((previous) => previous.includes(item.vehicleId) ? previous.filter((id) => id !== item.vehicleId) : [...previous, item.vehicleId])} />{item.vin}</label>)}
          <p>Выбрано: {allocatedIds.length}</p>
        </>}
        {action.kind !== 'allocate' && <label>Маршрут<Autocomplete aria-label="Маршрут" required value={route} onChange={setRoute} options={Object.entries(routeLabel).map(([value, label]) => ({ value, label }))} /></label>}
      </fieldset>
      {filling && <p role="status">Подбор автомобилей по всем страницам…</p>}
      <div className="order-fulfilment-footer"><Button icon="back" disabled={pending || filling} onClick={onBack}>Назад к заказу</Button><Button type="submit" variant="primary" disabled={pending || filling || !allowed}>{pending ? 'Отправка…' : title}</Button></div>
    </form>
  );
}
