import { useState } from 'react';
import {
  ActionButton,
  Badge,
  FormDialog,
  Tabs,
  errorText,
  Button,
  Details,
  Modal,
  Notice,
  Panel,
  Table,
  dateTime,
  fileUrl,
  get,
  list,
  money,
  post,
  useData,
  useRefresh,
} from '@justixauto/kit';
import type { FieldSpec } from '@justixauto/kit';
import { TermsButton, TermsDialog, TermsView } from '../shared';
import { OrderCreate } from '../order-create';
import { OrderFulfilmentPanel } from '../order-fulfilment';
import type { OrderFulfilmentAction } from '../order-fulfilment';
import {
  modelName,
  useOrderModels,
  routeLabel,
  useModelName,
  useWarehouses,
  warehouseOptions,
  orderLabel,
  orderSourceLabel,
  orderTone,
} from '../data';
import type { Invoice, Offer, Order, Shipment } from '../data';

import { OrderActionButton, OrderActionContext } from '../order-actions';
import type { OrderAction } from '../order-actions';
import { nextOrderAction, orderLineLabel, shortId } from '../order-workspace';
import { vehicleColorsLabel } from '../vehicle-colors';
import { ReceiptIdentification } from '../receipt-identification';
import { SaleForm } from '../sale-form';
import type { SaleSourceSeed } from '../sale-offer-source';

const reason: FieldSpec[] = [{ name: 'reason', label: 'Причина', type: 'textarea', required: true }];

// ---------------- partners ----------------

// ---------------- offers ----------------

export function OfferDialog({ id, onClose, onStartOrder }: { id: string; onClose: () => void; onStartOrder?: (offer: Offer) => void }) {
  const q = useData(['offer', id], () => get<Offer>(`/commerce/offers/${id}`));
  const name = useModelName();
  const warehouses = useWarehouses();
  const o = q.data?.data;
  const refresh = [['offer', id], ['offers']];
  const own = !!o?.versions;
  const latest = o?.versions?.[o.versions.length - 1];
  const shown = own ? latest : o?.publishedVersion;
  const [creating, setCreating] = useState(false);
  const [createdOrder, setCreatedOrder] = useState<string | null>(null);
  const [saleSeed, setSaleSeed] = useState<SaleSourceSeed | null>(null);
  if (createdOrder) return <><Notice kind="success">Заказ создан и отправлен поставщику на подтверждение.</Notice><OrderDialog id={createdOrder} onClose={onClose} /></>;
  if (creating && o) return <OrderCreate mode={{ kind: 'offer', offer: o }} onClose={() => setCreating(false)} onCreated={setCreatedOrder} />;
  if (saleSeed) return <SaleForm onClose={() => setSaleSeed(null)} sourceSeed={saleSeed} />;
  return (
    <Modal
      title={o ? `Предложение ${o.supplier.name}` : 'Предложение'}
      onClose={onClose}
      size="wide"
      footer={
        o &&
        (own
          ? o.status !== 'withdrawn' && (
              <>
                <TermsButton
                  label="Новая версия"
                  initial={latest?.terms}
                  refresh={refresh}
                  onSubmit={async (terms) => {
                    await post(
                      `/commerce/offers/${id}/versions`,
                      { terms, audience: latest?.audience ?? { mode: 'all-active', partnerCompanyIds: [] } },
                      { ifMatch: o.revision },
                    );
                  }}
                />
                {latest && (
                  <ActionButton
                    label={`Опубликовать №${latest.number}`}
                    variant="primary"
                    refresh={refresh}
                    onSubmit={() =>
                      post(`/commerce/offers/${id}/publish`, { offerVersionId: latest.id }, { ifMatch: o.revision })
                    }
                  />
                )}
                {latest && !q.error && <Button disabled={q.isFetching} onClick={() => setSaleSeed({ kind: 'own-offer', id, versionId: latest.id })}>Создать продажу</Button>}
                <ActionButton
                  label="Снять"
                  variant="danger"
                  fields={reason}
                  refresh={refresh}
                  onSubmit={(v) => post(`/commerce/offers/${id}/withdraw`, v, { ifMatch: o.revision })}
                />
              </>
            )
          : o.publishedVersion && (
              <>
                <Button variant="primary" onClick={() => onStartOrder ? onStartOrder(o) : setCreating(true)}>Заказать</Button>
                {o.status !== 'withdrawn' && !q.error && <Button disabled={q.isFetching} onClick={() => setSaleSeed({ kind: 'supplier-offer', id, versionId: o.publishedVersion!.id })}>Создать продажу</Button>}
              </>
            ))
      }
    >
      {shown && (
        <>
          <p>
            <b>Версия №{shown.number}</b> · итого {money(shown.total)}
            {own && o?.publishedVersion ? ` · опубликована №${o.publishedVersion.number}` : ''}
          </p>
          <TermsView terms={shown.terms} modelNameOf={name} />
        </>
      )}
    </Modal>
  );
}

