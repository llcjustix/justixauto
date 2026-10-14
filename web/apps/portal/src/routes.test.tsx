import { describe, expect, it } from 'vitest';
import { sections } from './sections';

const section = (id: string) => sections.find((candidate) => candidate.id === id)!;

describe('portal sections', () => {
  it('keeps the seller surface at its stable URLs and search destination', () => {
    const seller = section('realization');
    expect(seller.nav.find((item) => item.to === '/dashboard')).toBeDefined();
    expect(seller.nav.find((item) => item.to === '/billing')?.permission).toEqual(['retail.read', 'commerce.read']);
    expect(seller.search?.path('A B')).toBe('/vehicles?q=A%20B');
    expect(seller.nav.find((item) => item.to === '/insurance-applications')).toBeUndefined();
  });

  it('keeps accounting separate while reusing the billing route surface', () => {
    const accounting = section('accounting');
    expect(accounting.prefix).toBe('/accounting');
    expect(accounting.nav.map((item) => item.to)).toEqual(['/accounting/billing', '/accounting/profile']);
  });

  it('admits accounting only with seller read plus financial authority', () => {
    const accounting = section('accounting');
    const company = { id: 'seller', name: 'Seller', kind: 'seller' as const, access: 'full' as const, userType: 'dealer' as const, verification: 'unverified' as const, verificationReason: '' };
    const view = (permissions: string[]) => ({ permissions } as any);
    expect(accounting.access!(view(['retail.read', 'retail.invoices.manage']), company)).toBe(true);
    expect(accounting.access!(view(['commerce.read', 'commerce.payments.accept']), company)).toBe(true);
    expect(accounting.access!(view(['retail.read']), company)).toBe(false);
    expect(accounting.access!(view(['retail.invoices.manage']), company)).toBe(false);
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
