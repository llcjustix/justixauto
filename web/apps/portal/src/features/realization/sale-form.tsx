import { useEffect, useRef, useState } from 'react';
import { useQueryClient } from '@tanstack/react-query';
import type { QueryClient } from '@tanstack/react-query';
import {
  ApiError,
  Autocomplete,
  Button,
  Details,
  Modal,
  Notice,
  errorText,
  money,
  post,
  toMinor,
  useCommercialAccess,
  useData,
  useSession,
} from '@justixauto/kit';
import { schemeLabel, stageLabel } from './data';
import type { Branch, Customer, Deal, Lead, Model, Money, Vehicle } from './data';
import { positiveAmount } from './installment-calculator';
import {
  InstallmentPreview,
  InstallmentTermsForm,
  draftTerms,
  previewTerms,
  termsValues,
} from './installment-terms-form';
import { colorLabel } from './vehicle-colors';
import { SaleOfferPicker } from './sale-offer-picker';
import {
  checkAppliedSource,
  matchesSaleSource,
  readSaleSource,
  readSaleVehicles,
  salePriceText,
  saleSelectorItems,
  sourceDetailKey,
} from './sale-offer-source';
import type { AppliedSaleSource, SaleSourceDetail, SaleSourceSeed } from './sale-offer-source';

export function NewSale({
  autoOpen = false,
  onOpenDeal,
}: {
  autoOpen?: boolean;
  onOpenDeal?: ((id: string) => void) | undefined;
}) {
  const [open, setOpen] = useState(autoOpen);
  const commercial = useCommercialAccess();
  return (
    <>
      <Button variant="primary" disabled={!commercial.allowed} onClick={() => setOpen(true)}>
        Новая продажа
      </Button>
      {!commercial.allowed && (
        <Notice kind="warning">
          {commercial.reason} <a href="/settings">Открыть настройки компании</a>.
        </Notice>
      )}
      {open && (
        <SaleForm
          onClose={() => setOpen(false)}
          onOpenDeal={
            onOpenDeal &&
            ((id) => {
              setOpen(false);
              onOpenDeal(id);
            })
          }
        />
      )}
    </>
  );
}

type SaleFormProps = {
  onClose: () => void;
  onOpenDeal?: ((id: string) => void) | undefined;
  sourceSeed?: SaleSourceSeed | undefined;
};
type CompanySaleProps = SaleFormProps & { company: string; contextKey: string };
type SaleValues = {
  customerId: string;
  leadId: string;
  vehicleId: string;
  branchId: string;
  paymentScheme: string;
  price: string;
  currency: string;
};

export function SaleForm({ onClose, onOpenDeal, sourceSeed }: SaleFormProps) {
  const session = useSession();
  const commercial = useCommercialAccess();
  const company = session.company?.id ?? '';
  const contextKey = `${session.view.user.id}:${company}:${session.view.context.revision}`;
  const [initialContext] = useState(contextKey);
  if (!commercial.allowed)
    return (
      <div className="sale-dialog">
        <Modal title="Новая продажа" onClose={onClose}>
          <Notice kind="warning">
            {commercial.reason} <a href="/settings">Открыть настройки компании</a>.
          </Notice>
        </Modal>
      </div>
    );
  // A context change discards committed IDs and any in-flight preflight.
  return (
    <CompanySaleForm
      key={contextKey}
      company={company}
      contextKey={contextKey}
      onClose={onClose}
      onOpenDeal={onOpenDeal}
      sourceSeed={contextKey === initialContext ? sourceSeed : undefined}
    />
  );
}

