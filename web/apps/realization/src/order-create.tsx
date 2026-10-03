import { useEffect, useRef, useState } from 'react';
import { ApiError, Autocomplete, Button, Modal, Notice, money, post, toMinor, useRefresh } from '@justixauto/kit';
import { modelName, routeLabel, useOrderModelDetails, useOrderModels, useOrderSuppliers, useWarehouses } from './data';
import type { Offer, Terms, VehicleColors } from './data';
import { PaymentScheduleEditor, TermsView } from './shared';
import { colorOptions, exactSpecification, selectedColor } from './vehicle-colors';

type DraftLine = VehicleColors & { key: string; modelId: string; quantity: string; price: string; offerLineId?: string; exteriorEdited?: boolean; interiorEdited?: boolean };
type Mode = { kind: 'direct' } | { kind: 'offer'; offer: Offer };
type Option = [string, string];
const blank = (key: string): DraftLine => ({ key, modelId: '', quantity: '1', price: '' });
const validQuantity = (value: string, minimum: number) =>
  /^\d+$/.test(value) && Number(value) >= minimum && Number(value) <= 10000;
const colorValue = (options: string[], value: string | undefined, edited?: boolean) => edited
  ? options.find(option => option.toLowerCase() === value?.toLowerCase()) ?? '' : selectedColor(options, value);

function Selector({ label, value, options, onChange }: {
  label: string; value: string; options: Option[]; onChange: (id: string) => void;
}) {
  return (
    <div className="kit-field">
      <span>{label}</span>
      <Autocomplete aria-label={label} value={value} onChange={onChange} required placeholder="Выберите"
        options={options.map(([value, label]) => ({ value, label }))} />
    </div>
  );
}

function Quantity({ value, minimum, onChange, label }: {
  value: string; minimum: number; onChange: (value: string) => void; label: string;
}) {
  const valid = validQuantity(value, minimum);
  return <div className="kit-row">
    <Button title={`Уменьшить ${label.toLowerCase()}`} disabled={!valid || Number(value) <= minimum} onClick={() => onChange(String(Number(value) - 1))}><span aria-label={`Уменьшить ${label.toLowerCase()}`}>−</span></Button>
    <input aria-label={label} type="number" min={minimum} max="10000" step="1" value={value} onChange={e => onChange(e.target.value)} />
    <Button title={`Увеличить ${label.toLowerCase()}`} disabled={!valid || Number(value) >= 10000} onClick={() => onChange(String(Number(value) + 1))}><span aria-label={`Увеличить ${label.toLowerCase()}`}>+</span></Button>
  </div>;
}

function LoadState({ query, label, empty }: {
  query: { isLoading: boolean; error: unknown; data: unknown[] | undefined; refetch: () => unknown };
  label: string; empty: string;
}) {
  if (query.isLoading) return <Notice>Загрузка: {label}…</Notice>;
  if (query.error) return <Notice kind="danger">Не удалось загрузить: {label}. <Button onClick={() => void query.refetch()}>Повторить: {label}</Button></Notice>;
  if (query.data?.length === 0) return <Notice>{empty}</Notice>;
  return null;
}

