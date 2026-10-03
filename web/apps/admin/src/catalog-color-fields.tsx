import { useState } from 'react';
import type { ReactNode } from 'react';
import { Autocomplete, Button, Modal, Notice, errorText } from '@justixauto/kit';

export interface CatalogSpecification {
  make: string;
  model: string;
  variant: string;
  year: number;
  bodyType: string;
  exteriorColor: string;
  interiorColor: string;
  exteriorColors?: string[];
  interiorColors?: string[];
  powertrain: string;
  drivetrain: string;
  version: number;
}

export interface CatalogModel {
  id: string;
  specification: CatalogSpecification;
  revision: string;
}

const makes = ['BYD', 'Changan', 'Chery', 'Chevrolet', 'Exeed', 'Geely', 'Haval', 'Honda', 'Hongqi', 'Hyundai', 'Jetour', 'Kia', 'Lada', 'Leapmotor', 'Lexus', 'Li Auto', 'Mazda', 'Mercedes-Benz', 'BMW', 'Nissan', 'Ravon', 'Tesla', 'Toyota', 'Volkswagen', 'Voyah', 'Zeekr'].sort((a, b) => a.localeCompare(b));
const bodies = ['Седан', 'Хэтчбек', 'Лифтбек', 'Универсал', 'Кроссовер', 'Внедорожник', 'Минивэн', 'Купе', 'Кабриолет', 'Пикап', 'Фургон'];
const powertrains = ['Бензин', 'Дизель', 'Гибрид', 'Подключаемый гибрид', 'Электро', 'Газ / бензин'];
const drivetrains = ['Передний', 'Задний', 'Полный'];
const colors = ['Белый', 'Чёрный', 'Серый', 'Серебристый', 'Синий', 'Голубой', 'Красный', 'Бордовый', 'Бежевый', 'Коричневый', 'Зелёный', 'Жёлтый', 'Оранжевый'];
const thisYear = new Date().getFullYear();
const years = Array.from({ length: thisYear + 2 - 1990 }, (_, i) => String(thisYear + 1 - i));

const choices = (list: string[], current?: string) => (current && !list.includes(current) ? [current, ...list] : list);
const initialPalette = (palette: string[] | undefined, scalar: string) => palette?.length ? palette : scalar ? [scalar] : [];
const trimPalette = (palette: string[]) => {
  const seen = new Set<string>();
  return palette.reduce<string[]>((out, value) => {
    const trimmed = value.trim();
    const key = trimmed.toLocaleLowerCase();
    if (trimmed && !seen.has(key)) {
      seen.add(key);
      out.push(trimmed);
    }
    return out;
  }, []);
};

function Palette({ label, values, onChange }: { label: string; values: string[]; onChange: (next: string[]) => void }) {
  const [custom, setCustom] = useState('');
  const [error, setError] = useState('');
  const add = (value: string) => {
    const next = trimPalette([...values, value]);
    const trimmed = value.trim();
    if (!trimmed) return setError('Введите название цвета.');
    if (trimmed.length > 50) return setError('Название цвета должно быть не длиннее 50 символов.');
    if (next.length === values.length) return setError('Этот цвет уже выбран.');
    setError('');
    onChange(next);
    setCustom('');
  };
  const toggle = (color: string) => onChange(values.some((value) => value.toLocaleLowerCase() === color.toLocaleLowerCase()) ? values.filter((value) => value.toLocaleLowerCase() !== color.toLocaleLowerCase()) : trimPalette([...values, color]));
  return (
    <fieldset className="form-fieldset" style={{ padding: 16 }}>
      <legend>{label}</legend>
      <div className="kit-row" aria-label={`${label}: выбранные цвета`}>
        {values.map((color) => <span className="status status-neutral" key={color}>{color}<button type="button" aria-label={`Удалить ${color}`} onClick={() => onChange(values.filter((value) => value !== color))} style={{ marginLeft: 6, border: 0, background: 'transparent', cursor: 'pointer' }}>×</button></span>)}
        {!values.length && <span className="cell-sub">Выберите хотя бы один цвет.</span>}
      </div>
      <div className="kit-grid" style={{ marginTop: 12 }}>
        {colors.map((color) => <label className="kit-row" key={color}><input type="checkbox" checked={values.some((value) => value.toLocaleLowerCase() === color.toLocaleLowerCase())} onChange={() => toggle(color)} />{color}</label>)}
      </div>
      <div className="kit-row" style={{ marginTop: 12 }}>
        <label className="kit-field" style={{ flex: '1 1 220px' }}>Свой цвет<input value={custom} onChange={(event) => setCustom(event.target.value)} placeholder="Например, Песочный металлик" /></label>
        <Button onClick={() => add(custom)}>Добавить цвет</Button>
      </div>
      {error && <div className="feedback-danger" role="alert" style={{ marginTop: 8 }}>{error}</div>}
    </fieldset>
  );
}

