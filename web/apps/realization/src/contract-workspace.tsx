import { useState } from 'react';
import { ActionButton, Button, Details, Notice, Panel, date, fileUrl, list, money, patch, post, useData } from '@justixauto/kit';
import type { FieldSpec } from '@justixauto/kit';

type Money = { amountMinor: string; currency: string };
type Template = { id: string; name: string; body?: string; content?: string; revision: string };
export type Contract = { id: string; state: string; kind: string; content: string; reference?: string; signedOn?: string | null; bindingIds?: string[]; fileIds?: string[]; templateId?: string | null; templateVersion?: string | number | null; priorSignedVersionId?: string | null; partySnapshot?: any; termsSnapshot?: any; vehicleSnapshot?: any; contractRevision: string; dealRevision?: string; orderRevision?: string };

export type ContractWorkspaceProps = { owner: 'retail' | 'commerce'; id: string; revision: string; price?: Money; refresh: unknown[][]; canWrite: boolean; canSign?: boolean };
const paths = (owner: ContractWorkspaceProps['owner'], id: string) => owner === 'retail'
  ? { templates: '/retail/contract-templates', contracts: `/retail/deals/${id}/contracts`, text: 'body' }
  : { templates: '/commerce/contract-templates', contracts: `/commerce/orders/${id}/contracts`, text: 'content' };
const fields = (owner: ContractWorkspaceProps['owner'], templates: Template[]): FieldSpec[] => [
  { name: 'templateId', label: 'Шаблон компании (необязательно)', type: 'select', options: [['', 'Свой текст'], ...templates.map(t => [t.id, t.name])] },
  { name: owner === 'retail' ? 'body' : 'content', label: 'Текст договора', type: 'textarea', full: true },
  { name: 'bindingId', label: 'Внешний файл договора', type: 'file', purpose: 'contract' },
];
const person = (value: any) => value?.name || value?.legalName || '—';
function Snapshot({ c, price }: { c: Contract; price?: Money }) {
  const party = c.partySnapshot ?? {}; const terms = c.termsSnapshot ?? {};
  const vehicle = party.vehicle ?? c.vehicleSnapshot;
  const shownPrice = terms.price ?? price;
  return <Details items={[
    ['Стороны', `${person(party.seller ?? party.supplier)} · ${person(party.customer ?? party.buyer)}`],
    ['Автомобиль', vehicle ? `${vehicle.model ?? vehicle.modelId ?? 'Автомобиль'} · VIN ${vehicle.vin ?? '—'}` : '—'],
    ['Цена', shownPrice ? money(shownPrice) : '—'],
    ['Схема оплаты', terms.paymentScheme ?? '—'],
  ]} />;
}
function Files({ c }: { c: Contract }) { const ids = c.bindingIds ?? c.fileIds ?? []; return ids.length ? <p>Приватные файлы: {ids.map((id, n) => <span key={id}><a href={fileUrl(id)} target="_blank" rel="noreferrer">файл {n + 1}</a>{' '}</span>)}</p> : null; }

