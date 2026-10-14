import { useEffect, useRef, useState } from 'react';
import type { ReactNode } from 'react';
import { createRoot } from 'react-dom/client';
import { QueryClient, QueryClientProvider } from '@tanstack/react-query';
import { BrowserRouter, NavLink, Navigate, useLocation, useNavigate } from 'react-router-dom';
import { errorText } from './http';
import { SessionGate, useSession } from './session';
import type { Company } from './session';
import { Shell } from './shell';
import { allowedNav, allowedSections, matchingCompanies, navForPath, readLastSection, rememberSection, sectionForPath, sectionHome } from './section-selection';
import type { PortalSection, SectionId } from './section-selection';
import { Button, Card, KitStyles, Notice, css } from './ui';

export type { PortalSection, SectionId } from './section-selection';

function browserStorage(): Storage | undefined {
  try { return window.localStorage; } catch { return undefined; }
}

function SelectionScreen({
  title, children, available,
}: {
  title: string;
  children: ReactNode;
  available: { id: SectionId; label: string; home: string }[];
}) {
  return <div className={css.centered}>
    <Card title={title}>
      {children}
      {available.length > 0 && <nav aria-label="Доступные разделы" className={css.stack}>
        {available.map((item) => <NavLink key={item.id} to={item.home}>{item.label}</NavLink>)}
      </nav>}
    </Card>
  </div>;
}

function CompanySelection({ candidates, available, preferredId }: {
  candidates: Company[];
  available: { id: SectionId; label: string; home: string }[];
  preferredId?: string;
}) {
  const session = useSession();
  const [error, setError] = useState('');
  const attempted = useRef(false);
  const automatic = candidates.find((company) => company.id === preferredId)
    ?? (candidates.length === 1 ? candidates[0] : undefined);
  useEffect(() => {
    if (!automatic || attempted.current) return;
    attempted.current = true;
    void session.selectCompany(automatic.id).catch((cause) => setError(errorText(cause)));
  }, [automatic, session]);
  return <SelectionScreen title={candidates.length ? 'Выберите компанию' : 'Нет доступа'} available={available}>
    {error && <Notice kind="danger">{error}</Notice>}
    {automatic && !error && <p>Открываем компанию {automatic.name}…</p>}
    {candidates.length === 0 && <p>Нет подходящей компании для этого раздела.</p>}
    {(!automatic || error) && <div className={css.stack}>
      {candidates.map((company) => <Button key={company.id} onClick={() => {
        setError('');
        void session.selectCompany(company.id).catch((cause) => setError(errorText(cause)));
      }}>{company.name}</Button>)}
    </div>}
  </SelectionScreen>;
}

/** Routes inside the one admitted portal session and root browser router. */
export function PortalRoutes({ sections }: { sections: PortalSection[] }): ReactNode {
  const session = useSession();
  const location = useLocation();
  const navigate = useNavigate();
  const available = allowedSections(sections, session.view, session.company);
  const destinations = available.flatMap((section) => {
    const home = sectionHome(section, session.view, session.company);
    return home ? [{ id: section.id, label: section.brand, home }] : [];
  });
  const pathSection = sectionForPath(sections, location.pathname);
  const section = location.pathname === '/'
    ? available.find((candidate) => candidate.preferred?.(session.view, session.company))
      ?? readLastSection(session.view, available, browserStorage()) ?? available[0]
    : pathSection;
  const admittedNav = section ? allowedNav(section, session.view, session.company) : [];
  const admittedPath = section && location.pathname !== '/' && navForPath(admittedNav, location.pathname);
  useEffect(() => {
    if (section && admittedPath) rememberSection(session.view, section, browserStorage());
  }, [session.view, section, admittedPath]);

  if (location.pathname === '/' && section) {
    const home = sectionHome(section, session.view, session.company);
    if (home) return <Navigate to={{ pathname: home, search: location.search, hash: location.hash }} replace />;
  }

  if (!section) {
    const membershipSections = sections.filter((candidate) => candidate.id !== 'admin' && matchingCompanies(candidate, session.view).length > 0);
    const candidates = membershipSections.flatMap((candidate) => matchingCompanies(candidate, session.view));
    const unique = candidates.filter((company, index) => candidates.findIndex((other) => other.id === company.id) === index);
    return <CompanySelection candidates={unique} available={destinations} />;
  }

  if (!available.includes(section)) {
    const candidates = matchingCompanies(section, session.view);
    const preferredId = typeof location.state === 'object' && location.state !== null && 'portalCompanyId' in location.state
      ? String(location.state.portalCompanyId) : undefined;
    const requestedOtherCompany = !!preferredId && preferredId !== session.company?.id && candidates.some((company) => company.id === preferredId);
    if (section.id !== 'admin' && candidates.length > 0 && (requestedOtherCompany || !candidates.some((company) => company.id === session.company?.id))) {
      return <CompanySelection candidates={candidates} available={destinations} preferredId={preferredId} />;
    }
    return <Navigate to="/" replace />;
  }

  const home = sectionHome(section, session.view, session.company)!;
  if (location.pathname === section.prefix || !admittedPath) {
    return <Navigate to={{ pathname: home, search: location.pathname === section.prefix ? location.search : '', hash: location.pathname === section.prefix ? location.hash : '' }} replace />;
  }

  const displayedDestinations = destinations.filter((destination) =>
    !(section.id === 'realization' && destination.id === 'accounting')
    && !(section.id === 'accounting' && destination.id === 'realization'));
  const companyChoices = sections.filter((candidate) => candidate.id !== 'admin' && !available.includes(candidate)).flatMap((candidate) =>
    matchingCompanies(candidate, session.view).filter((company) => company.id !== session.company?.id).map((company) => ({ candidate, company })));
  const chooser = displayedDestinations.length > 1 || companyChoices.length > 0 ? <nav className="portal-switcher" aria-label="Разделы">
    <div className="popover-title">Разделы</div>
    {displayedDestinations.map((destination) => <NavLink key={destination.id} to={destination.home} className={destination.id === section.id ? 'portal-switcher-item active' : 'portal-switcher-item'}>
      {destination.label}
    </NavLink>)}
    {companyChoices.map(({ candidate, company }) => <button
      key={`${candidate.id}:${company.id}`}
      type="button"
      className="portal-switcher-item"
      onClick={() => navigate(candidate.prefix, { state: { portalCompanyId: company.id } })}
    >Выбрать компанию: {company.name} · {candidate.brand}</button>)}
  </nav> : undefined;
  return <Shell key={`${session.scopeKey}:${section.id}`} {...section} rootId="portal-root" basename="/" sectionSwitcher={chooser} />;
}

/** Mounts exactly one QueryClient, login gate and root BrowserRouter. */
export function mountPortal({ rootId, sections }: { rootId: string; sections: PortalSection[] }): void {
  const client = new QueryClient({ defaultOptions: { queries: { retry: 1, refetchOnWindowFocus: true } } });
  createRoot(document.getElementById(rootId)!).render(
    <QueryClientProvider client={client}>
      <KitStyles />
      <SessionGate title="JustixAuto" brand="JustixAuto">
        <BrowserRouter><PortalRoutes sections={sections} /></BrowserRouter>
      </SessionGate>
    </QueryClientProvider>,
  );
}
