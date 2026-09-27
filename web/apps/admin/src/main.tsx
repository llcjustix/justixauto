import { mountApp } from '@justixauto/kit';
import { AuditPage, CatalogPage, CompaniesPage, OverviewPage, PermissionsPage, RolesPage, UsersPage } from './pages';
import './design.css';

mountApp({
  rootId: 'admin-root',
  basename: '/admin',
  brand: 'Администрирование',
  variant: 'admin',
  nav: [
    { to: '/overview', label: 'Обзор', permission: 'platform.directory.read', element: <OverviewPage /> },
    {
      to: '/companies',
      label: 'Компании',
      permission: 'platform.directory.read',
      element: <CompaniesPage kind="seller" />,
    },
    { to: '/catalog', label: 'Каталог автомобилей', permission: 'platform.catalog.manage', element: <CatalogPage /> },
    { to: '/users', label: 'Сотрудники платформы', permission: 'platform.users.manage', element: <UsersPage /> },
    { to: '/roles', label: 'Роли', permission: 'platform.roles.manage', element: <RolesPage /> },
    { to: '/permissions', label: 'Разрешения', permission: 'platform.roles.manage', element: <PermissionsPage /> },
    {
      to: '/integrations/mfo',
      label: 'МФО',
      group: 'Интеграции',
      permission: 'platform.directory.read',
      element: <CompaniesPage kind="mfo" />,
    },
    {
      to: '/integrations/bank',
      label: 'Банки',
      group: 'Интеграции',
      permission: 'platform.directory.read',
      element: <CompaniesPage kind="bank" />,
    },
    {
      to: '/integrations/insurance',
      label: 'Страховые компании',
      group: 'Интеграции',
      permission: 'platform.directory.read',
      element: <CompaniesPage kind="insurance" />,
    },
    { to: '/audit', label: 'Журнал действий', bottom: true, permission: 'platform.audit.read', element: <AuditPage /> },
  ],
});