function useSaleOptions(company: string, contextKey: string) {
  const customers = useData(
    ['customers', 'sale', contextKey],
    () => saleSelectorItems<Customer>('/retail/customers'),
    !!company,
  );
  const vehicles = useData(['vehicles', 'sale-eligible', contextKey], readSaleVehicles, !!company);
  const branches = useData(
    ['branches', 'sale', contextKey],
    () => saleSelectorItems<Branch>(`/identity/companies/${encodeURIComponent(company)}/branches`),
    !!company,
  );
  const leads = useData(['leads', 'sale', contextKey], () => saleSelectorItems<Lead>('/retail/leads'), !!company);
  const models = useData(
    ['models', 'sale', contextKey],
    () => saleSelectorItems<Model>('/inventory/vehicle-models'),
    !!company,
  );
  const queries = [customers, vehicles, branches, leads, models];
  const optionsError = queries.find((query) => query.error)?.error;
  const healthy = !!company && queries.every((query) => !!query.data && !query.error && !query.isFetching);
  return { customers, vehicles, branches, leads, models, queries, optionsError, healthy };
}

function appliedSourceIssue(
  applied: AppliedSaleSource | null,
  failure: string,
  source: { data: SaleSourceDetail | undefined; error: unknown },
  company: string,
) {
  if (!applied) return '';
  if (failure) return failure;
  if (source.error) return errorText(source.error);
  return source.data ? checkAppliedSource(source.data, applied, company) : 'Загрузка предложения…';
}

const eligibleLead = (lead: Lead) => !lead.dealId && ['qualified', 'test-drive', 'negotiation'].includes(lead.stage);

function saleValuesValid(
  values: SaleValues,
  options: ReturnType<typeof useSaleOptions>,
  availableVehicles: Vehicle[],
  price: Money,
  preview: ReturnType<typeof previewTerms>,
) {
  return (
    !!options.customers.data?.some((customer) => customer.id === values.customerId) &&
    !!options.branches.data?.some((branch) => branch.id === values.branchId) &&
    (!values.leadId || !!options.leads.data?.some((lead) => lead.id === values.leadId && eligibleLead(lead))) &&
    availableVehicles.some((vehicle) => vehicle.id === values.vehicleId) &&
    Object.hasOwn(schemeLabel, values.paymentScheme) &&
    ['USD', 'UZS', 'EUR', 'RUB', 'KZT'].includes(values.currency) &&
    positiveAmount(price.amountMinor) &&
    (values.paymentScheme !== 'own-installment' || !!preview.draft)
  );
}

function cachedSourceValid(
  client: QueryClient,
  applied: AppliedSaleSource | null,
  company: string,
  contextKey: string,
) {
  if (!applied) return true;
  const query = client.getQueryState<Awaited<ReturnType<typeof readSaleSource>>>(
    sourceDetailKey(contextKey, applied.seed),
  );
  return (
    query?.status === 'success' &&
    query.fetchStatus === 'idle' &&
    !!query.data &&
    !checkAppliedSource(query.data, applied, company)
  );
}

function cachedOptionsValid(
  client: QueryClient,
  values: SaleValues,
  applied: AppliedSaleSource | null,
  contextKey: string,
) {
  const keys = [
    ['customers', 'sale', contextKey],
    ['vehicles', 'sale-eligible', contextKey],
    ['branches', 'sale', contextKey],
    ['leads', 'sale', contextKey],
    ['models', 'sale', contextKey],
  ];
  if (
    keys.some((key) => {
      const query = client.getQueryState(key);
      return query?.status !== 'success' || query.fetchStatus !== 'idle';
    })
  )
    return false;
  if (!client.getQueryData<Customer[]>(keys[0]!)?.some((item) => item.id === values.customerId)) return false;
  if (!client.getQueryData<Branch[]>(keys[2]!)?.some((item) => item.id === values.branchId)) return false;
  if (
    values.leadId &&
    !client.getQueryData<Lead[]>(keys[3]!)?.some((item) => item.id === values.leadId && eligibleLead(item))
  )
    return false;
  return !!client
    .getQueryData<Vehicle[]>(keys[1]!)
    ?.some((item) => item.id === values.vehicleId && matchesSaleSource(item, applied));
}

