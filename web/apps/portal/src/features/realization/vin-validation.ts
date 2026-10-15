export const normalizeVin = (value: string) => value.trim().replace(/[a-z]/g, (letter) => letter.toUpperCase());

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
  if (vins.length > cap)
    errors.push(
      `VIN больше допустимого: ${vins.length} из ${cap}. Удалите лишние строки или отправьте отдельной партией.`,
    );
  if (rowErrors.some(Boolean)) errors.push('Исправьте неверные или повторяющиеся VIN');
  return { vins, rowErrors, errors, valid: errors.length === 0 };
}
