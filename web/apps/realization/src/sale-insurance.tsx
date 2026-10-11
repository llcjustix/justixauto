import { useState } from 'react';
import { useQueryClient } from '@tanstack/react-query';
import { ActionButton, Button, Notice, Panel, dateTime, list, money, post, useData, useSession } from '@justixauto/kit';
import type { Deal } from './data';
import type { InsuranceApplication } from '@justixauto/kit';
import { InsuranceCase } from '@justixauto/kit';

/** Own-installment-only content embedded in the sale modal; it never claims a policy or coverage. */
export function SaleInsurance({ deal, refresh }: { deal: Deal; refresh: unknown[][] }) {
  const session = useSession();
  const client = useQueryClient();
  const applications = useData(['insurance-for-deal', deal.id], () => list<InsuranceApplication>('/insurance/applications?limit=100'));
  const [demoBusy, setDemoBusy] = useState(false);
  const [demoError, setDemoError] = useState('');
  const application = applications.data?.find(a => a.retailDealId === deal.id);
  const canRunDemo = session.can('retail.deals.manage');
  const runDemo = async () => {
    setDemoBusy(true); setDemoError('');
    try {
      await post(`/retail/deals/${deal.id}/demo-scoring`, {}, { ifMatch: deal.revision });
      // The new response has a revision of its own; all parent resources must refetch before another action.
      await Promise.all([...refresh, ['insurance-for-deal', deal.id], ['insurance-applications']].map(queryKey => client.invalidateQueries({ queryKey }, { throwOnError: true })));
    } catch {
      setDemoError('Не удалось запустить demo-оценку. Обновите сделку и повторите попытку.');
    } finally { setDemoBusy(false); }
  };
  return <div className="kit-stack">
    <Panel title="Демонстрационная оценка" padded>
      <Notice kind="warning">Демонстрационный результат не является полисом, покрытием или решением страховой компании.</Notice>
      {deal.demoScoring ? <p>{deal.demoScoring.current ? 'Текущая демонстрационная оценка завершена.' : 'Предыдущая оценка устарела; перед первым подписанием нужен новый запуск.'} · {dateTime(deal.demoScoring.completedAt)}</p> : <p>Оценка ещё не запускалась.</p>}
      {demoError && <Notice kind="danger">{demoError}</Notice>}
      {canRunDemo && <Button variant="primary" disabled={demoBusy} onClick={() => void runDemo()}>{demoBusy ? 'Запуск demo-оценки…' : 'Запустить / повторить demo-оценку'}</Button>}
      {!canRunDemo && <Notice kind="warning">Недостаточно прав для запуска demo-оценки.</Notice>}
    </Panel>
    {applications.error && <Notice kind="danger">Не удалось загрузить страховые заявки.</Notice>}
    {application ? <InsuranceCase id={application.id} describeDeal={() => `${deal.customer.displayName} · ${money(deal.price)}`} refresh={[...refresh, ['insurance-for-deal', deal.id], ['insurance-applications']]} /> : <Panel title="Заявка на страхование" padded>
      <p>Создайте черновик, выберите страховую и отправьте заявку из этой сделки.</p>
      <ActionButton label="Создать заявку" variant="primary" refresh={[...refresh, ['insurance-for-deal', deal.id], ['insurance-applications']]} fields={[
        { name: 'insurerCompanyId', label: 'Страховая компания', type: 'select', searchable: true, required: true, options: [], search: async text => (await list<{ id: string; name: string }>(`/identity/directory/companies?kind=insurance&limit=10&q=${encodeURIComponent(text)}`)).map(c => [c.id, c.name] as [string, string]) },
        { name: 'note', label: 'Комментарий для страховой', type: 'textarea' },
      ]} onSubmit={v => post('/insurance/applications', { retailDealId: deal.id, insurerCompanyId: v.insurerCompanyId, note: v.note ?? '' })} />
    </Panel>}
  </div>;
}