function useCompanySaleForm({ onClose, onOpenDeal, sourceSeed, company, contextKey }: CompanySaleProps) {
  const options = useSaleOptions(company, contextKey);
  const { vehicles } = options;
  const [applied, setApplied] = useState<AppliedSaleSource | null>(null);
  const [picker, setPicker] = useState(() => !!sourceSeed);
  const [pickerSeed, setPickerSeed] = useState(sourceSeed);
  const [sourceFailure, setSourceFailure] = useState('');
  const source = useData(
    sourceDetailKey(contextKey, applied?.seed),
    () => readSaleSource(applied!.seed),
    !!applied && !!company,
  );
  const sourceIssue = appliedSourceIssue(applied, sourceFailure, source, company);
  const client = useQueryClient();
  const [values, setValues] = useState<SaleValues>({
    customerId: '',
    leadId: '',
    vehicleId: '',
    branchId: '',
    paymentScheme: '',
    price: '',
    currency: 'USD',
  });
  const [terms, setTerms] = useState(() => termsValues());
  const [busy, setBusy] = useState(false),
    [error, setError] = useState(''),
    [saved, setSaved] = useState<Deal | null>(null);
  const [refreshFailed, setRefreshFailed] = useState(false);
  const sending = useRef(false),
    completed = useRef(false);
  const alive = useRef(true);
  useEffect(() => {
    alive.current = true;
    return () => {
      alive.current = false;
    };
  }, []);
  const price = { amountMinor: toMinor(values.price) ?? '', currency: values.currency };
  const own = values.paymentScheme === 'own-installment';
  const preview = previewTerms(price, terms);
  const set = (name: keyof typeof values, value: string) => setValues((current) => ({ ...current, [name]: value }));
  const availableVehicles = (vehicles.data ?? []).filter((vehicle) => matchesSaleSource(vehicle, applied));
  const valid =
    options.healthy &&
    !picker &&
    !sourceIssue &&
    (!applied || !source.isFetching) &&
    saleValuesValid(values, options, availableVehicles, price, preview);
  // Reconcile only the invalid vehicle selection before rendering the form.
  // Other draft fields survive inventory refreshes and proposal changes.
  if (values.vehicleId && !availableVehicles.some((vehicle) => vehicle.id === values.vehicleId)) {
    setValues({ ...values, vehicleId: '' });
  }
  const detach = () => {
    if (sending.current) return;
    setApplied(null);
    setPicker(false);
    setPickerSeed(undefined);
    setSourceFailure('');
  };
  const apply = (next: AppliedSaleSource) => {
    if (sending.current) return;
    setValues((current) => ({
      ...current,
      price: salePriceText(next.price.amountMinor),
      currency: next.price.currency,
      vehicleId: vehicles.data?.some((vehicle) => vehicle.id === current.vehicleId && matchesSaleSource(vehicle, next))
        ? current.vehicleId
        : '',
    }));
    setApplied(next);
    setSourceFailure('');
    setPicker(false);
    setPickerSeed(undefined);
  };
  const close = () => {
    if (!sending.current) onClose();
  };
  async function refresh() {
    const results = await Promise.allSettled(
      [['deals'], ['leads'], ['vehicles']].map((queryKey) =>
        client.invalidateQueries({ queryKey }, { throwOnError: true }),
      ),
    );
    setRefreshFailed(results.some((result) => result.status === 'rejected'));
  }
  // Query notifications may reach React after a forced submit event. Consult the
  // cache itself as well as rendered state, including during an async preflight.
  function currentOptionsValid() {
    return (
      cachedOptionsValid(client, values, applied, contextKey) && cachedSourceValid(client, applied, company, contextKey)
    );
  }
  async function preflight() {
    const [freshSource, freshVehicles] = await Promise.all([
      applied ? readSaleSource(applied.seed) : Promise.resolve(null),
      readSaleVehicles(),
    ]);
    if (!alive.current) return false;
    if (applied && freshSource) client.setQueryData(sourceDetailKey(contextKey, applied.seed), freshSource);
    client.setQueryData(['vehicles', 'sale-eligible', contextKey], freshVehicles);
    const issue = applied && freshSource ? checkAppliedSource(freshSource, applied, company) : '';
    if (issue) {
      setSourceFailure(issue);
      throw new Error(issue);
    }
    if (!freshVehicles.some((vehicle) => vehicle.id === values.vehicleId && matchesSaleSource(vehicle, applied))) {
      set('vehicleId', '');
      throw new Error('Выбранный VIN больше не доступен на складе. Выберите автомобиль заново.');
    }
    if (!alive.current) return false;
    return true;
  }
  async function recoverConflict(error: unknown) {
    if (!(error instanceof ApiError) || error.status !== 409 || !alive.current) return;
    try {
      const freshVehicles = await readSaleVehicles();
      if (!alive.current) return;
      client.setQueryData(['vehicles', 'sale-eligible', contextKey], freshVehicles);
      if (!freshVehicles.some((vehicle) => vehicle.id === values.vehicleId && matchesSaleSource(vehicle, applied)))
        set('vehicleId', '');
    } catch {
      /* Preserve the server conflict; the next reload can be retried. */
    }
  }
  async function submit() {
    if (sending.current || completed.current || !valid || !currentOptionsValid()) return;
    sending.current = true;
    setBusy(true);
    setError('');
    try {
      if (!(await preflight()) || !alive.current) return;
      if (!currentOptionsValid()) throw new Error('Варианты изменились или загружаются. Проверьте выбранные значения.');
      const response = await post<Deal>('/retail/deals', {
        customerId: values.customerId,
        leadId: values.leadId || null,
        vehicleId: values.vehicleId,
        branchId: values.branchId,
        paymentScheme: values.paymentScheme,
        price,
        ...(own && preview.draft ? { installmentTerms: draftTerms(preview.draft) } : {}),
      });
      if (!alive.current) return;
      completed.current = true;
      setSaved(response.data);
      await refresh();
    } catch (e) {
      await recoverConflict(e);
      if (alive.current)
        setError(
          [errorText(e), e instanceof ApiError ? Object.values(e.fields).join('; ') : ''].filter(Boolean).join(' '),
        );
    } finally {
      sending.current = false;
      if (alive.current) setBusy(false);
    }
  }
  const retryRefresh = () => {
    sending.current = true;
    setBusy(true);
    void refresh().finally(() => {
      sending.current = false;
      setBusy(false);
    });
  };
  const openSavedDeal = () => {
    if (!sending.current && saved && onOpenDeal) onOpenDeal(saved.id);
  };
  const cancelPicker = () => {
    if (!sending.current && (!applied || (!sourceIssue && !source.isFetching))) {
      setPicker(false);
      setPickerSeed(undefined);
    }
  };
  return {
    ...options,
    company,
    applied,
    picker,
    pickerSeed,
    sourceIssue,
    values,
    terms,
    busy,
    error,
    saved,
    refreshFailed,
    price,
    own,
    availableVehicles,
    valid,
    set,
    setTerms,
    setPicker,
    setPickerSeed,
    detach,
    apply,
    close,
    submit,
    retryRefresh,
    openSavedDeal,
    cancelPicker,
    canOpenDeal: !!onOpenDeal,
  };
}