export function ContractWorkspace({ owner, id, revision, price, refresh, canWrite, canSign = true }: ContractWorkspaceProps) {
  const p = paths(owner, id); const templates = useData(['contract-templates', owner], () => list<Template>(p.templates));
  const contracts = useData(['contracts', owner, id], () => list<Contract>(p.contracts));
  const [open, setOpen] = useState<string | null>(null); const [amendParent, setAmendParent] = useState<string>(''); const c = (contracts.data ?? []).find(x => x.id === open);
  const mutate = (path: string, body: Record<string, unknown>, ifMatch = revision) => post(path, body, { ifMatch });
  // Both APIs accept request bindingIds. Commerce deliberately exposes fileIds only in responses.
  const draftBody = (v: Record<string, unknown>, existing: string[] = []) => ({ templateId: v.templateId || undefined, [p.text]: v[p.text], bindingIds: v.bindingId ? [v.bindingId] : existing });
  const signed = (contracts.data ?? []).filter(row => row.state === 'signed');
  return <div className="kit-stack">
    <Panel title="Договоры" padded>
      <p>Текст шаблона настраивается компанией. Подписанный бумажный скан хранится приватно и открывается только после проверки доступа.</p>
      {canWrite && <div className="kit-row">
        <ActionButton label="Новый шаблон" fields={[{ name: 'name', label: 'Название', type: 'text', required: true }, { name: p.text, label: 'Текст шаблона', type: 'textarea', required: true, full: true }]} refresh={refresh} onSubmit={v => post(p.templates, { name: v.name, [p.text]: v[p.text] })} />
        <ActionButton label="Подготовить договор" variant="primary" fields={fields(owner, templates.data ?? [])} refresh={refresh} onSubmit={v => mutate(owner === 'retail' ? `${p.contracts}/drafts` : p.contracts, draftBody(v))} />
      </div>}
      {templates.error && <Notice kind="danger">Не удалось загрузить шаблоны.</Notice>}
      {(templates.data ?? []).map(t => canWrite && <ActionButton key={t.id} small label={`Изменить шаблон: ${t.name}`} fields={[{ name: 'name', label: 'Название', type: 'text', required: true, initial: t.name }, { name: p.text, label: 'Текст шаблона', type: 'textarea', required: true, full: true, initial: t.body ?? t.content ?? '' }]} refresh={refresh} onSubmit={v => patch(`${p.templates}/${t.id}`, { name: v.name, [p.text]: v[p.text] }, { ifMatch: t.revision })} />)}
    </Panel>
    {contracts.isLoading && <p role="status">Загрузка договоров…</p>}
    {contracts.error && <Notice kind="danger">Не удалось загрузить историю договоров.</Notice>}
    {!contracts.isLoading && !contracts.error && !(contracts.data ?? []).length && <p>Версий договора пока нет.</p>}
    {(contracts.data ?? []).map(row => <Panel key={row.id} title={`${row.kind === 'amendment' ? 'Дополнение' : 'Договор'} · ${row.state === 'signed' ? 'подписан' : 'черновик'}`} padded actions={<Button size="sm" onClick={() => setOpen(row.id)}>Открыть</Button>}>
      <p>{row.signedOn ? `${row.reference ?? 'Без номера'} от ${date(row.signedOn)}` : 'Черновик ещё не подписан'}</p><Snapshot c={row} price={price} />
    </Panel>)}
    {c && <Panel title={c.state === 'signed' ? 'Подписанная версия (только чтение)' : 'Черновик договора'} padded actions={<Button onClick={() => setOpen(null)}>Закрыть</Button>}>
      <Snapshot c={c} price={price} /><h3>Заполненный предварительный просмотр</h3><pre style={{ whiteSpace: 'pre-wrap' }}>{c.content}</pre><Files c={c} />
      {c.state === 'signed' ? <Notice>Подписанную версию нельзя заменить. Для новых условий создайте отдельное дополнение.</Notice> : canWrite && <div className="kit-row">
        <ActionButton label="Изменить черновик" fields={fields(owner, templates.data ?? []).map(f => f.name === p.text ? { ...f, initial: c.content } : f.name === 'templateId' ? { ...f, initial: c.templateId ?? '' } : f)} refresh={refresh} onSubmit={v => patch(`${p.contracts}/${c.id}`, { draftRevision: c.contractRevision, ...draftBody(v, c.bindingIds ?? c.fileIds ?? []) }, { ifMatch: revision })} />
        {canSign ? <><ActionButton label="Подписать бумажный договор" variant="primary" fields={[{ name: 'signedOn', label: 'Дата подписания', type: 'date', required: true }, { name: 'reference', label: 'Номер договора', type: 'text', required: true }, { name: 'bindingId', label: 'Скан с подписью', type: 'file', required: true, purpose: 'contract-signed' }]} refresh={refresh} onSubmit={v => mutate(`${p.contracts}/${c.id}/sign`, { signedOn: v.signedOn, reference: v.reference, bindingIds: [v.bindingId], draftRevision: c.contractRevision })} />
        <ActionButton label="Подписать как дополнение" fields={[{ name: 'priorSignedVersionId', label: 'Подписанная версия', type: 'select', required: true, initial: amendParent, options: signed.map(row => [row.id, `${row.reference ?? row.id} · ${row.signedOn ?? ''}`]) }, { name: 'signedOn', label: 'Дата подписания', type: 'date', required: true }, { name: 'reference', label: 'Номер дополнения', type: 'text', required: true }, { name: 'bindingId', label: 'Скан с подписью', type: 'file', required: true, purpose: 'contract-signed' }]} refresh={refresh} onSubmit={v => mutate(`${p.contracts}/${c.id}/amendments`, { priorSignedVersionId: v.priorSignedVersionId, signedOn: v.signedOn, reference: v.reference, bindingIds: [v.bindingId], draftRevision: c.contractRevision })} /></> : <Notice kind="warning">Перед первым подписанием запустите текущую demo-оценку.</Notice>}
      </div>}
      {c.state === 'signed' && canWrite && <ActionButton label="Подготовить дополнение" fields={fields(owner, templates.data ?? [])} refresh={refresh} onSubmit={v => { setAmendParent(c.id); return mutate(owner === 'retail' ? `${p.contracts}/drafts` : p.contracts, draftBody(v)); }} />}
    </Panel>}
  </div>;
}
