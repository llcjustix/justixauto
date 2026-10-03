import { useState } from 'react';
import { useSearchParams } from 'react-router-dom';
import {
  Badge,
  Button,
  Cell,
  FilterSelect,
  Modal,
  Notice,
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
  post,
  useSearchQuery,
} from '@justixauto/kit';
import { nextOrderAction } from '../order-workspace';
import { OrderCreate } from '../order-create';
import {
  orderLabel,
  orderSourceLabel,
  orderTone,
  useLinesLabel,
  useOffers,
  useOrders,
} from '../data';
import type { Offer, Order } from '../data';
import { OfferDialog, OrderDialog } from './trade';

const isOpen = (s: string) => ['awaiting-supplier', 'accepted', 'fulfilling'].includes(s);

type Side = 'out' | 'in';

const canCreateOrderFromOffer = (offer: Offer) => offer.status === 'published' && offer.publishedVersion !== null;

/** Wholesale orders of both sides: ours to suppliers and partners' orders to us. */
export function PurchasesPage() {
  const orders = useOrders();
  const lines = useLinesLabel();
  const [params, setParams] = useSearchParams();
  const [tab, setTab] = useState<Side>(params.get('tab') === 'in' ? 'in' : 'out');
  const [query, setQuery] = useSearchQuery();
  const [stage, setStage] = useState('');
  const [open, setOpen] = useState<string | null>(null);
  const [created, setCreated] = useState(false);
  const [create, setCreate] = useState<{ kind: 'direct' } | { kind: 'offer'; offer: Offer } | null>(null);
  const catalog = params.get('catalog') === '1';
  const setCatalog = (on: boolean) => setParams(on ? { catalog: '1' } : {}, { replace: true });
  const all = [...(orders.data ?? [])].sort((a, b) => b.updatedAt.localeCompare(a.updatedAt));
  const sent = all.filter((o) => o.party === 'buyer');
  const received = all.filter((o) => o.party === 'supplier');
  const mine = tab === 'out' ? sent : received;
  const other = (o: Order) => (tab === 'out' ? o.supplier.name : o.buyer.name);
  const needsMe = (o: Order) => nextOrderAction(o).needsMe;
  const rows = mine.filter((o) => (!stage || o.status === stage) && matches(query, o.id, other(o), lines(o.terms.lines)));
  const qty = rows.reduce((n, o) => n + o.terms.lines.reduce((m, l) => m + Number(l.quantity), 0), 0);
  return (
    <Page title="Закупки" subtitle="Ваши заказы поставщикам и заказы, полученные от партнёров">
      {created && <Notice kind="success">Заказ создан и отправлен поставщику на подтверждение.</Notice>}
      <Stats>
        <Stat
          label="Нужен ваш ответ"
          value={mine.filter(needsMe).length}
          note="следующий шаг по заказу"
        />
        <Stat
          label={tab === 'out' ? 'Ждём поставщика' : 'Ждут подтверждения'}
          value={mine.filter((o) => o.status === 'awaiting-supplier').length}
          note="заказ ждёт подтверждения"
        />
        <Stat
          label="Заказы в исполнении"
          value={mine.filter((o) => ['accepted', 'fulfilling'].includes(o.status)).length}
          note="назначение VIN и отгрузка"
        />
        <Stat label="Общий объём" value={`${qty} авто`} note="по текущему фильтру" />
      </Stats>
      <Panel>
        <Tabs
          value={tab}
          onChange={(t) => {
            setTab(t);
            setStage('');
          }}
          tabs={[
            ['out', 'Мои заказы', sent.filter((o) => isOpen(o.status)).length],
            ['in', 'Полученные заказы', received.filter((o) => isOpen(o.status)).length],
          ]}
          actions={
            <>
              <Button size="sm" onClick={() => setCatalog(true)}>
                Предложения от поставщиков
              </Button>
              <Button variant="primary" size="sm" onClick={() => setCreate({ kind: 'direct' })}>Создать заказ</Button>
            </>
          }
        />
        <Toolbar
          query={query}
          onQuery={setQuery}
          placeholder={tab === 'out' ? 'Автомобиль или поставщик' : 'Автомобиль или покупатель'}
          onReset={() => setStage('')}
        >
          <FilterSelect value={stage} onChange={setStage} all="Все статусы" options={Object.entries(orderLabel)} />
        </Toolbar>
        <ResultMeta>{plural(rows.length, ['заказ', 'заказа', 'заказов'])}</ResultMeta>
        <Table
          rows={rows}
          loading={orders.isLoading}
          error={orders.error}
          rowKey={(o) => o.id}
          onRowClick={(o) => setOpen(o.id)}
          empty={tab === 'out' ? 'Заказов пока нет — нажмите «Создать заказ»' : 'Партнёры пока ничего не заказали'}
          columns={[
            {
              title: 'Заказ',
              render: (o) => (
                <Cell main={`Заказ ${o.id} · ${money(o.total)}`} sub={`${date(o.updatedAt)} · ${orderSourceLabel[o.source] ?? ''} · отгружено ${o.lineProgress?.reduce((n, p) => n + Number(p.shipped), 0) ?? 0} из ${o.terms.lines.reduce((n, l) => n + Number(l.quantity), 0)}`} />
              ),
            },
            { title: tab === 'out' ? 'Поставщик' : 'Покупатель', render: other },
            { title: 'Автомобиль', render: (o) => lines(o.terms.lines) },
            {
              title: 'Статус',
              render: (o) => <Badge tone={orderTone(o.status)}>{orderLabel[o.status] ?? o.status}</Badge>,
            },
            {
              title: 'Действие',
              render: (o) => (
                <Button size="sm" variant={needsMe(o) ? 'primary' : 'secondary'} onClick={() => setOpen(o.id)}>
                  {nextOrderAction(o).label}
                </Button>
              ),
            },
          ]}
        />
      </Panel>
      {open && <OrderDialog id={open} onClose={() => setOpen(null)} />}
      {catalog && <SupplierCatalog onClose={() => setCatalog(false)} onCreate={(offer) => {
        if (!canCreateOrderFromOffer(offer)) return;
        setCatalog(false);
        setCreate({ kind: 'offer', offer });
      }} />}
      {create && <OrderCreate mode={create} onClose={() => setCreate(null)} onCreated={(id) => { setCreate(null); setCreated(true); setOpen(id); }} />}
    </Page>
  );
}