type SaleFormState = ReturnType<typeof useCompanySaleForm>;

function CompanySaleForm(props: CompanySaleProps) {
  const form = useCompanySaleForm(props);
  return (
    <div className="sale-dialog">
      <Modal title="Новая продажа" size="wide" onClose={form.close}>
        {form.saved ? <SaleCreated form={form} saved={form.saved} /> : <SaleEditor form={form} />}
      </Modal>
    </div>
  );
}

function savedVehicleLabel(saved: Deal, vehicles: Vehicle[] | undefined, models: Model[] | undefined) {
  const vehicle = vehicles?.find((item) => item.id === saved.vehicleId);
  const model = models?.find((item) => item.id === vehicle?.modelId)?.specification;
  return vehicle
    ? `${model ? `${model.make} ${model.model} ${model.variant}` : ''} · ${vehicle.vin}`.replace(/^ · /, '')
    : (saved.vehicleSnapshot?.vin ?? '—');
}

function SaleCreated({ form, saved }: { form: SaleFormState; saved: Deal }) {
  const {
    customers,
    vehicles,
    models,
    branches,
    refreshFailed,
    busy,
    close,
    retryRefresh,
    openSavedDeal,
    canOpenDeal,
  } = form;
  return (
    <div className="kit-stack sale-created">
      <Notice kind="success">
        Продажа создана. Автомобиль зарезервирован за клиентом; следующий шаг — договор и счёт.
      </Notice>
      <Details
        items={[
          [
            'Клиент',
            customers.data?.find((c) => c.id === saved.customer?.id)?.displayName ?? saved.customer?.displayName ?? '—',
          ],
          ['Автомобиль', savedVehicleLabel(saved, vehicles.data, models.data)],
          ['Филиал', branches.data?.find((b) => b.id === saved.branchId)?.name ?? '—'],
          ['Способ оформления', schemeLabel[saved.paymentScheme] ?? saved.paymentScheme],
          ['Цена', money(saved.price)],
        ]}
      />
      {saved.installmentDraft && (
        <>
          <p className="cell-sub">
            Предварительный график сохранён. Ежемесячные счета появятся после подписания договора и выставления счёта
            первого взноса.
          </p>
          <InstallmentPreview draft={saved.installmentDraft} />
        </>
      )}
      {refreshFailed && (
        <Notice kind="danger">
          Не удалось обновить данные. Продажа уже создана.
          <Button disabled={busy} onClick={retryRefresh}>
            Повторить обновление
          </Button>
        </Notice>
      )}
      <div className="kit-row sale-creation-actions">
        <Button variant="back" disabled={busy} onClick={close}>
          Назад к продажам
        </Button>
        <span className="sale-created-spacer" />
        <Button disabled={busy} onClick={close}>
          Готово
        </Button>
        {canOpenDeal && (
          <Button variant="primary" disabled={busy} onClick={openSavedDeal}>
            Открыть сделку
          </Button>
        )}
      </div>
    </div>
  );
}