export function OrderCreate({ mode, onClose, onCreated }: {
  mode: Mode; onClose: () => void; onCreated: (id: string) => void;
}) {
  // Capture the version at entry so later publication cannot rewrite the draft.
  const [offer] = useState(() => mode.kind === 'offer' ? structuredClone(mode.offer.publishedVersion) : undefined);
  const [step, setStep] = useState(1);
  const [supplier, setSupplier] = useState('');
  const [warehouse, setWarehouse] = useState('');
  const [lines, setLines] = useState<DraftLine[]>(() => offer ? offer.terms.lines.map((line, i) => ({
    key: String(i), modelId: line.modelId, offerLineId: line.lineId!, quantity: '0', price: '',
    modelSpecificationVersion: line.modelSpecificationVersion ?? '', exteriorColor: line.exteriorColor ?? '', interiorColor: line.interiorColor ?? '',
  })) : [blank('0')]);
  const nextKey = useRef(lines.length);
  const [currency, setCurrency] = useState('USD');
  const [route, setRoute] = useState('local');
  const [deliveryTerms, setDelivery] = useState('');
  const [warrantyTerms, setWarranty] = useState('');
  const [serviceTerms, setService] = useState('');
  const [schedule, setSchedule] = useState<{ amount: string; dueDate: string }[]>([]);
  const [error, setError] = useState('');
  const [busy, setBusy] = useState(false);
  const sending = useRef(false);
  const errorRef = useRef<HTMLDivElement>(null);
  const partners = useOrderSuppliers(mode.kind === 'direct');
  const models = useOrderModels();
  const details = useOrderModelDetails(lines.map(line => line.modelId));
  const warehouses = useWarehouses();
  const refresh = useRefresh();
  useEffect(() => { if (error) errorRef.current?.focus(); }, [error]);

  const supplierOptions: Option[] = (partners.data ?? []).filter(p => p.status === 'active').map(p => [p.counterparty.id, p.counterparty.name]);
  const modelOptions: Option[] = (models.data ?? []).map(m => [m.id, modelName(m)]);
  const modelLabel = (id: string) => modelOptions.find(([key]) => key === id)?.[1] ?? id;
  const setLine = (index: number, patch: Partial<DraftLine>) => setLines(current => current.map((line, i) => i === index ? { ...line, ...patch } : line));
  const choices = lines.map(line => {
    const detail = !details.isLoading && !details.error ? details.data?.find(model => model.id === line.modelId) : undefined;
    const version = line.modelSpecificationVersion || detail?.specification.version || '';
    const spec = exactSpecification(detail, version);
    const source = offer?.terms.lines.find(source => source.lineId === line.offerLineId);
    const exterior = colorOptions(spec, 'exterior', source?.exteriorColor);
    const interior = colorOptions(spec, 'interior', source?.interiorColor);
    return { spec, exterior, interior, line: { ...line, modelSpecificationVersion: version,
      exteriorColor: colorValue(exterior, line.exteriorColor, line.exteriorEdited), interiorColor: colorValue(interior, line.interiorColor, line.interiorEdited) } };
  });
  // Pin legacy offers once the exact detail arrives, and discard stale selections.
  // Loading/error never clears an otherwise useful draft and never permits submission.
  useEffect(() => {
    if (details.isLoading || details.error || !details.data) return;
    setLines(current => {
      let changed = false;
      const next = current.map(line => {
        const detail = details.data.find(model => model.id === line.modelId);
        const version = line.modelSpecificationVersion || detail?.specification.version || '';
        const spec = exactSpecification(detail, version);
        if (!spec) return line;
        const source = offer?.terms.lines.find(source => source.lineId === line.offerLineId);
        const exteriorColor = colorValue(colorOptions(spec, 'exterior', source?.exteriorColor), line.exteriorColor, line.exteriorEdited);
        const interiorColor = colorValue(colorOptions(spec, 'interior', source?.interiorColor), line.interiorColor, line.interiorEdited);
        if (line.modelSpecificationVersion === version && line.exteriorColor === exteriorColor && line.interiorColor === interiorColor) return line;
        changed = true;
        return { ...line, modelSpecificationVersion: version, exteriorColor, interiorColor };
      });
      return changed ? next : current;
    });
  }, [details.data, details.error, details.isLoading, offer]);
  const selectedOfferLines = choices.filter(choice => Number(choice.line.quantity) > 0);
  const terms: Terms = offer ? {
    ...offer.terms,
    paymentSchedule: offer.terms.lines.every(source => lines.filter(line => line.offerLineId === source.lineId)
      .reduce((sum, line) => sum + Number(line.quantity), 0) === Number(source.quantity)) ? offer.terms.paymentSchedule : [],
    lines: selectedOfferLines.map(({ line }) => ({ modelId: line.modelId, offerLineId: line.offerLineId!, quantity: line.quantity,
      modelSpecificationVersion: line.modelSpecificationVersion!, exteriorColor: line.exteriorColor!, interiorColor: line.interiorColor!,
      unitPrice: offer.terms.lines.find(source => source.lineId === line.offerLineId)!.unitPrice })),
  } : {
    lines: choices.map(({ line }) => ({ modelId: line.modelId, quantity: line.quantity,
      modelSpecificationVersion: line.modelSpecificationVersion!, exteriorColor: line.exteriorColor!, interiorColor: line.interiorColor!,
      unitPrice: { amountMinor: toMinor(line.price) ?? '0', currency } })),
    route, deliveryTerms, warrantyTerms, serviceTerms,
    paymentSchedule: schedule.map(payment => ({ amount: { amountMinor: toMinor(payment.amount) ?? '0', currency }, dueDate: payment.dueDate })),
  };
  const totals = new Map<string, bigint>();
  for (const line of terms.lines) {
    if (validQuantity(line.quantity, 1)) totals.set(line.unitPrice.currency, (totals.get(line.unitPrice.currency) ?? 0n) + BigInt(line.unitPrice.amountMinor) * BigInt(line.quantity));
  }

  function firstStepProblem() {
    if (!warehouse) return 'Выберите склад получения';
    if (warehouses.error || !warehouses.data?.some(w => w.id === warehouse)) return 'Загрузите и выберите склад получения';
    if (mode.kind === 'offer') return offer ? '' : 'Опубликованная версия акции недоступна';
    if (!supplier) return 'Выберите поставщика';
    if (partners.error || !supplierOptions.some(([id]) => id === supplier)) return 'Загрузите и выберите активного поставщика';
    return '';
  }

  function secondStepProblem() {
    if (offer) {
      if (lines.some(line => !validQuantity(line.quantity, 0))) return 'Количество в акции — целое число от 0 до 10000';
      if (!terms.lines.length) return 'Выберите хотя бы одну модель с положительным количеством';
      return colorsProblem();
    }
    if (models.error || !models.data) return 'Загрузите каталог моделей';
    if (!['USD', 'UZS', 'EUR'].includes(currency)) return 'Выберите валюту';
    if (!Object.hasOwn(routeLabel, route)) return 'Выберите маршрут поставки';
    if (!lines.length) return 'Добавьте хотя бы одну модель';
    for (const [i, line] of lines.entries()) {
      if (!modelOptions.some(([id]) => id === line.modelId)) return `Выберите модель ${i + 1} из каталога`;
      if (!validQuantity(line.quantity, 1)) return `Количество в строке ${i + 1} — целое число от 1 до 10000`;
      const minor = toMinor(line.price);
      if (minor === null || BigInt(minor) <= 0n) return `Укажите положительную цену в строке ${i + 1}`;
    }
    if (schedule.some(payment => toMinor(payment.amount) === null || !payment.dueDate)) return 'Заполните сумму и дату каждого платежа';
    return colorsProblem();
  }

  function colorsProblem() {
    if (terms.lines.length > 100) return 'Допускается не более 100 строк заказа';
    if (details.isLoading || details.error || !details.data) return 'Загрузите точные версии моделей для выбора цветов';
    for (const [i, choice] of choices.entries()) {
      if (offer && Number(choice.line.quantity) === 0) continue;
      if (!choice.spec) return `Версия модели в строке ${i + 1} недоступна. Обновите данные или выберите другую модель`;
      if (!choice.line.exteriorColor || !choice.line.interiorColor) return `Выберите цвет кузова и салона в строке ${i + 1}`;
    }
    return '';
  }

  function next() {
    const problem = step === 1 ? firstStepProblem() : secondStepProblem();
    setError(problem);
    if (!problem) setStep(step + 1);
    else errorRef.current?.focus();
  }

  async function submit() {
    if (sending.current || step !== 3) return;
    const first = firstStepProblem();
    const second = secondStepProblem();
    if (first || second) { setStep(first ? 1 : 2); setError(first || second); return; }
    sending.current = true;
    setBusy(true);
    setError('');
    let id: string;
    try {
      const payload = offer
        ? { offerVersionId: offer.id, warehouseId: warehouse, lines: terms.lines.map(line => ({ offerLineId: line.offerLineId, quantity: line.quantity,
          modelSpecificationVersion: line.modelSpecificationVersion, exteriorColor: line.exteriorColor, interiorColor: line.interiorColor })) }
        : { supplierCompanyId: supplier, warehouseId: warehouse, terms };
      const created = await post<{ id: string }>('/commerce/orders', payload);
      id = created.data.id;
    } catch (e) {
      setError(e instanceof ApiError ? [e.message, ...Object.values(e.fields)].filter(Boolean).join(' · ') : 'Не удалось создать заказ. Повторите попытку.');
      sending.current = false;
      setBusy(false);
      return;
    }
    // A successful POST must never become retryable because refreshing failed.
    try { await refresh(['orders']); } catch { /* The created order is still opened by its ID. */ }
    onCreated(id);
  }

  function colorFields(index: number) {
    const choice = choices[index]!;
    return <div className="kit-grid-2">
      <label className="kit-field">Цвет кузова {index + 1}<Autocomplete aria-label={`Цвет кузова ${index + 1}`} value={choice.line.exteriorColor ?? ''} disabled={!choice.spec} required placeholder="Выберите"
        onChange={value => setLine(index, { exteriorColor: value, exteriorEdited: true })}
        options={choice.exterior.map(color => ({ value: color, label: color }))} /></label>
      <label className="kit-field">Цвет салона {index + 1}<Autocomplete aria-label={`Цвет салона ${index + 1}`} value={choice.line.interiorColor ?? ''} disabled={!choice.spec} required placeholder="Выберите"
        onChange={value => setLine(index, { interiorColor: value, interiorEdited: true })}
        options={choice.interior.map(color => ({ value: color, label: color }))} /></label>
      {choice.line.modelSpecificationVersion && <small>Версия модели: {choice.line.modelSpecificationVersion}</small>}
      {!details.isLoading && !details.error && choice.line.modelId && !choice.spec && <Notice kind="danger">Точная версия модели недоступна.</Notice>}
    </div>;
  }

  return (
    <Modal title={mode.kind === 'offer' ? 'Заказ по акции' : 'Новый заказ поставщику'} size="wide" onClose={() => { if (!sending.current) onClose(); }} help={`Шаг ${step} из 3`} footer={
      <>
        <Button icon="back" disabled={busy} onClick={() => { setError(''); step === 1 ? onClose() : setStep(step - 1); }}>Назад</Button>
        {step < 3 ? <Button variant="primary" onClick={next}>Далее</Button> : <Button variant="primary" busy={busy} onClick={() => void submit()}>Отправить заказ</Button>}
      </>
    }>
      <div className="order-create kit-stack">
        {error && <div className="kit-notice" data-kind="danger" role="alert" tabIndex={-1} ref={errorRef}>{error}</div>}
        {step === 1 && <>
          <p>Выберите поставщика и склад, на который поступят автомобили.</p>
          {mode.kind === 'offer' ? <p>Поставщик: {mode.offer.supplier.name}</p> : <>
            <LoadState query={partners} label="поставщики" empty="Нет партнёров. Создайте активное партнёрство в разделе «Партнёры»." />
            {partners.data && !supplierOptions.length && partners.data.length > 0 && <Notice>Нет активных поставщиков. Активируйте партнёрство в разделе «Партнёры».</Notice>}
            <Selector label="Поставщик" value={supplier} options={supplierOptions} onChange={setSupplier} />
          </>}
          <LoadState query={warehouses} label="склады" empty="Нет складов. Добавьте склад в разделе «Склады», затем вернитесь к заказу." />
          <Selector label="Склад получения" value={warehouse} options={(warehouses.data ?? []).map(w => [w.id, `${w.name} · свободно ${w.free}`])} onChange={setWarehouse} />
        </>}
        {step === 2 && lines.some(line => line.modelId) && <>
          {details.isLoading && <Notice>Загрузка версий моделей…</Notice>}
          {details.error && <Notice kind="danger">Не удалось загрузить версии моделей.</Notice>}
          {(details.error || (!details.isLoading && choices.some(choice => choice.line.modelId && !choice.spec))) &&
            <Button onClick={() => void details.refetch()}>Повторить загрузку версий</Button>}
        </>}
        {step === 2 && (offer ? <>
          <p>Цены и условия зафиксированы версией акции. Количество может превышать указанное в акции; 0 исключает строку.</p>
          <LoadState query={models} label="модели" empty="Каталог пуст; ниже показаны идентификаторы моделей акции." />
          {lines.map((line, i) => <div className="order-create-line" key={line.key}>
            <span>{modelLabel(line.modelId)} · {money(offer.terms.lines.find(source => source.lineId === line.offerLineId)!.unitPrice)}</span>
            <Quantity label={`Количество ${i + 1}`} value={line.quantity} minimum={0} onChange={quantity => setLine(i, { quantity })} />
            {colorFields(i)}
            <Button onClick={() => {
              const source = offer.terms.lines.find(source => source.lineId === line.offerLineId)!;
              const key = String(nextKey.current++);
              setLines(current => [...current, { ...line, key, quantity: '1', exteriorColor: source.exteriorColor ?? '', interiorColor: source.interiorColor ?? '', exteriorEdited: false, interiorEdited: false }]);
            }}>Добавить сочетание {i + 1}</Button>
          </div>)}
        </> : <>
          <LoadState query={models} label="модели" empty="Нет моделей. Добавьте модель в каталог, затем вернитесь к заказу." />
          <div className="kit-grid-2">
            <label className="kit-field">Валюта<Autocomplete aria-label="Валюта" value={currency} onChange={setCurrency} required allowClear={false} options={['USD', 'UZS', 'EUR'].map(code => ({ value: code, label: code }))} /></label>
            <label className="kit-field">Маршрут поставки<Autocomplete aria-label="Маршрут поставки" value={route} onChange={setRoute} required allowClear={false} options={Object.entries(routeLabel).map(([value, label]) => ({ value, label }))} /></label>
          </div>
          {lines.map((line, i) => <div className="order-create-line" key={line.key}>
            <Selector label={`Модель ${i + 1}`} value={line.modelId} options={modelOptions} onChange={id => setLine(i, { modelId: id,
              modelSpecificationVersion: models.data?.find(model => model.id === id)?.specification.version ?? '', exteriorColor: '', interiorColor: '', exteriorEdited: false, interiorEdited: false })} />
            {colorFields(i)}
            <Quantity label={lines.length === 1 ? 'Количество' : `Количество ${i + 1}`} value={line.quantity} minimum={1} onChange={quantity => setLine(i, { quantity })} />
            <label className="kit-field">Цена ({currency})<input aria-label={lines.length === 1 ? 'Цена' : `Цена ${i + 1}`} value={line.price} inputMode="decimal" onChange={e => setLine(i, { price: e.target.value })} /></label>
            <Button variant="link" onClick={() => setLines(current => current.filter((_, index) => index !== i))}>Удалить модель {i + 1}</Button>
          </div>)}
          <Button onClick={() => { const key = String(nextKey.current++); setLines(current => [...current, blank(key)]); }}>Добавить модель</Button>
          <label className="kit-field">Условия поставки<textarea value={deliveryTerms} onChange={e => setDelivery(e.target.value)} /></label>
          <label className="kit-field">Гарантия<textarea value={warrantyTerms} onChange={e => setWarranty(e.target.value)} /></label>
          <label className="kit-field">Сервис<textarea value={serviceTerms} onChange={e => setService(e.target.value)} /></label>
          <PaymentScheduleEditor schedule={schedule} setSchedule={setSchedule} />
        </>)}
        {step === 3 && <>
          <h3>Проверьте заказ</h3>
          <p>Поставщик: {mode.kind === 'offer' ? mode.offer.supplier.name : supplierOptions.find(([id]) => id === supplier)?.[1]}</p>
          <p>Склад: {warehouses.data?.find(w => w.id === warehouse)?.name}</p>
          <TermsView terms={terms} modelNameOf={modelLabel} />
          {offer && offer.terms.paymentSchedule.length > 0 && terms.paymentSchedule.length === 0 && <Notice>Количество изменено: график оплаты акции не переносится в заказ. Новый график согласуется дополнением.</Notice>}
          {[...totals].map(([code, amount]) => <p key={code}>Итого: <strong>{money({ amountMinor: amount.toString(), currency: code })}</strong></p>)}
          <Notice kind="info">Заказ будет ожидать подтверждения поставщика. После отгрузки автомобили поступят на выбранный склад; VIN могут быть указаны позже.</Notice>
        </>}
      </div>
    </Modal>
  );
}