/** Published offers of active partners; ordering happens from the offer. */
function SupplierCatalog({ onClose, onCreate }: { onClose: () => void; onCreate: (offer: Offer) => void }) {
  const q = useOffers('available');
  const lines = useLinesLabel();
  const [open, setOpen] = useState<string | null>(null);
  if (open) return <OfferDialog id={open} onClose={() => setOpen(null)} onStartOrder={onCreate} />;
  return (
    <Modal
      title="Предложения от поставщиков"
      icon="tag"
      size="wide"
      onClose={onClose}
      help="Опубликованные предложения и специальные цены партнёров. Заказ по предложению ждёт подтверждения поставщика; количество не ограничено."
      footer={<Button onClick={onClose}>Закрыть</Button>}
    >
      <Table
        rows={q.data}
        loading={q.isLoading}
        error={q.error}
        rowKey={(o) => o.id}
        onRowClick={(o) => setOpen(o.id)}
        empty="Сейчас у партнёров нет предложений. Заказать можно и без предложения: закройте окно и нажмите «Создать заказ»."
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
              <span onClick={(event) => event.stopPropagation()}>
                <Button size="sm" onClick={() => setOpen(o.id)}>
                  Открыть
                </Button>
              </span>
            ),
          },
          {
            title: 'Действие',
            render: (o) => canCreateOrderFromOffer(o) ? (
              <span onClick={(event) => event.stopPropagation()}>
                <Button size="sm" variant="primary" onClick={() => onCreate(o)}>
                  Создать заказ
                </Button>
              </span>
            ) : '—',
          },
        ]}
      />
    </Modal>
  );
}
