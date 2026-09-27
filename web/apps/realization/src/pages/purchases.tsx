import { useState } from 'react';
import { useSearchParams } from 'react-router-dom';
import {
  Badge,
  Button,
  Cell,
  FilterSelect,
  Modal,
  Page,
  Panel,
  ResultMeta,
  Stat,
  Stats,
  Table,
  Tabs,
  Toolbar,
  date,
  matches,
  money,
  plural,
  useSearchQuery,
} from '@justixauto/kit';
import { orderLabel, orderTone, useLinesLabel, useOffers, useOrders } from '../data';
import type { Order } from '../data';
import { OfferDialog, OrderDialog } from './trade';

const isOpen = (s: string) => ['awaiting-supplier', 'accepted', 'fulfilling'].includes(s);

/** Buyer side of wholesale: orders to suppliers, created from their published offers. */
export function PurchasesPage() {
  const orders = useOrders();
  const lines = useLinesLabel();
  const [params, setParams] = useSearchParams();
  const [tab, setTab] = useState<'open' | 'done'>('open');
  const [query, setQuery] = useSearchQuery();
  const [stage, setStage] = useState('');
  const [open, setOpen] = useState<string | null>(null);
  const catalog = params.get('catalog') === '1';
  const setCatalog = (on: boolean) => setParams(on ? { catalog: '1' } : {}, { replace: true });
  const mine = (orders.data ?? [])
    .filter((o) => o.party === 'buyer')
    .sort((a, b) => b.updatedAt.localeCompare(a.updatedAt));
  const needsMe = (o: Order) =>
    o.shipments.some((s) => s.status !== 'received') || o.addenda.some((a) => a.status === 'proposed');
  const rows = mine.filter(
    (o) =>
      (tab === 'open') === isOpen(o.status) &&
      (!stage || o.status === stage) &&
      matches(query, o.supplier.name, lines(o.terms.lines)),
  );
  const qty = rows.reduce((n, o) => n + o.terms.lines.reduce((m, l) => m + Number(l.quantity), 0), 0);
  return (
    <Page title="Закупки" subtitle="Заказы поставщикам по их опубликованным предложениям">
      <Stats>
        <Stat label="Нужен ваш ответ" value={mine.filter(needsMe).length} note="поставка или дополнение" />
        <Stat
          label="Ждём поставщика"
          value={mine.filter((o) => o.status === 'awaiting-supplier').length}
          note="заказ ждёт подтверждения"
        />
        <Stat
          label="Заказы в исполнении"
          value={mine.filter((o) => ['accepted', 'fulfilling'].includes(o.status)).length}
          note="назначение VIN, путь и приёмка"
        />
        <Stat label="Общий объём" value={`${qty} авто`} note="по текущему фильтру" />
      </Stats>
      <Panel>
        <Tabs
          value={tab}
          onChange={setTab}
          tabs={[
            ['open', 'В работе', mine.filter((o) => isOpen(o.status)).length],
            ['done', 'Завершённые', mine.filter((o) => !isOpen(o.status)).length],
          ]}
          actions={
            <Button variant="primary" size="sm" onClick={() => setCatalog(true)}>
              Создать заказ
            </Button>
          }
        />
        <Toolbar query={query} onQuery={setQuery} placeholder="Автомобиль или поставщик" onReset={() => setStage('')}>
          <FilterSelect
            value={stage}
            onChange={setStage}
            all="Все этапы"
            options={[...new Set(mine.map((o) => o.status))].map((s) => [s, orderLabel[s] ?? s])}
          />
        </Toolbar>
        <ResultMeta>{plural(rows.length, ['заказ', 'заказа', 'заказов'])}</ResultMeta>
        <Table
          rows={rows}
          loading={orders.isLoading}
          error={orders.error}
          rowKey={(o) => o.id}
          onRowClick={(o) => setOpen(o.id)}
          empty="Заказов пока нет — нажмите «Создать заказ» и выберите предложение поставщика"
          columns={[
            { title: 'Заказ', render: (o) => <Cell main={date(o.updatedAt)} sub={money(o.total)} /> },
            { title: 'Поставщик', render: (o) => o.supplier.name },
            { title: 'Автомобиль', render: (o) => lines(o.terms.lines) },
            {
              title: 'Этап',
              render: (o) => <Badge tone={orderTone(o.status)}>{orderLabel[o.status] ?? o.status}</Badge>,
            },
            {
              title: 'Действие',
              render: (o) => (
                <Button size="sm" variant={needsMe(o) ? 'primary' : 'secondary'} onClick={() => setOpen(o.id)}>
                  {needsMe(o) ? 'Принять решение' : 'Открыть'}
                </Button>
              ),
            },
          ]}
        />
      </Panel>
      {open && <OrderDialog id={open} onClose={() => setOpen(null)} />}
      {catalog && <SupplierCatalog onClose={() => setCatalog(false)} />}
    </Page>
  );
}

/** Published offers of active partners; ordering happens from the offer. */
function SupplierCatalog({ onClose }: { onClose: () => void }) {
  const q = useOffers('available');
  const lines = useLinesLabel();
  const [open, setOpen] = useState<string | null>(null);
  return (
    <Modal
      title="Предложения поставщиков"
      icon="tag"
      size="wide"
      onClose={onClose}
      help="Опубликованные предложения активных партнёров. Заказ ждёт подтверждения поставщика и не резервирует VIN сразу."
    >
      <Table
        rows={q.data}
        loading={q.isLoading}
        error={q.error}
        rowKey={(o) => o.id}
        onRowClick={(o) => setOpen(o.id)}
        empty="Партнёры пока ничего не опубликовали. Поставщик публикует предложение в своём кабинете: Предложения → Партнёрам."
        columns={[
          {
            title: 'Поставщик',
            render: (o) => (
              <Cell
                main={o.supplier.name}
                sub={o.publishedVersion ? `Версия №${o.publishedVersion.number}` : undefined}
              />
            ),
          },
          { title: 'Автомобили', render: (o) => (o.publishedVersion ? lines(o.publishedVersion.terms.lines) : '—') },
          { title: 'Итого', render: (o) => money(o.publishedVersion?.total) },
          {
            title: '',
            render: (o) => (
              <Button size="sm" onClick={() => setOpen(o.id)}>
                Открыть
              </Button>
            ),
          },
        ]}
      />
      {open && <OfferDialog id={open} onClose={() => setOpen(null)} />}
    </Modal>
  );
}