// ---------------- purchases: orders ----------------

const eventLabel: Record<string, string> = {
  'order.created': 'Заказ создан',
  'order.confirmed': 'Подтверждён поставщиком',
  'order.receiving_warehouse_set': 'Покупатель выбрал склад получения',
  'order.rejected': 'Отклонён поставщиком',
  'order.cancelled': 'Отменён',
  'order.vehicles_allocated': 'Назначены VIN',
  'order.shipped': 'Отгрузка',
  'order.shipped_delivered': 'Отгружено и принято на склад покупателя',
  'order.addendum_proposed': 'Предложено изменение условий',
  'order.addendum_accepted': 'Изменение условий принято',
  'order.addendum_rejected': 'Изменение условий отклонено',
  'order.invoice_issued': 'Выставлен счёт',
  'order.invoice_voided': 'Счёт аннулирован',
  'order.payment_submitted': 'Сообщено об оплате',
  'order.payment_accepted': 'Оплата принята',
  'order.payment_rejected': 'Оплата отклонена',
  'shipment.receipt_accepted': 'Принято на склад',
  'shipment.receipt_rejected': 'Отказ в приёмке',
};

function OrderDialogFooter({
  o,
  id,
  refresh,
  invoices,
  openAddendum,
  name,
  onFulfilment,
  onTerms,
  invoicesReady,
}: {
  o: Order;
  id: string;
  refresh: unknown[][];
  invoices: Invoice[];
  openAddendum: Order['addenda'][number] | undefined;
  name: (modelId: string) => string;
  onFulfilment: (action: OrderFulfilmentAction) => void;
  onTerms: () => void;
  invoicesReady: boolean;
}) {
  const actions = o.allowedActions;
  const warehouses = useWarehouses();
  return (
    <>
      {actions.includes('confirm') && (
        <OrderActionButton
          label="Подтвердить"
          variant="primary"
          refresh={refresh}
          onSubmit={() => post(`/commerce/orders/${id}/supplier-confirmations`, {}, { ifMatch: o.revision })}
        />
      )}
      {actions.includes('reject') && (
        <OrderActionButton
          label="Отклонить"
          fields={reason}
          refresh={refresh}
          onSubmit={(v) => post(`/commerce/orders/${id}/supplier-rejections`, v, { ifMatch: o.revision })}
        />
      )}
      {actions.includes('set-warehouse') && (
        <OrderActionButton
          label={o.receivingWarehouseId ? 'Изменить склад получения' : 'Выбрать склад получения'}
          {...(o.receivingWarehouseId ? {} : { variant: 'primary' as const })}
          refresh={refresh}
          intro={<p>На этот склад автомобили поступят сразу после отгрузки поставщиком.</p>}
          fields={[
            {
              name: 'warehouseId',
              label: 'Склад получения',
              type: 'select',
              required: true,
              options: warehouseOptions(warehouses.data),
              ...(o.receivingWarehouseId ? { initial: o.receivingWarehouseId } : {}),
            },
          ]}
          onSubmit={(v) => post(`/commerce/orders/${id}/receiving-warehouse`, v, { ifMatch: o.revision })}
        />
      )}
      {actions.includes('ship-quantity') && <Button variant="primary" onClick={() => onFulfilment({ kind: 'ship-quantity' })}>Отгрузить по количеству</Button>}
      {actions.includes('allocate') && o.terms.lines.map((line) => (
        <Button key={line.lineId} onClick={() => onFulfilment({ kind: 'allocate', lineId: line.lineId! })}>Назначить VIN — {orderLineLabel(line, name)}</Button>
      ))}
      {actions.includes('ship') && <Button onClick={() => onFulfilment({ kind: 'ship-allocated' })}>Отгрузить назначенные VIN</Button>}
      {actions.includes('propose-addendum') && <Button onClick={onTerms}>Предложить изменение</Button>}
      {actions.includes('accept-addendum') && openAddendum?.proposedBy !== o.party && openAddendum && (
        <OrderActionButton
          label={`Принять изменение №${openAddendum.number}`}
          variant="primary"
          refresh={refresh}
          onSubmit={() => post(`/commerce/orders/${id}/addenda/${openAddendum.id}/accept`, {}, { ifMatch: o.revision })}
        />
      )}
      {actions.includes('reject-addendum') && openAddendum?.proposedBy !== o.party && openAddendum && (
        <OrderActionButton
          label="Отклонить изменение"
          fields={reason}
          refresh={refresh}
          onSubmit={(v) => post(`/commerce/orders/${id}/addenda/${openAddendum.id}/reject`, v, { ifMatch: o.revision })}
        />
      )}
      {invoicesReady && o.party === 'supplier' &&
        ['accepted', 'fulfilling'].includes(o.status) &&
        !invoices.some((i) => i.status === 'issued') && (
          <OrderActionButton
            label="Выставить счёт"
            refresh={refresh}
            fields={
              o.terms.paymentSchedule.length
                ? []
                : [{ name: 'dueDate', label: 'Срок оплаты', type: 'date', required: true }]
            }
            onSubmit={(v) => post(`/commerce/orders/${id}/invoices`, v, { ifMatch: o.revision })}
          />
        )}
      {actions.includes('cancel') && (
        <OrderActionButton
          label="Отменить заказ"
          variant="danger"
          fields={reason}
          refresh={refresh}
          onSubmit={(v) => post(`/commerce/orders/${id}/cancellations`, v, { ifMatch: o.revision })}
        />
      )}
    </>
  );
}


