import { useState } from 'react';
import {
  ActionButton,
  Cell,
  FinanceDialog,
  FinanceTable,
  Notice,
  Table,
  Tabs,
  list,
  money,
  percent,
  post,
  useCommercialAccess,
  useCompanyNames,
  useData,
  useFinanceApplications,
} from '@justixauto/kit';
import type { Program } from '@justixauto/kit';
import { useDeals } from '../data';

function useDealLabel() {
  const deals = useDeals();
  return (id: string) => {
    const d = deals.data?.find((x) => x.id === id);
    return d ? `${d.customer.displayName} · ${money(d.price)}` : id.slice(0, 8);
  };
}

/** Seller's applications to the Islamic line (Продажи → Клиентам → Исламская линия). */
export function FinancingPanel() {
  const commercial = useCommercialAccess();
  const [tab, setTab] = useState<'applications' | 'programs'>('applications');
  const q = useFinanceApplications();
  const programs = useData(['programs', 'published'], () => list<Program>('/financing/programs?limit=100'));
  const deals = useDeals();
  const name = useCompanyNames();
  const dealLabel = useDealLabel();
  const [open, setOpen] = useState<string | null>(null);
  const eligible = (deals.data ?? []).filter((d) => d.status === 'reserved' && d.paymentScheme === 'partner-finance');
  const published = (programs.data ?? []).filter((p) => p.publishedVersion !== null);
  const current = (p: Program) => p.versions.find((v) => Number(v.number) === p.publishedVersion);
  return (
    <>
      <Tabs
        value={tab}
        onChange={setTab}
        tabs={[
          ['applications', 'Заявки', q.data?.length],
          ['programs', 'Программы партнёров', published.length],
        ]}
        actions={
          <ActionButton
            label="Заявка в Исламскую линию"
            variant="primary"
            refresh={[['finance-applications']]}
            fields={[
              {
                name: 'retailDealId',
                label: 'Продажа (Исламская линия)',
                type: 'select',
                required: true,
                options: eligible.map((d) => [d.id, dealLabel(d.id)]),
              },
              {
                name: 'programId',
                label: 'Программа',
                type: 'select',
                required: true,
                options: published.map((p) => [p.id, `${p.provider.name} — ${current(p)?.name ?? ''}`]),
              },
              { name: 'downPayment', label: 'Первый взнос', type: 'money', required: true },
              { name: 'termMonths', label: 'Срок, месяцев', type: 'number', required: true },
              { name: 'firstDueDate', label: 'Дата первого платежа', type: 'date', required: true },
            ]}
            onSubmit={(v) => {
              if (!commercial.allowed) throw new Error(commercial.reason);
              const p = published.find((x) => x.id === v.programId)!;
              return post('/financing/applications', {
                retailDealId: v.retailDealId,
                providerCompanyId: p.provider.id,
                programId: p.id,
                programVersion: p.publishedVersion,
                calculationInputs: {
                  downPayment: v.downPayment,
                  termMonths: Number(v.termMonths),
                  firstDueDate: v.firstDueDate,
                },
              });
            }}
            disabled={!commercial.allowed}
          />
        }
      />
      {!commercial.allowed && (
        <Notice kind="warning">
          {commercial.reason} <a href="/settings">Открыть настройки компании</a>.
        </Notice>
      )}
      {tab === 'applications' ? (
        <FinanceTable
          rows={q.data}
          loading={q.isLoading}
          error={q.error}
          counterparty={(a) => `${name(a.providerCompanyId)} · ${dealLabel(a.retailDealId)}`}
          onOpen={setOpen}
        />
      ) : (
        <Table
          rows={published}
          loading={programs.isLoading}
          error={programs.error}
          rowKey={(p) => p.id}
          empty="Исламская линия пока не опубликовала программ"
          columns={[
            {
              title: 'Исламская линия',
              render: (p) => <Cell main={p.provider.name} sub={p.provider.kind === 'bank' ? 'Банк' : 'МФО'} />,
            },
            { title: 'Программа', render: (p) => current(p)?.name },
            { title: 'Наценка', render: (p) => current(p) && percent(current(p)!.terms.markupBps) },
            { title: 'Мин. взнос', render: (p) => current(p) && percent(current(p)!.terms.minDownPaymentBps) },
            { title: 'Сроки', render: (p) => `${current(p)?.terms.termMonths.join(', ')} мес.` },
            { title: 'Валюта', render: (p) => current(p)?.currency },
          ]}
        />
      )}
      {open && <FinanceDialog id={open} onClose={() => setOpen(null)} describeDeal={dealLabel} />}
    </>
  );
}
