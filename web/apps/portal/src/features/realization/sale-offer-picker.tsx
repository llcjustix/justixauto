import { useState } from 'react';
import { Autocomplete, Button, Notice, errorText, money, useData } from '@justixauto/kit';
import type { Model } from './data';
import { checkAppliedSource, readSaleSource, readSaleSources, resolveSaleSource, saleLineKey, shownSourceSeed, sourceDetailKey, sourceLabels, sourceVersion } from './sale-offer-source';
import type { AppliedSaleSource, SaleSourceKind, SaleSourceSeed } from './sale-offer-source';

export function SaleOfferPicker({ company, seed: initialSeed, models, disabled, onApply, onCancel, onDetach }: {
  company: string; seed?: SaleSourceSeed | undefined; models: Model[]; disabled: boolean;
  onApply: (source: AppliedSaleSource) => void; onCancel: () => void; onDetach: () => void;
}) {
  const [kind, setKind] = useState<SaleSourceKind | ''>(initialSeed?.kind ?? 'supplier-offer');
  const [seed, setSeed] = useState(initialSeed);
  const [line, setLine] = useState<string | null>(null);
  const [staged, setStaged] = useState<AppliedSaleSource | null>(null);
  const options = useData(['sale-sources', company, kind], () => readSaleSources(kind as SaleSourceKind), !!kind && !!company);
  const detail = useData(sourceDetailKey(company, seed), () => readSaleSource(seed!), !!seed && !!company);
  let issue = '', candidate: AppliedSaleSource | null = null;
  let lines: { value: string; label: string }[] = [];
  if (seed && detail.data) {
    try {
      if (seed.kind !== 'own-listing') {
        lines = sourceVersion(detail.data, seed, company, true).terms.lines.map((item, index) => ({
          value: saleLineKey(item, index),
          label: `${models.find(m => m.id === item.modelId)?.specification.model ?? item.modelId} · ${item.modelSpecificationVersion || 'Версия не указана'} · Кузов: ${item.exteriorColor || 'любой'} · Салон: ${item.interiorColor || 'любой'} · ${money(item.unitPrice)} · Строка ${index + 1}`,
        }));
      }
      candidate = resolveSaleSource(detail.data, seed, line ?? (lines.length === 1 ? lines[0]!.value : ''), company, true);
      if (staged && checkAppliedSource(detail.data, staged, company)) issue = 'Предложение изменилось. Подтвердите актуальные данные.';
    } catch (error) { issue = errorText(error); }
  }
  const healthy = !!seed && options.data?.some(item => item.id === seed.id) && !options.isFetching && !options.error && !detail.isFetching && !detail.error;
  const reset = (next?: SaleSourceSeed) => { setSeed(next); setLine(null); setStaged(null); };
  return <section className="kit-stack" aria-label="Выбор предложения">
    <label className="kit-field">Источник цены<Autocomplete aria-label="Источник цены" disabled={disabled} value={kind}
      options={Object.entries(sourceLabels).map(([value, label]) => ({ value, label }))}
      onChange={value => { setKind(value as SaleSourceKind | ''); reset(); }} /></label>
    {kind && <label className="kit-field">Предложение<Autocomplete aria-label="Предложение" disabled={disabled} value={seed?.id ?? ''}
      loading={options.isFetching} error={options.error ? errorText(options.error) : undefined}
      options={(options.data ?? []).filter(item => item.status !== 'withdrawn').map(item => ({ value: item.id,
        label: 'supplier' in item ? `${item.supplier.name} · ${item.id}` : `${item.text || item.id} · ${item.status === 'draft' ? 'Черновик' : 'Опубликовано'}` }))}
      onChange={id => { const item = options.data?.find(item => item.id === id); reset(item ? shownSourceSeed(kind, item) : undefined); }} /></label>}
    {(options.isFetching || seed && detail.isFetching) && <p role="status">Загрузка предложений…</p>}
    {(options.error || detail.error) && <Notice kind="danger">Не удалось загрузить предложение. {errorText(options.error || detail.error)}</Notice>}
    {kind && !options.isFetching && !options.error && options.data?.length === 0 && <p>Нет доступных предложений. Измените источник или продолжите вручную.</p>}
    {seed && options.data && !options.data.some(item => item.id === seed.id) && <Notice kind="danger">Предложение недоступно в списке текущей компании.</Notice>}
    {lines.length > 0 && <label className="kit-field">Строка предложения<Autocomplete aria-label="Строка предложения" disabled={disabled}
      value={line ?? (lines.length === 1 ? lines[0]!.value : '')} options={lines} onChange={value => { setLine(value); setStaged(null); }} /></label>}
    {issue && <Notice kind="danger">{issue}</Notice>}
    {seed && detail.data && issue && <Button disabled={disabled || detail.isFetching} onClick={() => {
      const next = shownSourceSeed(seed.kind, detail.data!); reset(next);
    }}>Выбрать актуальную версию</Button>}
    {candidate && <p>{candidate.label} · Цена за один автомобиль: {money(candidate.price)}</p>}
    <div className="kit-row">
      <Button disabled={disabled || !kind || options.isFetching || detail.isFetching} onClick={() => {
        setLine(null); setStaged(candidate); void options.refetch(); if (seed) void detail.refetch();
      }}>Обновить предложения</Button>
      <Button disabled={disabled} onClick={onCancel}>Отменить выбор предложения</Button>
      <Button disabled={disabled} onClick={onDetach}>Продолжить вручную</Button>
      <Button disabled={disabled || !healthy || !candidate || !!issue} onClick={() => { if (healthy && candidate && !issue) onApply(candidate); }}>Применить предложение</Button>
    </div>
  </section>;
}