type OrderTab = 'cars' | 'payments' | 'changes' | 'history';

function OrderCars({ o, name, setShipment, setBatch }: {
  o: Order;
  name: (id: string) => string;
  setShipment: (id: string) => void;
  setBatch: (batch: NonNullable<Order['receiptBatches']>[number]) => void;
}) {
  const warehouses = useWarehouses();
  return <div className="kit-stack">
    {o.party === 'buyer' && o.receivingWarehouseId && <p>Склад получения: {warehouses.data?.find((w) => w.id === o.receivingWarehouseId)?.name ?? 'Загрузка…'}</p>}
    {o.hasReceivingWarehouse === false && ['accepted', 'fulfilling'].includes(o.status) && <Notice kind="warning">
      {o.party === 'supplier'
        ? 'Для отгрузки по количеству покупатель ещё не выбрал склад получения. Покупателю нужно открыть заказ и выбрать склад. Прежняя отгрузка назначенных VIN остаётся доступна, если разрешена в заказе.'
        : 'Выберите склад получения в действиях заказа для автоматического поступления. Для прежних отгрузок доступна ручная приёмка.'}
    </Notice>}
    <Panel title="Автомобили и исполнение">
      <div className="order-lines"><Table rows={o.terms.lines} rowKey={(l) => l.lineId!} columns={[
        { title: 'Модель и заказанные цвета', render: (l) => <><div className="cell-main">{name(l.modelId)}</div><div className="cell-sub">{vehicleColorsLabel(l)}</div></> },
        { title: 'Заказано', render: (l) => l.quantity },
        { title: 'Назначено к отгрузке', render: (l) => o.lineProgress?.find((p) => p.orderLineId === l.lineId)?.allocated ?? '0' },
        { title: 'Отгружено', render: (l) => `${o.lineProgress?.find((p) => p.orderLineId === l.lineId)?.shipped ?? '0'} из ${l.quantity}` },
        { title: 'Поступило с VIN', render: (l) => o.lineProgress?.find((p) => p.orderLineId === l.lineId)?.identified ?? '0' },
        { title: 'Поступило без VIN', render: (l) => o.lineProgress?.find((p) => p.orderLineId === l.lineId)?.unidentified ?? '0' },
        { title: 'Осталось отгрузить', render: (l) => String(Number(l.quantity) - Number(o.lineProgress?.find((p) => p.orderLineId === l.lineId)?.shipped ?? 0)) },
      ]} /></div>
      {o.lineProgress?.some((p) => p.receiptQuantityAdjusted) && <Notice kind="warning">Количество поступившей партии исправлено на складе. Исторически отгруженное количество сохранено; текущие количества с VIN и без VIN могут отличаться от него.</Notice>}
    </Panel>
    {o.party === 'buyer' && <Panel title="Поступившие партии" padded>
      {!o.receiptBatches?.length && <p>Партий поступления по количеству пока нет. Прежние отгрузки доступны ниже.</p>}
      {o.terms.lines.map((line) => {
        const batches = o.receiptBatches?.filter((b) => b.orderLineId === line.lineId) ?? [];
        return batches.length > 0 && <section key={line.lineId}>
          <h3>{orderLineLabel(line, name)}</h3>
          {[...new Set(batches.map((b) => b.shipmentId))].map((shipmentId) => <section key={shipmentId}>
            <h4 title={shipmentId}>Отгрузка {shortId(shipmentId)}</h4>
            {batches.filter((b) => b.shipmentId === shipmentId).map((b) => <div className="order-workspace-batch" key={b.receiptBatchId}>
              <p><span title={b.receiptBatchId}>Партия {shortId(b.receiptBatchId)}</span> · отгружено {b.shippedQuantity}, принято сейчас {b.confirmedQuantity} · с VIN {b.identifiedCount}, без VIN {b.unidentifiedCount}</p>
              <p>Фактические цвета партии · {vehicleColorsLabel(b)}</p>
              <Button onClick={() => setBatch(b)}>{Number(b.unidentifiedCount) > 0 ? 'Ввести VIN' : 'VIN заполнены'}</Button>
            </div>)}
          </section>)}
        </section>;
      })}
    </Panel>}
    <Panel title="Назначенные VIN">
      <Table rows={o.allocations} rowKey={(a) => a.vehicleId + a.status} empty="VIN ещё не назначены" columns={[
        { title: 'VIN', render: (a) => <code>{a.vin}</code> },
        { title: 'Статус', render: (a) => ({ allocated: 'Назначен', shipped: 'Отгружен', delivered: 'Принят', rejected: 'Отклонён при приёмке', released: 'Освобождён' })[a.status] ?? a.status },
      ]} />
    </Panel>
    <Panel title="Отгрузки">
      <Table rows={o.shipments} rowKey={(s) => s.id} empty="Отгрузок пока нет" columns={[
        { title: 'Отгрузка', render: (s) => <Button variant="link" title={s.id} onClick={() => setShipment(s.id)}>Отгрузка {shortId(s.id)}</Button> },
        { title: 'Маршрут', render: (s) => routeLabel[s.route] },
        { title: 'Статус', render: (s) => <Badge tone={s.status === 'received' ? 'success' : 'info'}>{s.status === 'received' ? 'Передача завершена' : 'В пути'}</Badge> },
      ]} />
    </Panel>
    <details className="kit-details"><summary>Коммерческие условия</summary><TermsView terms={o.terms} modelNameOf={name} /></details>
  </div>;
}

