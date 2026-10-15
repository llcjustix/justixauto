export type LegalForm = '' | 'llc' | 'sole_proprietor' | 'other';

export const legalFormOptions: [string, string][] = [
  ['', 'Не указана'],
  ['llc', 'ООО'],
  ['sole_proprietor', 'ИП'],
  ['other', 'Другое'],
];

export function legalFormLabel(value: string | null | undefined): string {
  return legalFormOptions.find(([key]) => key === value)?.[1] ?? 'Не указана';
}
