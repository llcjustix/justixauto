import { useId, useLayoutEffect, useRef, useState } from 'react';
import type { CSSProperties } from 'react';

export type AutocompleteOption = { value: string; label: string; disabled?: boolean };

export interface AutocompleteProps {
  id?: string | undefined;
  'aria-label'?: string | undefined;
  options: readonly AutocompleteOption[];
  value: string;
  onChange: (value: string) => void;
  mode?: 'strict' | 'free';
  required?: boolean | undefined;
  disabled?: boolean | undefined;
  placeholder?: string | undefined;
  emptyLabel?: string | undefined;
  allowClear?: boolean | undefined;
  selectedOption?: AutocompleteOption | undefined;
  onBlur?: (() => void) | undefined;
  onQueryChange?: ((query: string) => void) | undefined;
  filter?: boolean | undefined;
  loading?: boolean | undefined;
  error?: string | undefined;
  onRetry?: (() => void) | undefined;
}

/** Query text is deliberately separate from the committed option identity. */
export function Autocomplete({
  id, 'aria-label': ariaLabel, options, value, onChange, mode = 'strict', required,
  disabled, placeholder, emptyLabel = 'Ничего не найдено', allowClear = true,
  selectedOption, onBlur, onQueryChange, filter = true, loading, error, onRetry,
}: AutocompleteProps) {
  const generatedId = useId();
  const inputId = id ?? generatedId;
  const listId = `${inputId}-list`;
  const input = useRef<HTMLInputElement>(null);
  const control = useRef<HTMLSpanElement>(null);
  const [placement, setPlacement] = useState<CSSProperties>();
  const labels = useRef(new Map<string, string>());
  const [draft, setDraft] = useState({ value, text: '', editing: false });
  const [open, setOpen] = useState(false);
  const [active, setActive] = useState<string | null>(null);
  const current = selectedOption?.value === value ? selectedOption : options.find((option) => option.value === value);
  const label = current?.label ?? labels.current.get(value) ?? value;
  const editing = draft.value === value && draft.editing;
  const text = editing ? draft.text : mode === 'free' ? value : label;
  const query = editing ? text.trim().toLocaleLowerCase() : '';
  const shown = options.filter((option) => !filter || !query || option.label.toLocaleLowerCase().includes(query));
  const enabled = shown.filter((option) => !option.disabled);
  const expanded = open && !disabled;
  const activeOption = expanded ? shown.find((option) => option.value === active && !option.disabled) : undefined;
  const optionId = (option: AutocompleteOption) => `${listId}-${encodeURIComponent(option.value)}`;

  useLayoutEffect(() => {
    if (activeOption) document.getElementById(optionId(activeOption))?.scrollIntoView?.({ block: 'nearest' });
  }, [active, expanded]);

  // A scrolling dialog body would clip an absolutely placed list, so the open list
  // is pinned to the viewport next to its field and flips up when there is no room below.
  useLayoutEffect(() => {
    if (!expanded) return;
    const place = () => {
      const box = control.current?.getBoundingClientRect();
      if (!box) return;
      const below = window.innerHeight - box.bottom - 12;
      const above = box.top - 12;
      const up = below < 200 && above > below;
      setPlacement({
        position: 'fixed', boxSizing: 'border-box', left: box.left, right: 'auto', width: box.width,
        maxHeight: Math.max(120, Math.min(264, up ? above : below)),
        ...(up ? { top: 'auto', bottom: window.innerHeight - box.top + 4 } : { top: box.bottom + 4 }),
      });
    };
    place();
    window.addEventListener('scroll', place, true);
    window.addEventListener('resize', place);
    return () => { window.removeEventListener('scroll', place, true); window.removeEventListener('resize', place); };
  }, [expanded]);

  useLayoutEffect(() => {
    for (const option of options) labels.current.set(option.value, option.label);
    if (selectedOption) labels.current.set(selectedOption.value, selectedOption.label);
    if (draft.value !== value) {
      setDraft({ value, text: '', editing: false });
      setActive(null);
    }
    // Required checks the committed ID, including when an empty sentinel has a label.
    input.current?.setCustomValidity(mode === 'strict' && required && !value ? 'Выберите вариант из списка' : '');
  }, [options, selectedOption, value, draft.value, mode, required]);

  const close = () => { setOpen(false); setActive(null); };
  const pick = (option: AutocompleteOption) => {
    if (disabled || option.disabled) return;
    labels.current.set(option.value, option.label);
    setDraft({ value: option.value, text: '', editing: false });
    onChange(option.value);
    input.current?.focus();
    close();
  };

  return (
    <span className="kit-autocomplete">
      <span className="kit-autocomplete-control" ref={control}>
        <input
          id={inputId} ref={input} role="combobox" aria-label={ariaLabel}
          aria-autocomplete="list" aria-expanded={expanded} aria-controls={listId}
          aria-activedescendant={activeOption ? optionId(activeOption) : undefined}
          aria-busy={loading || undefined} autoComplete="off" required={required}
          disabled={disabled} placeholder={placeholder} value={text}
          onFocus={() => { if (!disabled) setOpen(true); }}
          onClick={() => { if (!disabled) setOpen(true); }}
          onBlur={() => { close(); onBlur?.(); }}
          onChange={(event) => {
            if (disabled) return;
            const next = event.target.value;
            const committed = mode === 'free' ? next : '';
            setDraft({ value: committed, text: next, editing: true });
            setOpen(true);
            setActive(null);
            onChange(committed);
            onQueryChange?.(next);
          }}
          onKeyDown={(event) => {
            if (disabled || event.nativeEvent.isComposing) return;
            if (event.key === 'ArrowDown' || event.key === 'ArrowUp') {
              event.preventDefault();
              setOpen(true);
              const index = enabled.findIndex((option) => option.value === active);
              const next = index < 0 ? (event.key === 'ArrowDown' ? 0 : enabled.length - 1)
                : (index + (event.key === 'ArrowDown' ? 1 : -1) + enabled.length) % enabled.length;
              setActive(enabled[next]?.value ?? null);
            } else if (event.key === 'Enter' && expanded) {
              event.preventDefault();
              if (activeOption) pick(activeOption);
            } else if (event.key === 'Escape' && expanded) {
              event.preventDefault();
              event.stopPropagation();
              close();
            } else if (event.key === 'Tab') close();
          }}
        />
        {allowClear && (value !== '' || editing && text !== '') && (
          <button type="button" tabIndex={-1} aria-label="Очистить" disabled={disabled}
            onMouseDown={(event) => event.preventDefault()}
            onClick={() => {
              setDraft({ value: '', text: '', editing: false });
              onChange('');
              onQueryChange?.('');
              setActive(null);
              input.current?.focus();
              setOpen(true);
            }}>×</button>
        )}
      </span>
      {expanded && (
        <span className="kit-autocomplete-popup" style={placement}>
          <ul role="listbox" id={listId} aria-label={ariaLabel}>
            {shown.map((option) => (
              <li key={option.value} id={optionId(option)} role="option"
                aria-selected={option.value === value} aria-disabled={option.disabled || undefined}
                data-active={option.value === active || undefined}
                onMouseDown={(event) => event.preventDefault()}
                onMouseEnter={() => setActive(option.disabled ? null : option.value)}
                onClick={() => pick(option)}>{option.label}</li>
            ))}
          </ul>
          {loading && <span role="status">Поиск…</span>}
          {error && <span role="alert">{error}{onRetry && <button type="button"
            onMouseDown={(event) => event.preventDefault()} onClick={onRetry}>Повторить</button>}</span>}
          {!loading && !error && shown.length === 0 && <span role="status">{emptyLabel}</span>}
        </span>
      )}
    </span>
  );
}