export function OrderDialog({ id, onClose }: { id: string; onClose: () => void }) {
  const q = useData(['order', id], () => get<Order>(`/commerce/orders/${id}`));
  const invoicesQuery = useData(['order-invoices', id], () => list<Invoice>(`/commerce/orders/${id}/invoices`));
  const models = useOrderModels();
  const warehouses = useWarehouses();
  const name = (modelId: string) => modelName(models.data?.find((m) => m.id === modelId));
  const o = q.data?.data;
  const refresh = [['order', id], ['orders'], ['order-invoices', id], ['vehicles']];
  const reload = useRefresh();
  const [tab, setTab] = useState<OrderTab>('cars');
  const [shipment, setShipment] = useState<string | null>(null);
  const [batch, setBatch] = useState<NonNullable<Order['receiptBatches']>[number] | null>(null);
  const [fulfilment, setFulfilment] = useState<OrderFulfilmentAction | null>(null);
  const [operation, setOperation] = useState<OrderAction | null>(null);
  const [terms, setTerms] = useState(false);
  const [success, setSuccess] = useState('');
  const openAction = (action: OrderAction) => { setSuccess(''); setOperation(action); };
  const done = (message: string) => { setSuccess(message); setFulfilment(null); setBatch(null); };
  async function refreshAfter(message: string, keys = refresh) {
    try { await reload(...keys); setSuccess(message); }
    catch { setSuccess(message + ' Не удалось обновить данные. Повторите загрузку заказа.'); }
  }

  // Replacing the outer modal preserves the order/tab/shipment state without stacking dialogs.
  const needsWarehouse = operation?.fields?.some((field) => field.name === 'warehouseId');
  const operationFields = operation?.fields?.map((field) => field.name === 'warehouseId'
    ? { ...field, options: warehouseOptions(warehouses.data) } : field) ?? [];
  if (operation) return <FormDialog title={operation.title ?? operation.label} fields={operationFields}
    submitLabel={operation.submitLabel ?? operation.label} size={operation.size}
    intro={<><Button variant="back" onClick={() => setOperation(null)}>Назад</Button>{operation.intro}
      {needsWarehouse && warehouses.isLoading && <p role="status">Загрузка складов…</p>}
      {needsWarehouse && warehouses.error && <Notice kind="danger">Не удалось загрузить склады. <Button onClick={() => void warehouses.refetch()}>Повторить загрузку складов</Button></Notice>}
      {needsWarehouse && !warehouses.isLoading && !warehouses.error && !warehouses.data?.length && <Notice>Нет доступных складов. Добавьте склад в разделе «Склады».</Notice>}
    </>}
    onClose={() => setOperation(null)} onSubmit={async (values) => {
      await operation.onSubmit(values);
      await refreshAfter(`${operation.label}: выполнено.`, operation.refresh ?? refresh);
    }} />;
  if (terms && o) return <TermsDialog title="Предложить изменение" initial={o.terms}
    intro={<Button variant="back" onClick={() => setTerms(false)}>Назад</Button>}
    extra={[{ name: 'reason', label: 'Причина изменения', required: true }]}
    onClose={() => setTerms(false)} onSubmit={async (newTerms, extra) => {
      await post(`/commerce/orders/${id}/addenda`, { terms: newTerms, reason: extra.reason }, { ifMatch: o.revision });
      await refreshAfter('Изменение предложено. Ждём решения контрагента.');
    }} />;

  return <OrderActionContext.Provider value={openAction}>
    <div className="order-dialog"><Modal title={o ? `Заказ ${shortId(id)} · ${o.party === 'buyer' ? 'Покупка у' : 'Продажа'} ${o.party === 'buyer' ? o.supplier.name : o.buyer.name}` : 'Заказ'}
      onClose={fulfilment ? () => setFulfilment(null) : batch ? () => setBatch(null) : shipment ? () => setShipment(null) : onClose} size="wide"
      footer={!fulfilment && !batch && !shipment && <>
        <Button onClick={onClose}>Закрыть</Button>
        {/* A finished order is read-only: the footer only closes the dialog. */}
        {o && !['completed', 'cancelled', 'rejected'].includes(o.status) && <OrderDialogFooter o={{ ...o, allowedActions: o.allowedActions.filter((a) => !['confirm', 'allocate', 'ship', 'ship-quantity'].includes(a)) }}
          id={id} refresh={refresh} invoices={invoicesQuery.data ?? []}
          openAddendum={o.addenda.find((a) => a.status === 'proposed')} name={name}
          onFulfilment={setFulfilment} onTerms={() => setTerms(true)} invoicesReady={!invoicesQuery.isLoading && !invoicesQuery.error} />}
      </>}>
      <div className="order-workspace kit-stack">
        {q.error && <><Notice kind="danger">Не удалось загрузить заказ. {errorText(q.error)}</Notice><Button onClick={() => void q.refetch()}>Повторить загрузку заказа</Button></>}
        {!o && !q.error && <p role="status">Загрузка заказа…</p>}
        {o && <>
          <header className="order-summary">
            <div className="order-summary-top"><Badge tone={orderTone(o.status)}>{orderLabel[o.status] ?? o.status}</Badge>
              <b className="order-summary-total">{money(o.total)}</b>
              <small>{orderSourceLabel[o.source] ?? o.source} · <span title={id}>№ {id}</span></small></div>
            <p>{nextOrderAction(o).message}</p>
            {o.statusReason && <p>{o.statusReason}</p>}
          </header>
          {success && <Notice kind="success">{success}</Notice>}
          {fulfilment ? <OrderFulfilmentPanel order={o} action={fulfilment} refresh={refresh} onBack={() => setFulfilment(null)} onDone={done} />
            : batch ? <ReceiptIdentification key={batch.receiptBatchId}
              batch={{ ...batch, id: batch.receiptBatchId }} modelName={name(batch.modelId)}
              refresh={refresh} onBack={() => setBatch(null)} onDone={done} />
            : shipment ? <ShipmentPanel id={shipment} party={o.party} onClose={() => setShipment(null)} refreshOrder={refresh} />
            : <>
              <Tabs value={tab} onChange={setTab} tabs={[['cars', 'Автомобили и исполнение'], ['payments', 'Оплаты'], ['changes', 'Изменения'], ['history', 'История']]} />
              {tab === 'cars' && <>
                {models.error && <><Notice kind="danger">Не удалось загрузить названия моделей.</Notice><Button onClick={() => void models.refetch()}>Повторить загрузку моделей</Button></>}
                <OrderCars o={o} name={name} setShipment={setShipment} setBatch={setBatch} />
                <div className="kit-row">
                  {o.allowedActions.includes('confirm') && <OrderActionButton label="Подтвердить" variant="primary" refresh={refresh} onSubmit={() => post(`/commerce/orders/${id}/supplier-confirmations`, {}, { ifMatch: o.revision })} />}
                  {o.allowedActions.includes('ship-quantity') && <Button variant="primary" onClick={() => setFulfilment({ kind: 'ship-quantity' })}>Отгрузить по количеству</Button>}
                  {o.allowedActions.includes('allocate') && o.terms.lines.map((l) => <Button key={l.lineId} onClick={() => setFulfilment({ kind: 'allocate', lineId: l.lineId! })}>Назначить VIN — {orderLineLabel(l, name)}</Button>)}
                  {o.allowedActions.includes('ship') && <Button onClick={() => setFulfilment({ kind: 'ship-allocated' })}>Отгрузить назначенные VIN</Button>}
                </div>
              </>}
              {tab === 'payments' && <>
                {invoicesQuery.isLoading && <p role="status">Загрузка счетов…</p>}
                {invoicesQuery.error && <><Notice kind="danger">Не удалось загрузить счета. {errorText(invoicesQuery.error)}</Notice><Button onClick={() => void invoicesQuery.refetch()}>Повторить загрузку счетов</Button></>}
                {!invoicesQuery.isLoading && !invoicesQuery.error && <>
                  {!invoicesQuery.data?.length && <p>Счета пока не выставлены.</p>}
                  {invoicesQuery.data?.map((i) => <InvoicePanel key={i.id} invoice={i} party={o.party} refresh={refresh} />)}
                </>}
              </>}
              {tab === 'changes' && <>
                {!o.addenda.length && <p>Изменений условий пока нет.</p>}
                {o.addenda.map((a) => <Panel key={a.id} title={`Изменение №${a.number} · ${({ proposed: 'Предложено', accepted: 'Принято', rejected: 'Отклонено' })[a.status] ?? a.status}`} padded>
                  <p>{a.proposedBy === o.party ? 'Ваше предложение' : 'Предложение контрагента'} · {money(a.total)} · {a.reason}</p>
                  {a.decisionReason && <p>Причина решения: {a.decisionReason}</p>}
                  <TermsView terms={a.terms} modelNameOf={name} />
                </Panel>)}
              </>}
              {tab === 'history' && <Panel title="История заказа" padded>
                {!o.history?.length && <p>Событий пока нет.</p>}
                <ul className="kit-timeline">{o.history?.map((h, i) => <li key={i}>{dateTime(h.occurredAt)} — {eventLabel[h.type] ?? h.type}{h.reason ? ` · ${h.reason}` : ''}</li>)}</ul>
              </Panel>}
            </>}
        </>}
      </div>
    </Modal></div>
  </OrderActionContext.Provider>;
}

