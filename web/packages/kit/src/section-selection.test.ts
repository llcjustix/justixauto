import { describe, expect, it } from 'vitest';
import { allowedNav, allowedSections, lastSectionKey, matchingCompanies, navForPath, readLastSection, rememberSection, sectionForPath, sectionHome } from './section-selection';
import type { PortalSection } from './section-selection';
import type { Company, SessionView } from './session';

const company = (id: string, kind: Company['kind']): Company => ({
  id, name: id, kind, access: 'full', userType: kind === 'seller' ? 'dealer' : null,
  verification: 'unverified', verificationReason: '',
});
const seller = company('seller', 'seller');
const bank = company('bank', 'bank');
const insurer = company('insurer', 'insurance');
const view = (active: Company | undefined, permissions: string[] = [], companies = [seller, bank, insurer]): SessionView => ({
  user: { id: 'user:1', displayName: 'User', status: 'active', passwordChangeRequired: false },
  roles: [], permissions,
  context: { revision: 'r1', companyId: active?.id ?? null, branchScope: { mode: 'ALL', branchIds: [] } },
  accessibleCompanies: companies, setup: { next: 'none' },
});
const node = null;
const sections: PortalSection[] = [
  { id: 'realization', prefix: '/', brand: 'Реализация', kinds: ['seller'], nav: [
    { to: '/dashboard', label: 'Dashboard', element: node },
    { to: '/billing', label: 'Billing', permission: ['retail.read', 'commerce.read'], element: node },
    { to: '/settings', label: 'Settings', element: node },
  ] },
  { id: 'financing', prefix: '/finance', brand: 'Финансы', kinds: ['bank', 'mfo'], nav: [
    { to: '/finance/overview', label: 'Overview', permission: 'financing.read', element: node },
    { to: '/finance/settings', label: 'Settings', element: node },
  ] },
  { id: 'insurance', prefix: '/insurance', brand: 'Страхование', kinds: ['insurance'], nav: [
    { to: '/insurance/overview', label: 'Overview', permission: 'insurance.read', element: node },
    { to: '/insurance/settings', label: 'Settings', element: node },
  ] },
  { id: 'admin', prefix: '/admin', brand: 'Админ', nav: [
    { to: '/admin/overview', label: 'Overview', permission: 'platform.directory.read', element: node },
  ] },
];

describe('section admission from effective grants and active kind', () => {
  it('admits seller dealer/distributor homes and accountant OR navigation even when unverified', () => {
    const distributor = { ...seller, userType: 'distributor' as const };
    for (const active of [seller, distributor]) {
      const session = view(active, ['retail.read']);
      expect(allowedSections(sections, session, active).map((s) => s.id)).toEqual(['realization']);
      expect(allowedNav(sections[0]!, session, active).map((n) => n.to)).toEqual(['/dashboard', '/billing', '/settings']);
    }
  });

  it('keeps unguarded settings within matching bank/MFO or insurer kind', () => {
    const mfo = company('mfo', 'mfo');
    expect(sectionHome(sections[1]!, view(mfo), mfo)).toBe('/finance/settings');
    expect(sectionHome(sections[2]!, view(insurer), insurer)).toBe('/insurance/settings');
    expect(allowedNav(sections[1]!, view(seller, ['financing.read']), seller)).toEqual([]);
    expect(allowedNav(sections[0]!, view(bank, ['retail.read']), bank)).toEqual([]);
  });

  it('admits Admin on a platform grant without company and rejects it with no grants', () => {
    expect(allowedSections(sections, view(undefined, ['platform.directory.read']), undefined).map((s) => s.id)).toEqual(['admin']);
    expect(allowedSections(sections, view(undefined), undefined)).toEqual([]);
    expect(allowedSections(sections, view(bank, ['platform.directory.read']), bank).map((s) => s.id)).toEqual(['financing', 'admin']);
  });

  it('treats inactive memberships as context candidates, never current grants', () => {
    const session = view(seller, ['financing.read']);
    expect(allowedSections(sections, session, seller).map((s) => s.id)).toEqual(['realization']);
    expect(matchingCompanies(sections[1]!, session).map((c) => c.id)).toEqual(['bank']);
    expect(matchingCompanies(sections[1]!, view(seller, [], [seller]))).toEqual([]);
    expect(matchingCompanies(sections[1]!, view(seller, [], [seller, bank, company('mfo', 'mfo')]))).toHaveLength(2);
  });

  it('selects the longest owned prefix and only a granted nav route', () => {
    expect(sectionForPath(sections, '/admin/integrations/mfo')?.id).toBe('admin');
    expect(sectionForPath(sections, '/insurance/overview')?.id).toBe('insurance');
    expect(sectionForPath(sections, '/dashboard')?.id).toBe('realization');
    expect(navForPath(allowedNav(sections[0]!, view(seller), seller), '/billing/123')).toBeUndefined();
    expect(navForPath(allowedNav(sections[0]!, view(seller, ['commerce.read']), seller), '/billing/123')?.to).toBe('/billing');
  });
});

describe('advisory last section', () => {
  it('keys by actual user and context, validates revocation and ignores bad storage', () => {
    const session = view(bank, ['platform.directory.read']);
    const available = allowedSections(sections, session, bank);
    const values = new Map<string, string>();
    const storage = { getItem: (key: string) => values.get(key) ?? null, setItem: (key: string, value: string) => { values.set(key, value); } };
    rememberSection(session, sections[3]!, storage);
    expect(values.get(lastSectionKey(session))).toBe('admin');
    expect(readLastSection(session, available, storage)?.id).toBe('admin');
    expect(readLastSection(view(undefined, ['platform.directory.read']), available, storage)).toBeUndefined();
    expect(readLastSection(session, [sections[1]!], storage)).toBeUndefined();
    storage.setItem(lastSectionKey(session), 'unknown');
    expect(readLastSection(session, available, storage)).toBeUndefined();
    expect(readLastSection(session, available, { getItem: () => { throw new Error('unavailable'); } })).toBeUndefined();
    expect(() => rememberSection(session, sections[1]!, { setItem: () => { throw new Error('unavailable'); } })).not.toThrow();
  });
});
