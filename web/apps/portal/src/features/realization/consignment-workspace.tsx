import { useRef, useState } from 'react';
import {
  ApiError,
  Button,
  Notice,
  Panel,
  dateTime,
  errorText,
  get,
  list,
  money,
  post,
  useData,
  useRefresh,
  useSession,
} from '@justixauto/kit';
import type { Consignment, ConsignmentReturn, ConsignmentUnit, Invoice, Order, Vehicle, Warehouse } from './data';

import { loadConsignmentReturnHistory } from './consignment-history';

const modeName = { purchase: 'Выкуп', consignment: 'Консигнация' };
const returnName = {
  pending: 'Ожидает решения дилера',
  confirmed: 'Возврат подтверждён',
  rejected: 'Возврат отклонён',
};
const unitName = {
  prepared: 'Назначен, ещё не передан',
  held: 'На хранении у дилера',
  sold: 'Продан',
  returned: 'Возвращён',
};

/** Reads the immutable terms saved on the accepted order, never a draft checkbox. */
export function AcceptedHandover({ order }: { order: Order }) {
  if (order.source !== 'reservation') return null;
  const agreement = order.terms.reservation;
  if (!agreement)
    return (
      <Notice kind="danger">
        Согласованные условия продолжения резерва недоступны. Передачу автомобиля нужно проверить по заказу.
      </Notice>
    );
  return (
    <Panel title="Согласованные условия передачи" padded>
      <p>
        Режим: <b>{modeName[agreement.mode]}</b>. Условия закреплены в принятом заказе из резерва.
      </p>
      <p>Передача: {agreement.handoverTerms || 'Отдельный текст не указан'}.</p>
      {agreement.handoverConditions && <p>Условия передачи: {agreement.handoverConditions}</p>}
      {agreement.mode === 'purchase' && (
        <p>
          {agreement.allowPartialPurchaseHandover
            ? `Частично оплаченный выкуп разрешён соглашением${agreement.minimumPaid ? `; минимум оплаты ${money(agreement.minimumPaid)}` : ''}.`
            : 'Передача при неполной оплате не согласована.'}
        </p>
      )}
      <p>
        Расчёты по заказу:{' '}
        {order.terms.paymentSchedule.length
          ? order.terms.paymentSchedule.map((item) => `${money(item.amount)} до ${item.dueDate}`).join('; ')
          : 'отдельный график не указан'}
        .
      </p>
    </Panel>
  );
}

function partyName(id: string | null, order?: Order) {
  if (id === null) return 'Нет компании';
  if (id === order?.supplier.id) return `${order.supplier.name} (${id})`;
  if (id === order?.buyer.id) return `${order.buyer.name} (${id})`;
  return id;
}

/** Values come solely from the current inventory DTO, not agreement party IDs. */
export function CurrentVehicleFacts({ vehicleId, order }: { vehicleId: string; order?: Order }) {
  const session = useSession();
  const companyId = session.company?.id ?? '';
  const contextRevision = session.view.context.revision;
  const detail = useData(
    ['vehicle', companyId, contextRevision, vehicleId],
    () => get<{ vehicle: Vehicle }>(`/inventory/vehicle-units/${encodeURIComponent(vehicleId)}`),
    !!companyId && !!vehicleId,
  );
  if (detail.isLoading) return <span>Собственник и хранитель: загрузка…</span>;
  if (detail.error || !detail.data?.data?.vehicle)
    return (
      <span>
        Текущие собственник и хранитель недоступны. <Button onClick={() => void detail.refetch()}>Повторить</Button>
      </span>
    );
  const vehicle = detail.data.data.vehicle;
  return (
    <span>
      Собственник: {partyName(vehicle.ownerCompanyId, order)} · Хранитель:{' '}
      {partyName(vehicle.custodianCompanyId, order)}
    </span>
  );
}

type Edit =
  { kind: 'request'; unit: ConsignmentUnit } | { kind: 'confirm' | 'reject'; request: ConsignmentReturn } | null;

type ConsignmentWorkspaceProps = {
  order: Order;
  invoices: Invoice[];
  invoiceLoading?: boolean;
  invoiceError?: unknown;
  onRetryInvoices?: () => void;
};

