import { useId, useState } from 'react';
import { Button } from '@justixauto/kit';

import { normalizeVin, validateVins } from './vin-validation';

export interface VinEditorProps {
  rows: string[];
  onChange: (rows: string[]) => void;
  cap: number;
  required?: boolean;
  disabled?: boolean;
  label?: string;
  showEmptyRequiredError?: boolean;
}

/** Controlled drafts survive server failures. The caller supplies its actual command limit. */
export function VinEditor({
  rows,
  onChange,
  cap,
  required = false,
  disabled = false,
  label = 'VIN',
  showEmptyRequiredError = true,
}: VinEditorProps) {
  const id = useId();
  const [paste, setPaste] = useState('');
  const [page, setPage] = useState(0);
  const checked = validateVins(rows, cap, required);
  const pageSize = 50;
  const pages = Math.max(1, Math.ceil(rows.length / pageSize));
  const current = Math.min(page, pages - 1);
  const start = current * pageSize;
  const update = (index: number, value: string) => onChange(rows.map((row, i) => (i === index ? value : row)));
  const invalidCount = checked.rowErrors.filter(Boolean).length;
  return (
    <fieldset className="vin-editor" disabled={disabled}>
      <legend>
        {label}
        {required ? ' — обязательно' : ' — необязательно'}
      </legend>
      <p className="vin-editor-count">
        Введено {checked.vins.length} из {cap}
        {checked.vins.length < cap ? ` · можно добавить ещё ${cap - checked.vins.length}` : ''}
      </p>
      {checked.errors
        .filter((error) => showEmptyRequiredError || error !== 'Введите хотя бы один VIN')
        .map((error) => (
          <p className="vin-editor-error" role="alert" key={error}>
            {error}
          </p>
        ))}
      {invalidCount > 0 && (
        <p className="vin-editor-error">Строк с ошибками: {invalidCount}, включая другие страницы.</p>
      )}
      {rows.slice(start, start + pageSize).map((row, offset) => {
        const index = start + offset;
        return (
          <div className="vin-editor-row" key={index}>
            <label htmlFor={`${id}-${index}`}>VIN {index + 1}</label>
            <input
              id={`${id}-${index}`}
              value={row}
              aria-invalid={!!checked.rowErrors[index]}
              placeholder="17 знаков"
              autoComplete="off"
              spellCheck={false}
              aria-describedby={checked.rowErrors[index] ? `${id}-error-${index}` : undefined}
              onChange={(event) => update(index, event.target.value)}
              onBlur={() => update(index, normalizeVin(row))}
            />
            <Button variant="link" onClick={() => onChange(rows.filter((_, i) => i !== index))}>
              Удалить VIN {index + 1}
            </Button>
            {checked.rowErrors[index] && <span id={`${id}-error-${index}`}>{checked.rowErrors[index]}</span>}
          </div>
        );
      })}
      <div className="vin-editor-actions">
        <Button
          size="sm"
          disabled={rows.length >= cap}
          onClick={() => {
            onChange([...rows, '']);
            setPage(Math.floor(rows.length / pageSize));
          }}
        >
          Добавить VIN
        </Button>
      </div>
      {pages > 1 && (
        <div className="vin-editor-pages">
          <Button disabled={current === 0} onClick={() => setPage(current - 1)}>
            Предыдущие VIN
          </Button>
          <span>
            Страница VIN {current + 1} из {pages}
          </span>
          <Button disabled={current === pages - 1} onClick={() => setPage(current + 1)}>
            Следующие VIN
          </Button>
        </div>
      )}
      <details>
        <summary>Вставить список VIN</summary>
        <label htmlFor={`${id}-paste`}>Список VIN для добавления</label>
        <textarea id={`${id}-paste`} value={paste} onChange={(event) => setPaste(event.target.value)} />
        <Button
          disabled={!paste.trim()}
          onClick={() => {
            const added = paste.trim().split(/[\s,;]+/);
            // Retain every supplied row, including excess rows, so validation is explicit.
            onChange([...rows, ...added.map(normalizeVin)]);
            setPage(Math.floor(rows.length / pageSize));
            setPaste('');
          }}
        >
          Добавить список
        </Button>
      </details>
    </fieldset>
  );
}