const milestoneLabel: Record<string, string> = {
  departed: 'Отправлено',
  'border-crossed': 'Граница пройдена',
  'customs-cleared': 'Таможня пройдена',
  arrived: 'Прибыло',
  'damage-reported': 'Повреждение',
};

function ShipmentPanel({
  id,
  party,
  onClose,
  refreshOrder,
}: {
  id: string;
  party: string;
  onClose: () => void;
  refreshOrder: unknown[][];
}) {
  const q = useData(['shipment', id], () => get<Shipment>(`/commerce/shipments/${id}`));
  const warehouses = useWarehouses();
  const s = q.data?.data;
  const refresh = [['shipment', id], ...refreshOrder, ['warehouses']];
  const waiting = s?.vehicles.filter((v) => v.status === 'shipped') ?? [];
  return (
    <section className="kit-stack">
      <Button variant="back" onClick={onClose}>Назад к заказу</Button>
      <h3 title={id}>Отгрузка {shortId(id)}</h3>
      {q.isLoading && <p role="status">Загрузка отгрузки…</p>}
      {q.error && <><Notice kind="danger">Не удалось загрузить отгрузку. {errorText(q.error)}</Notice><Button onClick={() => void q.refetch()}>Повторить загрузку отгрузки</Button></>}
      {
        s && (
          <>
            <OrderActionButton
              label="Отметить этап"
              refresh={refresh}
              fields={[
                {
                  name: 'milestoneType',
                  label: 'Этап',
                  type: 'select',
                  required: true,
                  options: Object.entries(milestoneLabel),
                },
                { name: 'occurredAt', label: 'Когда', type: 'datetime', required: true },
                { name: 'location', label: 'Где', type: 'text', required: true },
                { name: 'note', label: 'Примечание', type: 'textarea' },
              ]}
              onSubmit={(v) => post(`/commerce/shipments/${id}/milestones`, v)}
            />
            {party === 'buyer' && s.status !== 'received' && waiting.length > 0 && (
              <>
                <OrderActionButton
                  label="Принять на склад"
                  variant="primary"
                  refresh={refresh}
                  fields={[
                    {
                      name: 'vehicleIds',
                      label: 'Автомобили',
                      type: 'multiselect',
                      options: waiting.map((v) => [v.vehicleId, v.vin]),
                    },
                    {
                      name: 'warehouseId',
                      label: 'Склад',
                      type: 'select',
                      required: true,
                      options: (warehouses.data ?? []).map((w) => [w.id, `${w.name} (свободно ${w.free})`]),
                    },
                  ]}
                  onSubmit={(v) =>
                    post(
                      `/commerce/shipments/${id}/receipt-decisions`,
                      { decision: 'accept', ...v },
                      { ifMatch: s.revision },
                    )
                  }
                />
                <OrderActionButton
                  label="Отказать в приёмке"
                  variant="danger"
                  refresh={refresh}
                  fields={[
                    {
                      name: 'vehicleIds',
                      label: 'Автомобили',
                      type: 'multiselect',
                      options: waiting.map((v) => [v.vehicleId, v.vin]),
                    },
                    ...reason,
                  ]}
                  onSubmit={(v) =>
                    post(
                      `/commerce/shipments/${id}/receipt-decisions`,
                      { decision: 'reject', ...v },
                      { ifMatch: s.revision },
                    )
                  }
                />
              </>
            )}
          </>
        )
      }
      {s && (
        <>
          {s.status === 'received' && <Notice kind="success">Передача завершена. Автомобили поступили на склад покупателя; повторная приёмка не требуется.</Notice>}
          <Details
            items={[
              ['Маршрут', routeLabel[s.route]],
              ['Статус', s.status === 'received' ? 'Принята' : 'В пути'],
              ['VIN', s.vehicles.map((v) => `${v.vin} (${v.status})`).join(', ')],
            ]}
          />
          <Panel title="Этапы" padded>
            <ul className="kit-timeline">
              {s.milestones.map((m, i) => (
                <li key={i}>
                  {dateTime(m.occurredAt)} — {milestoneLabel[m.milestoneType]} · {m.location}
                  {m.note ? ` · ${m.note}` : ''}
                </li>
              ))}
            </ul>
          </Panel>
        </>
      )}
    </section>
  );
}