function useConsignmentWorkspace(order: Order) {
  const session = useSession();
  const companyId = session.company?.id ?? '';
  const contextRevision = session.view.context.revision;
  const consigned = order.source === 'reservation' && order.terms.reservation?.mode === 'consignment';
  const agreementKey = ['consignment', companyId, contextRevision, order.id];
  const returnsKey = ['consignment-returns', companyId, contextRevision, order.id];
  const agreement = useData(
    agreementKey,
    () => get<Consignment>(`/commerce/consignments/orders/${encodeURIComponent(order.id)}`),
    !!companyId && consigned,
  );
  const returns = useData(returnsKey, () => loadConsignmentReturnHistory(order.id), !!companyId && consigned);
  const warehouses = useData(
    ['warehouses', companyId, contextRevision],
    () => list<Warehouse>('/inventory/warehouses'),
    !!companyId && consigned,
  );
  const refresh = useRefresh();
  const [edit, setEdit] = useState<Edit>(null);
  const [warehouseId, setWarehouseId] = useState('');
  const [reason, setReason] = useState('');
  const [movedAt, setMovedAt] = useState('');
  const [busy, setBusy] = useState(false);
  const [blocked, setBlocked] = useState(false);
  const [message, setMessage] = useState('');
  const [failure, setFailure] = useState('');
  const sending = useRef(false);
  const current = agreement.data?.data;
  const canTrade = session.can('commerce.trade');
  const isSupplier = current?.supplierCompanyId === companyId;
  const isDealer = current?.dealerCompanyId === companyId;

  async function refreshAuthority() {
    // A failed page or read keeps actions locked. No partially fetched history is published.
    await Promise.all([
      session.refresh(),
      get<Order>(`/commerce/orders/${encodeURIComponent(order.id)}`),
      list<Invoice>(`/commerce/orders/${encodeURIComponent(order.id)}/invoices`),
      get<Consignment>(`/commerce/consignments/orders/${encodeURIComponent(order.id)}`),
      loadConsignmentReturnHistory(order.id),
      list<Warehouse>('/inventory/warehouses'),
    ]);
    await refresh(
      ['order', order.id],
      ['orders'],
      ['order-invoices', order.id],
      agreementKey,
      returnsKey,
      ['vehicle'],
      ['vehicles'],
      ['warehouses'],
      ['stock'],
      ['eligible-vehicles'],
      ['consignment-candidates'],
    );
  }

  async function retryRefresh() {
    if (sending.current) return;
    sending.current = true;
    setBusy(true);
    setFailure('');
    try {
      await refreshAuthority();
      setBlocked(false);
    } catch (error) {
      setBlocked(true);
      setFailure(`Не удалось обновить данные: ${errorText(error)}. Повторите загрузку.`);
    } finally {
      sending.current = false;
      setBusy(false);
    }
  }

  function start(next: Edit) {
    setEdit(next);
    setWarehouseId('');
    setReason('');
    setMovedAt('');
    setFailure('');
  }

  async function submit() {
    if (!edit || sending.current || blocked || !canTrade || !current) return;
    const validation = returnEditError(edit, current, returns.data, {
      isSupplier,
      isDealer,
      warehouseId,
      reason,
      movedAt,
      warehouseError: warehouses.error,
      warehouses: warehouses.data,
    });
    if (validation) {
      setFailure(validation);
      return;
    }
    sending.current = true;
    setBusy(true);
    setFailure('');
    setMessage('');
    try {
      await sendReturnEdit(edit, warehouseId, reason, movedAt);
      setEdit(null);
      setMessage(
        edit.kind === 'request'
          ? 'Запрос записан. Автомобиль остаётся у дилера до подтверждения.'
          : edit.kind === 'confirm'
            ? 'Фактический возврат подтверждён.'
            : 'Отказ записан. Автомобиль остаётся у дилера.',
      );
      setBlocked(true);
      try {
        await refreshAuthority();
        setBlocked(false);
      } catch (error) {
        setFailure(
          `Действие выполнено, но данные не обновились: ${errorText(error)}. Повторите загрузку; не отправляйте действие снова.`,
        );
      }
    } catch (error) {
      setFailure(errorText(error));
      if (!(error instanceof ApiError) || [0, 409, 412].includes(error.status)) {
        setBlocked(true);
        try {
          await refreshAuthority();
          setBlocked(false);
        } catch (readError) {
          setFailure(`${errorText(error)} Данные не обновились: ${errorText(readError)}. Повторите загрузку.`);
        }
      }
    } finally {
      sending.current = false;
      setBusy(false);
    }
  }

  const controlsDisabled =
    busy || blocked || agreement.isLoading || returns.isLoading || !!agreement.error || !!returns.error;
  return {
    consigned,
    agreement,
    returns,
    warehouses,
    edit,
    warehouseId,
    setWarehouseId,
    reason,
    setReason,
    movedAt,
    setMovedAt,
    busy,
    blocked,
    message,
    failure,
    current,
    canTrade,
    isSupplier,
    isDealer,
    retryRefresh,
    start,
    submit,
    controlsDisabled,
  };
}