function SaleSourceFields({ form }: { form: SaleFormState }) {
  const {
    applied,
    sourceIssue,
    busy,
    setPickerSeed,
    setPicker,
    picker,
    detach,
    company,
    pickerSeed,
    models,
    apply,
    cancelPicker,
    vehicles,
    availableVehicles,
  } = form;
  return (
    <>
      {applied && (
        <div className="kit-stack">
          <p aria-label="Применённое предложение">{applied.label}</p>
          {sourceIssue && <Notice kind="danger">{sourceIssue}</Notice>}
          <div className="kit-row">
            <Button
              disabled={busy}
              onClick={() => {
                setPickerSeed(applied.seed);
                setPicker(true);
              }}
            >
              Изменить / применить заново
            </Button>
            <Button disabled={busy} onClick={detach}>
              Отключить предложение
            </Button>
          </div>
        </div>
      )}
      {!picker && !applied && (
        <Button
          disabled={busy}
          onClick={() => {
            setPickerSeed(undefined);
            setPicker(true);
          }}
        >
          Выбрать предложение
        </Button>
      )}
      {picker && (
        <SaleOfferPicker
          company={company}
          seed={pickerSeed}
          models={models.data ?? []}
          disabled={busy}
          onApply={apply}
          onDetach={detach}
          onCancel={cancelPicker}
        />
      )}
      {applied && !vehicles.isFetching && !vehicles.error && availableVehicles.length === 0 && (
        <Notice>На складе нет подходящего VIN. Измените предложение или отключите его для ручного выбора.</Notice>
      )}
    </>
  );
}

