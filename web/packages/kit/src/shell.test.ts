import { describe, expect, it } from 'vitest';
import { permitted } from './shell';

describe('shell navigation permissions', () => {
  it('admits a navigation item when any declared company permission is present', () => {
    expect(permitted(['retail.read', 'commerce.read'], (key) => key === 'commerce.read')).toBe(true);
    expect(permitted(['retail.read', 'commerce.read'], () => false)).toBe(false);
    expect(permitted('retail.read', (key) => key === 'retail.read')).toBe(true);
  });
});