export function ConsignmentWorkspace(props: ConsignmentWorkspaceProps) {
  const { order } = props;
  const state = useConsignmentWorkspace(order);
  if (order.source !== 'reservation') return null;
  return (
    <div className="kit-stack">
      <AcceptedHandover order={order} />
      <ConsignmentInvoices {...props} />
      {state.consigned && <ConsignmentReturns state={state} order={order} />}
    </div>
  );
}

type ConsignmentState = ReturnType<typeof useConsignmentWorkspace>;

function ConsignmentInvoices({
  order,
  invoices,
  invoiceLoading = false,
  invoiceError,
  onRetryInvoices,
}: ConsignmentWorkspaceProps) {
  return (
    <Panel title="Обязательства по заказу" padded>
      <p>Стоимость по принятому заказу: {money(order.total)}. Возврат автомобиля не отменяет расчёты по заказу.</p>
      {invoiceLoading && <p role="status">Загрузка счетов…</p>}
      {!!invoiceError && (
        <Notice kind="danger">
          Не удалось загрузить счета: {errorText(invoiceError)}{' '}
          {onRetryInvoices && <Button onClick={onRetryInvoices}>Повторить</Button>}
        </Notice>
      )}
      {!invoiceLoading && !invoiceError && !invoices.length && <p>Выставленного счёта пока нет.</p>}
      {!invoiceLoading &&
        !invoiceError &&
        invoices.map((invoice) => (
          <p key={invoice.id}>
            Счёт {invoice.id}: {money(invoice.total)} · оплачено {money(invoice.paid)} · к оплате{' '}
            {money(invoice.outstanding)}. Принятый аванс уже включён в «оплачено».
          </p>
        ))}
    </Panel>
  );
}

function ConsignmentReturns({ state, order }: { state: ConsignmentState; order: Order }) {
  const { agreement, returns, blocked, busy, message, failure, current, retryRefresh } = state;
  return (
    <Panel title="Консигнация и возвраты" padded>
      {agreement.isLoading && <p role="status">Загрузка соглашения…</p>}
      {agreement.error && (
        <Notice kind="danger">
          Не удалось загрузить соглашение: {errorText(agreement.error)}{' '}
          <Button onClick={() => void agreement.refetch()}>Повторить</Button>
        </Notice>
      )}
      {returns.isLoading && <p role="status">Загрузка истории возвратов…</p>}
      {returns.error && (
        <Notice kind="danger">
          Не удалось загрузить полную историю возвратов: {errorText(returns.error)}{' '}
          <Button onClick={() => void returns.refetch()}>Повторить</Button>
        </Notice>
      )}
      {blocked && (
        <Notice kind="warning">
          Действия остановлены до обновления заказа, соглашения, истории и складов.{' '}
          <Button disabled={busy} onClick={() => void retryRefresh()}>
            Повторить загрузку
          </Button>
        </Notice>
      )}
      {message && <Notice kind="success">{message}</Notice>}
      {failure && <Notice kind="danger">{failure}</Notice>}
      {current && (
        <>
          <p>
            Соглашение по заказу {current.orderId}. Стороны соглашения: {order.supplier.name} и {order.buyer.name}.
            Текущие собственник и хранитель показаны отдельно по данным склада.
          </p>
          {!current.units.length && <p>VIN ещё не назначены. Соглашение действует для будущей поставки.</p>}
          <ConsignmentUnits state={state} current={current} order={order} />
          <ReturnHistory state={state} current={current} />
          <ReturnEditForm state={state} />
        </>
      )}
    </Panel>
  );
}

function returnableUnit(unit: ConsignmentUnit) {
  return unit.status === 'held' && !!unit.handedOverAt && !unit.currentRetailDealId && !unit.pendingReturnId;
}

