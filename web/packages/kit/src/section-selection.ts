import type { Company, SessionView } from './session';
import { permitted } from './shell';
import type { NavItem, ShellOptions } from './shell';

export type SectionId = 'realization' | 'financing' | 'insurance' | 'admin';

export type PortalSection = Omit<ShellOptions, 'rootId' | 'basename'> & {
  id: SectionId;
  prefix: '/' | '/finance' | '/insurance' | '/admin';
};

const sectionKinds: Record<Exclude<SectionId, 'admin'>, Company['kind'][]> = {
  realization: ['seller'],
  financing: ['bank', 'mfo'],
  insurance: ['insurance'],
};

export function matchesCompany(section: PortalSection, company: Company | undefined): boolean {
  if (section.id === 'admin') return true;
  return !!company && sectionKinds[section.id].includes(company.kind) && (!section.kinds || section.kinds.includes(company.kind));
}

export function matchingCompanies(section: PortalSection, view: SessionView): Company[] {
  return section.id === 'admin' ? [] : view.accessibleCompanies.filter((company) => matchesCompany(section, company));
}

export function allowedNav(section: PortalSection, view: SessionView, company: Company | undefined): NavItem[] {
  if (!matchesCompany(section, company)) return [];
  const can = (permission: string) => view.permissions.includes(permission);
  // Admin has no permission-free home. Its nav items must carry an actual platform grant.
  return section.nav.filter((item) => section.id === 'admin'
    ? !!item.permission && permitted(item.permission, can)
    : permitted(item.permission, can));
}

export function allowedSections(sections: PortalSection[], view: SessionView, company: Company | undefined): PortalSection[] {
  return sections.filter((section) => allowedNav(section, view, company).length > 0);
}

export function sectionForPath(sections: PortalSection[], pathname: string): PortalSection | undefined {
  const owned = sections.filter((section) => section.prefix !== '/' &&
    (pathname === section.prefix || pathname.startsWith(`${section.prefix}/`)));
  if (owned.length) return owned.sort((a, b) => b.prefix.length - a.prefix.length)[0];
  return pathname.startsWith('/') ? sections.find((section) => section.prefix === '/') : undefined;
}

export function navForPath(nav: NavItem[], pathname: string): NavItem | undefined {
  return nav.filter((item) => pathname === item.to || pathname.startsWith(`${item.to}/`))
    .sort((a, b) => b.to.length - a.to.length)[0];
}

export function sectionHome(section: PortalSection, view: SessionView, company: Company | undefined): string | undefined {
  return allowedNav(section, view, company)[0]?.to;
}

export function lastSectionKey(view: SessionView): string {
  return `justixauto:last-section:v1:${encodeURIComponent(view.user.id)}:${encodeURIComponent(view.context.companyId ?? 'platform')}`;
}

export function readLastSection(view: SessionView, available: PortalSection[], storage: Pick<Storage, 'getItem'> | undefined): PortalSection | undefined {
  try {
    const id = storage?.getItem(lastSectionKey(view));
    return available.find((section) => section.id === id);
  } catch {
    return undefined;
  }
}

export function rememberSection(view: SessionView, section: PortalSection, storage: Pick<Storage, 'setItem'> | undefined): void {
  try {
    storage?.setItem(lastSectionKey(view), section.id);
  } catch {
    // Browser storage is advisory; routing must still work when it is unavailable.
  }
}