export function CatalogColorFields({ title, submitLabel, models, specification, onClose, onSubmit, intro }: {
  title: string;
  submitLabel: string;
  models: CatalogModel[];
  specification?: CatalogSpecification;
  onClose: () => void;
  onSubmit: (body: { specification: Record<string, unknown> }) => Promise<unknown>;
  intro?: ReactNode;
}) {
  const [values, setValues] = useState(() => ({ make: specification?.make ?? '', model: specification?.model ?? '', variant: specification?.variant ?? '', year: String(specification?.year ?? thisYear), bodyType: specification?.bodyType ?? '', powertrain: specification?.powertrain ?? '', drivetrain: specification?.drivetrain ?? '' }));
  const [exteriorColors, setExteriorColors] = useState(() => trimPalette(initialPalette(specification?.exteriorColors, specification?.exteriorColor ?? '')));
  const [interiorColors, setInteriorColors] = useState(() => trimPalette(initialPalette(specification?.interiorColors, specification?.interiorColor ?? '')));
  const [error, setError] = useState('');
  const [busy, setBusy] = useState(false);
  const modelOptions = [...new Set(models.filter((model) => model.specification.make.toLocaleLowerCase() === values.make.trim().toLocaleLowerCase()).map((model) => model.specification.model))];
  const makeOptions = [...new Set([...makes, ...models.map((model) => model.specification.make)])].sort((a, b) => a.localeCompare(b));
  const set = (name: keyof typeof values, value: string) => setValues((current) => ({ ...current, [name]: value, ...(name === 'make' && current.make !== value ? { model: '' } : {}) }));
  async function submit() {
    const body = { ...values, exteriorColors: trimPalette(exteriorColors), interiorColors: trimPalette(interiorColors), year: Number(values.year) };
    if (Object.values(values).some((value) => !value.trim()) || !Number.isFinite(body.year)) return setError('Заполните все характеристики модели.');
    if (!body.exteriorColors.length || !body.interiorColors.length) return setError('Выберите хотя бы один цвет кузова и салона.');
    setBusy(true); setError('');
    try { await onSubmit({ specification: body }); onClose(); } catch (cause) { setError(errorText(cause)); } finally { setBusy(false); }
  }
  return <Modal title={title} onClose={onClose} size="wide" footer={<><Button onClick={onClose} disabled={busy}>Отмена</Button><Button variant="primary" busy={busy} onClick={() => void submit()}>{submitLabel}</Button></>}>
    {intro}
    {error && <Notice kind="danger">{error}</Notice>}
    <form className="form-grid" onSubmit={(event) => { event.preventDefault(); void submit(); }}>
      <label className="kit-field">Марка<Autocomplete aria-label="Марка" mode="free" options={makeOptions.map((value) => ({ value, label: value }))} value={values.make} onChange={(value) => set('make', value)} required placeholder="Выберите или введите марку" /></label>
      <label className="kit-field">Модель<Autocomplete aria-label="Модель" mode="free" options={modelOptions.map((value) => ({ value, label: value }))} value={values.model} onChange={(value) => set('model', value)} required disabled={!values.make.trim()} placeholder={values.make.trim() ? 'Выберите или введите модель' : 'Сначала выберите марку'} /></label>
      <label className="kit-field">Комплектация<input value={values.variant} onChange={(event) => set('variant', event.target.value)} required /></label>
      <label className="kit-field">Год<Autocomplete aria-label="Год" options={choices(years, values.year).map((value) => ({ value, label: value }))} value={values.year} onChange={(value) => set('year', value)} required /></label>
      <label className="kit-field">Кузов<Autocomplete aria-label="Кузов" options={[{ value: '', label: '—' }, ...choices(bodies, values.bodyType).map((value) => ({ value, label: value }))]} value={values.bodyType} onChange={(value) => set('bodyType', value)} required /></label>
      <label className="kit-field">Двигатель<Autocomplete aria-label="Двигатель" options={[{ value: '', label: '—' }, ...choices(powertrains, values.powertrain).map((value) => ({ value, label: value }))]} value={values.powertrain} onChange={(value) => set('powertrain', value)} required /></label>
      <label className="kit-field">Привод<Autocomplete aria-label="Привод" options={[{ value: '', label: '—' }, ...choices(drivetrains, values.drivetrain).map((value) => ({ value, label: value }))]} value={values.drivetrain} onChange={(value) => set('drivetrain', value)} required /></label>
      <Palette label="Цвет кузова" values={exteriorColors} onChange={setExteriorColors} />
      <Palette label="Цвет салона" values={interiorColors} onChange={setInteriorColors} />
    </form>
  </Modal>;
}
