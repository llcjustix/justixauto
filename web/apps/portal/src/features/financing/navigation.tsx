import { OrgSettings } from '@justixauto/kit';
import type { NavItem } from '@justixauto/kit';
import { ApplicationsPage, OverviewPage, PartnersPage, ProgramsPage } from './pages';

export const navigation: NavItem[] = [
  { to: '/finance/overview', label: 'Обзор', group: 'Работа', permission: 'financing.read', element: <OverviewPage /> },
  {
    to: '/finance/applications',
    label: 'Заявки',
    group: 'Работа',
    permission: 'financing.read',
    element: <ApplicationsPage />,
  },
  { to: '/finance/programs', label: 'Программы', group: 'Работа', permission: 'financing.read', element: <ProgramsPage /> },
  { to: '/finance/partners', label: 'Партнёры', group: 'Работа', permission: 'financing.read', element: <PartnersPage /> },
  { to: '/finance/settings', label: 'Настройки', group: 'Работа', element: <OrgSettings /> },
];