function returnVin(current: Consignment, request: ConsignmentReturn) {
  return current.units.find((unit) => unit.id === request.unitId)?.vin ?? request.vehicleId;
}

function ConsignmentUnits({ state, current, order }: { state: ConsignmentState; current: Consignment; order: Order }) {
  const { isSupplier, canTrade, controlsDisabled, start } = state;
  return (
    <>
      {current.units.map((unit) => {
        const requestable = isSupplier && canTrade && returnableUnit(unit);
        return (
          <section key={unit.id}>
            <h3>VIN {unit.vin}</h3>
            <p>
              {unitName[unit.status]}
              {unit.handedOverAt ? ` · передан ${dateTime(unit.handedOverAt)}` : ''}
            </p>
            <p>
              <CurrentVehicleFacts vehicleId={unit.vehicleId} order={order} />
            </p>
            {unit.currentRetailDealId && <p>Автомобиль связан с розничной сделкой; возврат недоступен.</p>}
            {unit.pendingReturnId && <p>Запрос возврата уже ожидает решения дилера.</p>}
            {unit.status === 'sold' && <p>Проданный автомобиль нельзя вернуть по консигнации.</p>}
            {requestable && (
              <Button disabled={controlsDisabled} onClick={() => start({ kind: 'request', unit })}>
                Запросить возврат VIN {unit.vin}
              </Button>
            )}
          </section>
        );
      })}
    </>
  );
}

function ReturnHistory({ state, current }: { state: ConsignmentState; current: Consignment }) {
  const { returns, warehouses, isDealer, canTrade, controlsDisabled, start } = state;
  const canDecide = isDealer && canTrade;
  return (
    <>
      <h3>История запросов и решений</h3>
      {!returns.isLoading && !returns.error && !returns.data?.length && <p>Запросов возврата пока нет.</p>}
      {!returns.error &&
        returns.data?.map((request) => (
          <section key={request.id}>
            <p>
              VIN {returnVin(current, request)} · {returnName[request.status]} · запрошен{' '}
              {dateTime(request.requestedAt)}
            </p>
            <p>
              Склад назначения:{' '}
              {warehouses.data?.find((w) => w.id === request.destinationWarehouseId)?.name ??
                request.destinationWarehouseId}
              . Причина: {request.reason}
            </p>
            {request.decidedAt && (
              <p>
                Решение: {dateTime(request.decidedAt)}
                {request.decisionReason ? ` · причина: ${request.decisionReason}` : ''}
                {request.movedAt ? ` · движение: ${dateTime(request.movedAt)}` : ''}
              </p>
            )}
            {canDecide && request.allowedActions.includes('confirm') && (
              <Button disabled={controlsDisabled} onClick={() => start({ kind: 'confirm', request })}>
                Подтвердить возврат VIN {returnVin(current, request)}
              </Button>
            )}
            {canDecide && request.allowedActions.includes('reject') && (
              <Button disabled={controlsDisabled} onClick={() => start({ kind: 'reject', request })}>
                Отклонить возврат VIN {returnVin(current, request)}
              </Button>
            )}
          </section>
        ))}
    </>
  );
}

