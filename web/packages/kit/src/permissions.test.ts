import { describe, expect, it } from 'vitest';
import { permissionNames, permissionOptions } from './permissions';

const catalog = [
  { key: 'retail.read', scope: 'company', name: 'Продажи: просмотр', assignable: true },
  { key: 'inventory.read', scope: 'company', name: 'Склад: просмотр', assignable: true },
  { key: 'platform.users.manage', scope: 'platform', name: 'Платформа: сотрудники', assignable: false },
];

describe('permission catalog helpers', () => {
  it('offers assignable permissions sorted by their database name', () => {
    expect(permissionOptions(catalog)).toEqual([
      ['retail.read', 'Продажи: просмотр'],
      ['inventory.read', 'Склад: просмотр'],
    ]);
  });

  it('names keys from the catalog and shows unknown keys as is', () => {
    expect(permissionNames(catalog, ['inventory.read', 'future.key'])).toBe('Склад: просмотр, future.key');
  });
});
