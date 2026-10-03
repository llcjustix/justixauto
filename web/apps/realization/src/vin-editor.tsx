import { useId, useState } from 'react';
import { Button } from '@justixauto/kit';

const normalizeVin = (value: string) => value.trim().replace(/[a-z]/g, (letter) => letter.toUpperCase());

export function validateVins(rows: string[], cap: number, required = false) {
  const vins = rows.map(normalizeVin).filter(Boolean);
  const counts = new Map<string, number>();
  vins.forEach((vin) => counts.set(vin, (counts.get(vin) ?? 0) + 1));
  const rowErrors = rows.map((row) => {
    if (!row.trim()) return '';
    if (!/^[A-HJ-NPR-Za-hj-npr-z0-9]{17}$/.test(row.trim())) return 'VIN: 17 латинских букв и цифр, без I, O, Q';
    return (counts.get(normalizeVin(row)) ?? 0) > 1 ? 'Повторяющийся VIN' : '';
  });
  const errors: string[] = [];
  if (required && vins.length === 0) errors.push('Введите хотя бы один VIN');
  if (vins.length > cap) errors.push(`VIN больше допустимого: ${vins.length} из ${cap}. Удалите лишние строки или отправьте отдельной партией.`);
  if (rowErrors.some(Boolean)) errors.push('Исправьте неверные или повторяющиеся VIN');
  return { vins, rowErrors, errors, valid: errors.length === 0 };
}

export interface VinEditorProps {
  rows: string[];
  onChange: (rows: string[]) => void;
  cap: number;
  required?: boolean;
  disabled?: boolean;
  label?: string;
}

/** Controlled drafts survive server failures. The caller supplies its actual command limit. */
export function VinEditor({ rows, onChange, cap, required = false, disabled = false, label = 'VIN' }: VinEditorProps) {
  const id = useId();
  const [paste, setPaste] = useState('');
  const [page, setPage] = useState(0);
  const checked = validateVins(rows, cap, required);
  const pageSize = 50;
  const pages = Math.max(1, Math.ceil(rows.length / pageSize));
  const current = Math.min(page, pages - 1);
  const start = current * pageSize;
  const update = (index: number, value: string) => onChange(rows.map((row, i) => i === index ? value : row));
  const invalidCount = checked.rowErrors.filter(Boolean).length;
  return (
    <fieldset className="vin-editor" disabled={disabled}>
      <legend>{label}{required ? ' — обязательно' : ' — необязательно'}</legend>
      <p>Введено VIN: {checked.vins.length} · Можно добавить: {Math.max(0, cap - checked.vins.length)} · Лимит: {cap}</p>
      {checked.errors.map((error) => <p role="alert" key={error}>{error}</p>)}
      {invalidCount > 0 && <p>Строк с ошибками: {invalidCount}, включая другие страницы.</p>}
      {rows.slice(start, start + pageSize).map((row, offset) => {
        const index = start + offset;
        return (
          <div className="vin-editor-row" key={index}>
            <label htmlFor={`${id}-${index}`}>VIN {index + 1}</label>
            <input id={`${id}-${index}`} value={row} aria-invalid={!!checked.rowErrors[index]}
              aria-describedby={checked.rowErrors[index] ? `${id}-error-${index}` : undefined}
              onChange={(event) => update(index, event.target.value)}
              onBlur={() => update(index, normalizeVin(row))} />
            <Button onClick={() => onChange(rows.filter((_, i) => i !== index))}>Удалить VIN {index + 1}</Button>
            {checked.rowErrors[index] && <span id={`${id}-error-${index}`}>{checked.rowErrors[index]}</span>}
          </div>
        );
      })}
      <Button onClick={() => { onChange([...rows, '']); setPage(Math.floor(rows.length / pageSize)); }}>Добавить VIN</Button>
      {pages > 1 && <div className="vin-editor-pages">
        <Button disabled={current === 0} onClick={() => setPage(current - 1)}>Предыдущие VIN</Button>
        <span>Страница VIN {current + 1} из {pages}</span>
        <Button disabled={current === pages - 1} onClick={() => setPage(current + 1)}>Следующие VIN</Button>
      </div>}
      <details>
        <summary>Вставить список VIN</summary>
        <label htmlFor={`${id}-paste`}>Список VIN для добавления</label>
        <textarea id={`${id}-paste`} value={paste} onChange={(event) => setPaste(event.target.value)} />
        <Button disabled={!paste.trim()} onClick={() => {
          const added = paste.trim().split(/[\s,;]+/);
          // Retain every supplied row, including excess rows, so validation is explicit.
          onChange([...rows, ...added.map(normalizeVin)]);
          setPage(Math.floor(rows.length / pageSize));
          setPaste('');
        }}>Добавить список</Button>
      </details>
    </fieldset>
  );
}
