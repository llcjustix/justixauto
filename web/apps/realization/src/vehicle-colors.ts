import type { Model, Spec, VehicleColors } from './data';

/** An exact historical lookup. Missing versions and unpinned history stay unknown. */
export function exactSpecification(model: Model | undefined, version: string | undefined): Spec | undefined {
  return version ? model?.versions?.find(spec => spec.version === version) : undefined;
}

/** Scalars are compatibility inputs only; a multi-choice palette has no default. */
export function colorPalette(spec: Spec | undefined, kind: 'exterior' | 'interior'): string[] {
  if (!spec) return [];
  const palette = kind === 'exterior' ? spec.exteriorColors : spec.interiorColors;
  const scalar = kind === 'exterior' ? spec.exteriorColor : spec.interiorColor;
  return palette ?? (scalar?.trim() ? [scalar] : []);
}

export function colorOptions(spec: Spec | undefined, kind: 'exterior' | 'interior', required?: string): string[] {
  const palette = colorPalette(spec, kind);
  return required ? palette.filter(value => value.toLowerCase() === required.toLowerCase()) : palette;
}

/** Invalid/stale choices clear; only a singleton can select itself. */
export function selectedColor(options: string[], value?: string): string {
  return options.find(option => option.toLowerCase() === value?.toLowerCase()) ?? (options.length === 1 ? options[0]! : '');
}

export const colorLabel = (value: string | null | undefined): string => value?.trim() || 'Не указан';
export const vehicleColorsLabel = (facts: VehicleColors): string =>
  `Кузов: ${colorLabel(facts.exteriorColor)} · Салон: ${colorLabel(facts.interiorColor)} · Версия: ${colorLabel(facts.modelSpecificationVersion)}`;
