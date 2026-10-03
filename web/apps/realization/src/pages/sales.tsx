import { useEffect, useState } from 'react';
import { useSearchParams } from 'react-router-dom';
import {
  Badge,
  Button,
  Cell,
  FilterSelect,
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
  useFinanceApplications,
  useSearchQuery,
} from '@justixauto/kit';
import {
  dealLabel,
  schemeLabel,
  useBranches,
  useDeals,
  useModels,
  useVehicles,
} from '../data';
import { DealDialog } from './retail';
import { FinancingPanel } from './partners-finance';
import { NewSale } from '../sale-form';

type View = 'active' | 'delivered' | 'installments' | 'finance';

export function SalesPage() {
  const [params] = useSearchParams();
  // A sale created from the header button opens its deal card in the list below.
  const [created, setCreated] = useState<string | null>(null);
  return (
    <Page
      title="Продажи"
      subtitle="Розничные продажи вашей компании"
      actions={<NewSale autoOpen={params.get('new') === '1'} onOpenDeal={setCreated} />}
    >
      <ClientSales openId={created} onOpened={() => setCreated(null)} />
    </Page>
  );
}

function useCarLabel() {
  const vehicles = useVehicles();
  const models = useModels();
  return (id: string) => {
    const v = vehicles.data?.find((x) => x.id === id);
    const m = models.data?.find((x) => x.id === v?.modelId)?.specification;
    return { model: m ? `${m.make} ${m.model} ${m.variant}` : '—', vin: v?.vin ?? '' };
  };
}

function ClientSales({ openId, onOpened }: { openId: string | null; onOpened: () => void }) {
  const q = useDeals();
  const branches = useBranches();
  const finance = useFinanceApplications();
  const car = useCarLabel();
  const [view, setView] = useState<View>('active');
  const [query, setQuery] = useSearchQuery();
  const [scheme, setScheme] = useState('');
  const [branch, setBranch] = useState('');
  const [open, setOpen] = useState<string | null>(null);
  useEffect(() => { if (openId) { setOpen(openId); onOpened(); } }, [openId]);
  const all = q.data ?? [];
  const inView = (d: (typeof all)[number]) =>
    view === 'active'
      ? d.status === 'reserved'
      : view === 'delivered'
        ? d.status === 'delivered'
        : view === 'installments'
          ? d.paymentScheme === 'own-installment' && d.status !== 'cancelled'
          : false;
  const rows = all.filter(
    (d) =>
      inView(d) &&
      (!scheme || d.paymentScheme === scheme) &&
      (!branch || d.branchId === branch) &&
      matches(query, d.customer.displayName, d.customer.phone, car(d.vehicleId).vin, car(d.vehicleId).model),
  );
  const count = (v: View) =>
    all.filter((d) =>
      v === 'active'
        ? d.status === 'reserved'
        : v === 'delivered'
          ? d.status === 'delivered'
          : d.paymentScheme === 'own-installment' && d.status !== 'cancelled',
    ).length;
  return (
    <>
      <Stats>
        <Stat label="Активные продажи" value={count('active')} note="до выдачи автомобиля" />
        <Stat label="Рассрочки" value={count('installments')} note="собственная рассрочка продавца" />
        <Stat
          label="Заявки в банк / МФО"
          value={(finance.data ?? []).filter((a) => !['declined', 'agreed'].includes(a.status)).length}
          note="на рассмотрении у партнёров"
        />
        <Stat label="Выдано" value={count('delivered')} note="автомобилей клиентам" />
      </Stats>
      <Panel>
        <Tabs
          value={view}
          onChange={setView}
          tabs={[
            ['active', 'Активные продажи', count('active')],
            ['delivered', 'Выданные', count('delivered')],
            ['installments', 'Рассрочки', count('installments')],
            ['finance', 'Банк / МФО', finance.data?.length],
          ]}
        />
        {view === 'finance' ? (
          <FinancingPanel />
        ) : (
          <>
            <Toolbar
              query={query}
              onQuery={setQuery}
              placeholder="Клиент, телефон или VIN"
              onReset={() => {
                setScheme('');
                setBranch('');
              }}
            >
              <FilterSelect
                value={scheme}
                onChange={setScheme}
                all="Все способы оформления"
                options={Object.entries(schemeLabel)}
              />
              <FilterSelect
                value={branch}
                onChange={setBranch}
                all="Все филиалы"
                options={(branches.data ?? []).map((b) => [b.id, b.name])}
              />
            </Toolbar>
            <ResultMeta>{plural(rows.length, ['продажа', 'продажи', 'продаж'])}</ResultMeta>
            <Table
              rows={rows}
              loading={q.isLoading}
              error={q.error}
              rowKey={(d) => d.id}
              onRowClick={(d) => setOpen(d.id)}
              empty="Продаж в этом разделе нет"
              columns={[
                {
                  title: 'Сделка',
                  render: (d) => <Cell main={date(d.updatedAt)} sub={schemeLabel[d.paymentScheme]} />,
                },
                { title: 'Клиент', render: (d) => <Cell main={d.customer.displayName} sub={d.customer.phone} /> },
                {
                  title: 'Автомобиль',
                  render: (d) => <Cell main={car(d.vehicleId).model} sub={car(d.vehicleId).vin} />,
                },
                { title: 'Филиал', render: (d) => branches.data?.find((b) => b.id === d.branchId)?.name ?? '—' },
                {
                  title: 'Этап',
                  render: (d) => (
                    <Badge tone={d.status === 'delivered' ? 'success' : d.status === 'cancelled' ? 'danger' : 'info'}>
                      {dealLabel[d.status]}
                    </Badge>
                  ),
                },
                { title: 'Сумма', render: (d) => money(d.price) },
                {
                  title: 'Действие',
                  render: (d) => (
                    <Button
                      size="sm"
                      variant={d.status === 'reserved' ? 'primary' : 'secondary'}
                      onClick={() => setOpen(d.id)}
                    >
                      {view === 'installments' ? 'Открыть график' : d.status === 'reserved' ? 'Продолжить' : 'Открыть'}
                    </Button>
                  ),
                },
              ]}
            />
          </>
        )}
      </Panel>
      {open && <DealDialog id={open} onClose={() => setOpen(null)} />}
    </>
  );
}
