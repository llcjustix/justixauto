import { useEffect, useRef, useState } from 'react';
import { useQueryClient } from '@tanstack/react-query';
import { ApiError, Autocomplete, Button, Modal, Notice, errorText, money, post, toMinor, useData, useSession } from '@justixauto/kit';
import { schemeLabel, stageLabel } from './data';
import type { Branch, Customer, Deal, Lead, Model } from './data';
import { positiveAmount } from './installment-calculator';
import { InstallmentPreview, InstallmentTermsForm, draftTerms, previewTerms, termsValues } from './installment-terms-form';
import { colorLabel } from './vehicle-colors';
import { SaleOfferPicker } from './sale-offer-picker';
import { checkAppliedSource, matchesSaleSource, readSaleSource, readSaleVehicles, salePriceText, saleSelectorItems, sourceDetailKey } from './sale-offer-source';
import type { AppliedSaleSource, SaleSourceSeed } from './sale-offer-source';

export function NewSale({ autoOpen = false }: { autoOpen?: boolean }) {
  const [open, setOpen] = useState(autoOpen);
  return <><Button variant="primary" onClick={() => setOpen(true)}>Новая продажа</Button>
    {open && <SaleForm onClose={() => setOpen(false)} />}</>;
}

export function SaleForm({ onClose, sourceSeed }: { onClose: () => void; sourceSeed?: SaleSourceSeed | undefined }) {
  const session = useSession();
  const company = session.company?.id ?? '';
  const contextKey = `${company}:${session.view.context.revision}`;
  const initialContext = useRef(contextKey);
  // A context change discards committed IDs and any in-flight preflight.
  return <CompanySaleForm key={contextKey} company={company}
    onClose={onClose} sourceSeed={contextKey === initialContext.current ? sourceSeed : undefined} />;
}