// ---------------- invoices ----------------

const evidenceLabel: Record<string, string> = { submitted: 'На проверке', accepted: 'Принято', rejected: 'Отклонено' };

export function InvoicePanel({
  invoice: i,
  party,
  refresh,
}: {
  invoice: Invoice;
  party: string;
  refresh: unknown[][];
}) {
  return (
    <Panel
      title={`Счёт ${money(i.total)} · оплачено ${money(i.paid)} · остаток ${money(i.outstanding)}${i.status === 'void' ? ' · аннулирован' : ''}`}
      actions={
        <>
          {i.allowedActions.includes('submit-payment') && (
            <OrderActionButton
              label="Сообщить об оплате"
              refresh={refresh}
              fields={[
                { name: 'claimedAmount', label: 'Сумма', type: 'money', required: true, currency: i.total.currency },
                { name: 'paidOn', label: 'Дата оплаты', type: 'date', required: true },
                { name: 'externalReference', label: 'Номер платёжки', type: 'text', required: true },
                { name: 'file', label: 'Подтверждение (файл)', type: 'file', purpose: 'payment-evidence' },
              ]}
              onSubmit={(v) =>
                post(`/commerce/invoices/${i.id}/payment-evidence`, {
                  claimedAmount: v.claimedAmount,
                  paidOn: v.paidOn,
                  externalReference: v.externalReference,
                  attachmentBindingIds: v.file ? [v.file] : [],
                })
              }
            />
          )}
          {i.allowedActions.includes('void') && (
            <OrderActionButton
              label="Аннулировать"
              fields={reason}
              refresh={refresh}
              onSubmit={(v) => post(`/commerce/invoices/${i.id}/void`, v, { ifMatch: i.revision })}
            />
          )}
        </>
      }
    >
      <Table
        rows={i.paymentEvidence}
        rowKey={(e) => e.id}
        empty="Оплат пока нет"
        columns={[
          { title: 'Сумма', render: (e) => money(e.amount) },
          { title: 'Дата', render: (e) => e.paidOn },
          {
            title: 'Документ',
            render: (e) => (
              <>
                {e.externalReference}{' '}
                {e.attachmentIds.map((f) => (
                  <a key={f} href={fileUrl(f)} target="_blank" rel="noreferrer">
                    файл
                  </a>
                ))}
              </>
            ),
          },
          {
            title: 'Статус',
            render: (e) => (
              <Badge tone={e.status === 'accepted' ? 'success' : e.status === 'rejected' ? 'danger' : 'warning'}>
                {evidenceLabel[e.status]}
              </Badge>
            ),
          },
          {
            title: '',
            render: (e) =>
              party === 'supplier' &&
              e.allowedActions.length > 0 && (
                <div className="kit-row">
                  {e.allowedActions.includes('accept') && <OrderActionButton
                    label="Принять"
                    variant="primary"
                    refresh={refresh}
                    fields={[{ name: 'confirmation', label: 'Деньги поступили на наш счёт', type: 'checkbox' }]}
                    onSubmit={async (v) => {
                      await post(`/commerce/payment-evidence/${e.id}/accept`, v, { ifMatch: e.revision });
                    }}
                  />}
                  {e.allowedActions.includes('reject') && <OrderActionButton
                    label="Отклонить"
                    refresh={refresh}
                    fields={reason}
                    onSubmit={async (v) => {
                      await post(`/commerce/payment-evidence/${e.id}/reject`, v, { ifMatch: e.revision });
                    }}
                  />}
                </div>
              ),
          },
        ]}
      />
    </Panel>
  );
}