function SaleEditor({ form }: { form: SaleFormState }) {
  const { submit, error, optionsError, busy, queries, own, price, terms, setTerms, close, valid } = form;
  return (
    <form
      className="kit-stack"
      onSubmit={(e) => {
        e.preventDefault();
        void submit();
      }}
    >
      <p>Продажа резервирует выбранный автомобиль. Лид можно связать при наличии.</p>
      <SaleSourceFields form={form} />
      {error && <Notice kind="danger">{error}</Notice>}
      {optionsError && (
        <Notice kind="danger">
          Не удалось загрузить варианты. {errorText(optionsError)}
          <Button
            disabled={busy}
            onClick={() => {
              for (const query of queries) void query.refetch();
            }}
          >
            Повторить загрузку вариантов
          </Button>
        </Notice>
      )}
      <SaleValueFields form={form} />
      {own && <InstallmentTermsForm price={price} values={terms} onChange={setTerms} disabled={busy} />}
      <div className="kit-row sale-creation-actions">
        <Button disabled={busy} onClick={close}>
          Отмена
        </Button>
        <Button
          type="submit"
          variant="primary"
          busy={busy}
          disabled={!valid || !!optionsError || queries.some((q) => q.isLoading)}
        >
          Создать продажу
        </Button>
      </div>
    </form>
  );
}

function SaleValueFields({ form }: { form: SaleFormState }) {
  const { values, busy, set, customers, leads, availableVehicles, models, branches } = form;
  const select = (name: keyof typeof values, label: string, options: [string, string][], required = true) => (
    <label className="kit-field">
      {label}
      <Autocomplete
        aria-label={label}
        required={required}
        disabled={busy}
        value={values[name]}
        onChange={(value) => set(name, value)}
        placeholder="Выберите…"
        allowClear={name !== 'currency'}
        options={[
          ...(!required ? [{ value: '', label: 'Без лида' }] : []),
          ...options.map(([value, label]) => ({ value, label })),
        ]}
      />
    </label>
  );
  return (
    <div className="sale-creation-fields">
      {select(
        'customerId',
        'Клиент',
        (customers.data ?? []).map((c) => [c.id, `${c.displayName}${c.phone ? ` · ${c.phone}` : ''}`]),
      )}
      {select(
        'leadId',
        'Лид (необязательно)',
        (leads.data ?? [])
          .filter(eligibleLead)
          .map((l) => [l.id, `${l.customer?.displayName ?? ''} · ${stageLabel[l.stage]}`]),
        false,
      )}
      {select(
        'vehicleId',
        'Автомобиль',
        availableVehicles.map((v) => {
          const model = models.data?.find((m) => m.id === v.modelId)?.specification;
          return [
            v.id,
            `${model ? `${model.make} ${model.model} ${model.variant}` : '—'} · ${v.vin} · Кузов: ${colorLabel(v.exteriorColor)} · Салон: ${colorLabel(v.interiorColor)}`,
          ];
        }),
      )}
      {select(
        'branchId',
        'Филиал',
        (branches.data ?? []).map((b) => [b.id, b.name]),
      )}
      {select('paymentScheme', 'Способ оформления', Object.entries(schemeLabel))}
      <label className="kit-field">
        Цена
        <input
          aria-label="Цена"
          inputMode="decimal"
          required
          value={values.price}
          disabled={busy}
          onChange={(e) => set('price', e.target.value)}
        />
      </label>
      {select(
        'currency',
        'Валюта',
        ['USD', 'UZS', 'EUR', 'RUB', 'KZT'].map((c) => [c, c]),
      )}
    </div>
  );
}