function CompanySaleForm({ onClose, sourceSeed, company }: { onClose: () => void; sourceSeed?: SaleSourceSeed | undefined; company: string }) {
  const customers = useData(['customers', 'sale', company], () => saleSelectorItems<Customer>('/retail/customers'), !!company);
  const vehicles = useData(['vehicles', 'sale-eligible', company], readSaleVehicles, !!company);
  const branches = useData(['branches', 'sale', company], () => saleSelectorItems<Branch>(`/identity/companies/${encodeURIComponent(company)}/branches`), !!company);
  const leads = useData(['leads', 'sale', company], () => saleSelectorItems<Lead>('/retail/leads'), !!company);
  const models = useData(['models', 'sale', company], () => saleSelectorItems<Model>('/inventory/vehicle-models'), !!company);
  const [applied, setApplied] = useState<AppliedSaleSource | null>(null);
  const [picker, setPicker] = useState(() => !!sourceSeed);
  const [pickerSeed, setPickerSeed] = useState(sourceSeed);
  const [sourceFailure, setSourceFailure] = useState('');
  const source = useData(sourceDetailKey(company, applied?.seed), () => readSaleSource(applied!.seed), !!applied && !!company);
  const sourceIssue = applied ? sourceFailure || (source.error ? errorText(source.error) : source.data ? checkAppliedSource(source.data, applied, company) : 'Загрузка предложения…') : '';
  const client = useQueryClient();
  const [values, setValues] = useState({ customerId: '', leadId: '', vehicleId: '', branchId: '', paymentScheme: '', price: '', currency: 'USD' });
  const [colors, setColors] = useState({ exterior: '', interior: '' });
  const [terms, setTerms] = useState(() => termsValues());
  const [busy, setBusy] = useState(false), [error, setError] = useState(''), [saved, setSaved] = useState<Deal | null>(null);
  const [refreshFailed, setRefreshFailed] = useState(false);
  const sending = useRef(false), completed = useRef(false);
  const alive = useRef(true);
  useEffect(() => { alive.current = true; return () => { alive.current = false; }; }, []);
  const price = { amountMinor: toMinor(values.price) ?? '', currency: values.currency };
  const own = values.paymentScheme === 'own-installment';
  const preview = previewTerms(price, terms);
  const queries = [customers, vehicles, branches, leads, models];
  const optionsError = queries.find(query => query.error)?.error;
  const healthy = !!company && queries.every(query => !!query.data && !query.error && !query.isFetching);
  const set = (name: keyof typeof values, value: string) => setValues(current => ({ ...current, [name]: value }));
  const availableVehicles = (vehicles.data ?? []).filter(vehicle => matchesSaleSource(vehicle, applied));
  const matchesColors = (vehicle: typeof availableVehicles[number], next = colors) =>
    (!next.exterior || (vehicle.exteriorColor ?? '').trim().toLowerCase() === next.exterior.toLowerCase()) &&
    (!next.interior || (vehicle.interiorColor ?? '').trim().toLowerCase() === next.interior.toLowerCase());
  const colorOptions = (kind: 'exterior' | 'interior') => [...new Set(
    availableVehicles
      .filter(vehicle => kind === 'exterior' || !colors.exterior || (vehicle.exteriorColor ?? '').trim().toLowerCase() === colors.exterior.toLowerCase())
      .filter(vehicle => kind === 'interior' || !colors.interior || (vehicle.interiorColor ?? '').trim().toLowerCase() === colors.interior.toLowerCase())
      .map(vehicle => (kind === 'exterior' ? vehicle.exteriorColor : vehicle.interiorColor)?.trim())
      .filter((color): color is string => !!color),
  )];
  const exteriorOptions = colorOptions('exterior');
  const interiorOptions = colorOptions('interior');
  const valid = healthy && !picker && !sourceIssue && (!applied || !source.isFetching)
    && !!customers.data?.some(customer => customer.id === values.customerId)
    && !!branches.data?.some(branch => branch.id === values.branchId)
    && (!values.leadId || !!leads.data?.some(lead => lead.id === values.leadId && !lead.dealId && ['qualified', 'test-drive', 'negotiation'].includes(lead.stage)))
    && availableVehicles.some(vehicle => vehicle.id === values.vehicleId && matchesColors(vehicle))
    && Object.hasOwn(schemeLabel, values.paymentScheme)
    && ['USD', 'UZS', 'EUR', 'RUB', 'KZT'].includes(values.currency) && positiveAmount(price.amountMinor) && (!own || !!preview.draft);
  useEffect(() => {
    setColors(current => ({
      exterior: exteriorOptions.some(color => color.toLowerCase() === current.exterior.toLowerCase()) ? current.exterior : '',
      interior: interiorOptions.some(color => color.toLowerCase() === current.interior.toLowerCase()) ? current.interior : '',
    }));
  }, [vehicles.data, applied]);
  useEffect(() => {
    if (values.vehicleId && !availableVehicles.some(vehicle => vehicle.id === values.vehicleId && matchesColors(vehicle))) set('vehicleId', '');
  }, [values.vehicleId, colors.exterior, colors.interior, vehicles.data, applied]);
  const detach = () => { if (sending.current) return; setApplied(null); setPicker(false); setPickerSeed(undefined); setSourceFailure(''); };
  const apply = (next: AppliedSaleSource) => {
    if (sending.current) return;
    setValues(current => ({ ...current, price: salePriceText(next.price.amountMinor), currency: next.price.currency,
      vehicleId: vehicles.data?.some(vehicle => vehicle.id === current.vehicleId && matchesSaleSource(vehicle, next)) ? current.vehicleId : '',
    }));
    setApplied(next); setSourceFailure(''); setPicker(false); setPickerSeed(undefined);
  };
  const close = () => { if (!sending.current) onClose(); };
  async function refresh() {
    const results = await Promise.allSettled([['deals'], ['leads'], ['vehicles']].map(queryKey => client.invalidateQueries({ queryKey }, { throwOnError: true })));
    setRefreshFailed(results.some(result => result.status === 'rejected'));
  }
  // Query notifications may reach React after a forced submit event. Consult the
  // cache itself as well as rendered state, including during an async preflight.
  function currentOptionsValid() {
    const keys = [['customers', 'sale', company], ['vehicles', 'sale-eligible', company], ['branches', 'sale', company], ['leads', 'sale', company], ['models', 'sale', company]];
    if (keys.some(key => { const query = client.getQueryState(key); return query?.status !== 'success' || query.fetchStatus !== 'idle'; })) return false;
    if (!client.getQueryData<Customer[]>(keys[0]!)?.some(item => item.id === values.customerId)) return false;
    if (!client.getQueryData<Branch[]>(keys[2]!)?.some(item => item.id === values.branchId)) return false;
    if (values.leadId && !client.getQueryData<Lead[]>(keys[3]!)?.some(item => item.id === values.leadId && !item.dealId && ['qualified', 'test-drive', 'negotiation'].includes(item.stage))) return false;
    if (!client.getQueryData<Awaited<ReturnType<typeof readSaleVehicles>>>(keys[1]!)?.some(item => item.id === values.vehicleId && matchesSaleSource(item, applied) && matchesColors(item))) return false;
    if (applied) {
      const query = client.getQueryState<Awaited<ReturnType<typeof readSaleSource>>>(sourceDetailKey(company, applied.seed));
      if (query?.status !== 'success' || query.fetchStatus !== 'idle' || !query.data || checkAppliedSource(query.data, applied, company)) return false;
    }
    return true;
  }
  async function submit() {
    if (sending.current || completed.current || !valid || !currentOptionsValid()) return;
    sending.current = true; setBusy(true); setError('');
    try {
      if (applied) {
        try {
          const [freshSource, freshVehicles] = await Promise.all([readSaleSource(applied.seed), readSaleVehicles()]);
          if (!alive.current) return;
          client.setQueryData(sourceDetailKey(company, applied.seed), freshSource);
          client.setQueryData(['vehicles', 'sale-eligible', company], freshVehicles);
          const issue = checkAppliedSource(freshSource, applied, company);
          if (issue) throw new Error(issue);
          if (!freshVehicles.some(vehicle => vehicle.id === values.vehicleId && matchesSaleSource(vehicle, applied) && matchesColors(vehicle)))
            throw new Error('Выбранный VIN больше не доступен на складе. Выберите автомобиль заново.');
        } catch (error) {
          if (alive.current) setSourceFailure(errorText(error));
          throw error;
        }
      }
      if (!alive.current) return;
      if (!currentOptionsValid()) throw new Error('Варианты изменились или загружаются. Проверьте выбранные значения.');
      const response = await post<Deal>('/retail/deals', { customerId: values.customerId, leadId: values.leadId || null,
        vehicleId: values.vehicleId, branchId: values.branchId, paymentScheme: values.paymentScheme, price,
        ...(own && preview.draft ? { installmentTerms: draftTerms(preview.draft) } : {}),
      });
      if (!alive.current) return;
      completed.current = true; setSaved(response.data);
      await refresh();
    } catch (e) {
      setError([errorText(e), e instanceof ApiError ? Object.values(e.fields).join('; ') : ''].filter(Boolean).join(' '));
    } finally { sending.current = false; if (alive.current) setBusy(false); }
  }
  const select = (name: keyof typeof values, label: string, options: [string, string][], required = true) =>
    <label className="kit-field">{label}<Autocomplete aria-label={label} required={required} disabled={busy} value={values[name]} onChange={value => set(name, value)}
      placeholder="Выберите…" allowClear={name !== 'currency'}
      options={[...(!required ? [{ value: '', label: 'Без лида' }] : []), ...options.map(([value, label]) => ({ value, label }))]} /></label>;
  return <div className="sale-dialog"><Modal title="Новая продажа" size="wide" onClose={close}>
    {saved ? <div className="kit-stack">
      <Notice>Продажа создана. {money(saved.price)}</Notice>
      {saved.installmentDraft && <><p>Предварительный график сохранён. Ежемесячные счета появятся после подписания договора и выставления счёта первого взноса.</p><InstallmentPreview draft={saved.installmentDraft} /></>}
      {refreshFailed && <Notice kind="danger">Не удалось обновить данные. Продажа уже создана.
        <Button disabled={busy} onClick={() => { sending.current = true; setBusy(true); void refresh().finally(() => { sending.current = false; setBusy(false); }); }}>Повторить обновление</Button>
      </Notice>}
      <Button disabled={busy} onClick={close}>Готово</Button>
    </div> : <form className="kit-stack" onSubmit={e => { e.preventDefault(); void submit(); }}>
      <p>Продажа резервирует выбранный автомобиль. Лид можно связать при наличии.</p>
      {applied && <div className="kit-stack"><p aria-label="Применённое предложение">{applied.label}</p>
        {sourceIssue && <Notice kind="danger">{sourceIssue}</Notice>}
        <div className="kit-row"><Button disabled={busy} onClick={() => { setPickerSeed(applied.seed); setPicker(true); }}>Изменить / применить заново</Button>
          <Button disabled={busy} onClick={detach}>Отключить предложение</Button></div>
      </div>}
      {!picker && !applied && <Button disabled={busy} onClick={() => { setPickerSeed(undefined); setPicker(true); }}>Выбрать предложение</Button>}
      {picker && <SaleOfferPicker company={company} seed={pickerSeed} models={models.data ?? []} disabled={busy} onApply={apply} onDetach={detach}
        onCancel={() => { if (!sending.current && (!applied || !sourceIssue && !source.isFetching)) { setPicker(false); setPickerSeed(undefined); } }} />}
      {applied && !vehicles.isFetching && !vehicles.error && availableVehicles.length === 0 && <Notice>На складе нет подходящего VIN. Измените предложение или отключите его для ручного выбора.</Notice>}
      {error && <Notice kind="danger">{error}</Notice>}
      {optionsError && <Notice kind="danger">Не удалось загрузить варианты. {errorText(optionsError)}<Button disabled={busy} onClick={() => { for (const query of queries) void query.refetch(); }}>Повторить загрузку вариантов</Button></Notice>}
      <div className="sale-creation-fields">
        {select('customerId', 'Клиент', (customers.data ?? []).map(c => [c.id, `${c.displayName}${c.phone ? ` · ${c.phone}` : ''}`]))}
        {select('leadId', 'Лид (необязательно)', (leads.data ?? []).filter(l => !l.dealId && ['qualified', 'test-drive', 'negotiation'].includes(l.stage)).map(l => [l.id, `${l.customer?.displayName ?? ''} · ${stageLabel[l.stage]}`]), false)}
        <label className="kit-field">Цвет кузова<Autocomplete aria-label="Цвет кузова" disabled={busy} value={colors.exterior} onChange={value => setColors(current => ({ ...current, exterior: value }))}
          options={[{ value: '', label: 'Все цвета кузова' }, ...exteriorOptions.map(color => ({ value: color, label: colorLabel(color) }))]} /></label>
        <label className="kit-field">Цвет салона<Autocomplete aria-label="Цвет салона" disabled={busy} value={colors.interior} onChange={value => setColors(current => ({ ...current, interior: value }))}
          options={[{ value: '', label: 'Все цвета салона' }, ...interiorOptions.map(color => ({ value: color, label: colorLabel(color) }))]} /></label>
        {select('vehicleId', 'Автомобиль', availableVehicles.filter(vehicle => matchesColors(vehicle)).map(v => {
          const model = models.data?.find(m => m.id === v.modelId)?.specification;
          return [v.id, `${model ? `${model.make} ${model.model} ${model.variant}` : '—'} · ${v.vin} · Кузов: ${colorLabel(v.exteriorColor)} · Салон: ${colorLabel(v.interiorColor)}`];
        }))}
        {select('branchId', 'Филиал', (branches.data ?? []).map(b => [b.id, b.name]))}
        {select('paymentScheme', 'Способ оформления', Object.entries(schemeLabel))}
        <label className="kit-field">Цена<input aria-label="Цена" inputMode="decimal" required value={values.price} disabled={busy} onChange={e => set('price', e.target.value)} /></label>
        {select('currency', 'Валюта', ['USD', 'UZS', 'EUR', 'RUB', 'KZT'].map(c => [c, c]))}
      </div>
      {own && <InstallmentTermsForm price={price} values={terms} onChange={setTerms} disabled={busy} />}
      <div className="kit-row"><Button disabled={busy} onClick={close}>Отмена</Button><Button type="submit" variant="primary" busy={busy} disabled={!valid || !!optionsError || queries.some(q => q.isLoading)}>Создать продажу</Button></div>
    </form>}
  </Modal></div>;
}
