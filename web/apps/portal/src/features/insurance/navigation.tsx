import { OrgSettings } from '@justixauto/kit';
import type { NavItem } from '@justixauto/kit';
import { ApplicationsPage, OverviewPage } from './pages';

export const navigation: NavItem[] = [
  { to: '/insurance/overview', label: 'Обзор', permission: 'insurance.read', element: <OverviewPage /> },
  { to: '/insurance/applications', label: 'Заявки', permission: 'insurance.read', element: <ApplicationsPage /> },
  { to: '/insurance/settings', label: 'Настройки', element: <OrgSettings /> },
];