function ReturnEditForm({ state }: { state: ConsignmentState }) {
  const {
    edit,
    warehouses,
    warehouseId,
    setWarehouseId,
    busy,
    movedAt,
    setMovedAt,
    reason,
    setReason,
    blocked,
    start,
    submit,
  } = state;
  return (
    <>
      {edit && (
        <section>
          <h3>
            {edit.kind === 'request'
              ? `Запрос возврата VIN ${edit.unit.vin}`
              : edit.kind === 'confirm'
                ? 'Подтвердить фактическое движение'
                : 'Отклонить запрос'}
          </h3>
          {edit.kind === 'request' && (
            <>
              <p>Запрос не перемещает автомобиль. Дилер подтвердит фактическое движение или отклонит запрос.</p>
              {warehouses.error && (
                <Notice kind="danger">
                  Не удалось загрузить ваши склады: {errorText(warehouses.error)}{' '}
                  <Button onClick={() => void warehouses.refetch()}>Повторить</Button>
                </Notice>
              )}
              <label>
                Склад возврата
                <select
                  value={warehouseId}
                  onChange={(event) => setWarehouseId(event.target.value)}
                  disabled={busy || !!warehouses.error}
                >
                  <option value="">Выберите свой склад</option>
                  {warehouses.data?.map((w) => (
                    <option key={w.id} value={w.id}>
                      {w.name} · свободно {w.free}
                    </option>
                  ))}
                </select>
              </label>
            </>
          )}
          {edit.kind === 'confirm' && (
            <label>
              Фактическое время движения
              <input
                type="datetime-local"
                value={movedAt}
                onChange={(event) => setMovedAt(event.target.value)}
                disabled={busy}
              />
            </label>
          )}
          {edit.kind !== 'confirm' && (
            <label>
              {edit.kind === 'request' ? 'Причина возврата' : 'Причина отказа'}
              <textarea value={reason} onChange={(event) => setReason(event.target.value)} disabled={busy} />
            </label>
          )}
          <div className="kit-row">
            <Button disabled={busy} onClick={() => start(null)}>
              Отмена
            </Button>
            <Button variant="primary" disabled={busy || blocked} onClick={() => void submit()}>
              {busy
                ? 'Отправка…'
                : edit.kind === 'request'
                  ? 'Отправить запрос'
                  : edit.kind === 'confirm'
                    ? 'Подтвердить движение'
                    : 'Отклонить запрос'}
            </Button>
          </div>
        </section>
      )}
    </>
  );
}

function latestReturnEditError(
  edit: NonNullable<Edit>,
  current: Consignment,
  returns: ConsignmentReturn[] | undefined,
): string | null {
  if (edit.kind === 'request') {
    const latest = current.units.find((unit) => unit.id === edit.unit.id);
    if (!latest || latest.revision !== edit.unit.revision || !returnableUnit(latest))
      return 'VIN больше не доступен для возврата. Обновите соглашение.';
  } else {
    const latest = returns?.find((request) => request.id === edit.request.id);
    if (
      !latest ||
      latest.revision !== edit.request.revision ||
      latest.status !== 'pending' ||
      !latest.allowedActions.includes(edit.kind)
    )
      return 'Запрос изменился. Обновите историю возвратов.';
  }
  return null;
}

type ReturnDraft = {
  isSupplier: boolean;
  isDealer: boolean;
  warehouseId: string;
  reason: string;
  movedAt: string;
  warehouseError: unknown;
  warehouses: Warehouse[] | undefined;
};

function returnEditError(
  edit: NonNullable<Edit>,
  current: Consignment,
  returns: ConsignmentReturn[] | undefined,
  draft: ReturnDraft,
): string | null {
  const latestError = latestReturnEditError(edit, current, returns);
  if (latestError) return latestError;
  if (
    edit.kind === 'request' &&
    (!draft.isSupplier ||
      !draft.warehouseId ||
      !draft.reason.trim() ||
      draft.warehouseError ||
      !draft.warehouses?.some((w) => w.id === draft.warehouseId))
  )
    return 'Выберите свой склад назначения и укажите причину возврата.';
  if (
    edit.kind === 'confirm' &&
    (!draft.isDealer ||
      !edit.request.allowedActions.includes('confirm') ||
      !draft.movedAt ||
      Number.isNaN(Date.parse(draft.movedAt)))
  )
    return 'Укажите фактическое время движения автомобиля.';
  if (
    edit.kind === 'reject' &&
    (!draft.isDealer || !edit.request.allowedActions.includes('reject') || !draft.reason.trim())
  )
    return 'Укажите причину отказа.';
  return null;
}

async function sendReturnEdit(edit: NonNullable<Edit>, warehouseId: string, reason: string, movedAt: string) {
  if (edit.kind === 'request') {
    await post(
      `/commerce/consignments/vehicles/${encodeURIComponent(edit.unit.vehicleId)}/return-requests`,
      { destinationWarehouseId: warehouseId, reason: reason.trim() },
      { ifMatch: edit.unit.revision },
    );
  } else if (edit.kind === 'confirm') {
    await post(
      `/commerce/consignments/return-requests/${encodeURIComponent(edit.request.id)}/confirm`,
      { movedAt: new Date(movedAt).toISOString() },
      { ifMatch: edit.request.revision },
    );
  } else {
    await post(
      `/commerce/consignments/return-requests/${encodeURIComponent(edit.request.id)}/reject`,
      { reason: reason.trim() },
      { ifMatch: edit.request.revision },
    );
  }
}
