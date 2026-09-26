/** One permission of the catalog kept in PostgreSQL (identity.permissions). */
export interface Permission {
  key: string;
  scope: string;
  name: string;
  assignable: boolean;
}

/** Picker options for assignable permissions, sorted by name. */
export const permissionOptions = (catalog: Permission[]): [string, string][] =>
  catalog
    .filter((p) => p.assignable)
    .map((p): [string, string] => [p.key, p.name || p.key])
    .sort((a, b) => a[1].localeCompare(b[1], 'ru'));

/** The names of keys, comma-separated; keys missing from the catalog are shown as is. */
export function permissionNames(catalog: Permission[], keys: string[]): string {
  const names = new Map(catalog.map((p) => [p.key, p.name]));
  return keys.map((k) => names.get(k) || k).join(', ');
}
