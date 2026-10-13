import { describe, expect, it } from 'vitest';
import { sections } from './sections';

const section = (id: string) => sections.find((candidate) => candidate.id === id)!;

describe('portal sections', () => {
  it('keeps the seller surface at its stable URLs and search destination', () => {
    const seller = section('realization');
    expect(seller.nav.find((item) => item.to === '/dashboard')).toBeDefined();
    expect(seller.nav.find((item) => item.to === '/billing')?.permission).toEqual(['retail.read', 'commerce.read']);
    expect(seller.search?.path('A B')).toBe('/vehicles?q=A%20B');
  });

  it('uses the final absolute provider and admin browser URLs', () => {
    expect(section('financing').nav.map((item) => item.to)).toEqual([
      '/finance/overview', '/finance/applications', '/finance/programs', '/finance/partners', '/finance/settings',
    ]);
    expect(section('insurance').nav.map((item) => item.to)).toEqual([
      '/insurance/overview', '/insurance/applications', '/insurance/settings',
    ]);
    expect(section('admin').nav.map((item) => item.to)).toContain('/admin/audit');
    expect(section('admin').nav.find((item) => item.to === '/admin/companies')?.permission).toBe('platform.directory.read');
  });
});
