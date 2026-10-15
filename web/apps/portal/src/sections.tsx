import type { PortalSection } from '@justixauto/kit';
import {
  AuditPage,
  CatalogPage,
  CompaniesPage,
  OverviewPage as AdminOverviewPage,
  PermissionsPage,
  RolesPage,
  UsersPage,
} from './features/admin/pages';
import { navigation as financingNavigation } from './features/financing/navigation';
import { navigation as insuranceNavigation } from './features/insurance/navigation';
import { accountingNavigation } from './features/accounting/navigation';
import { navigation as realizationNavigation } from './features/realization/navigation';

const accountantPermissions = [
  'commerce.invoices.manage',
  'commerce.payments.accept',
  'commerce.read',
  'documents.read',
  'documents.sensitive.download',
  'documents.upload',
  'retail.invoices.manage',
  'retail.payments.accept',
  'retail.read',
];
const isPureAccountant = (permissions: string[]) =>
  permissions.length === accountantPermissions.length &&
  accountantPermissions.every((key) => permissions.includes(key));

export const sections: PortalSection[] = [
  {
    id: 'realization',
    prefix: '/',
    brand: 'Реализация',
    kinds: ['seller'],
    nav: realizationNavigation,
    branches: true,
    shellClass: 'dealer-shell',
    access: (view) => !isPureAccountant(view.permissions),
    search: { placeholder: 'VIN, модель или клиент', path: (q) => `/vehicles?q=${encodeURIComponent(q)}` },
  },
  {
    id: 'accounting',
    prefix: '/accounting',
    brand: 'Бухгалтерия',
    kinds: ['seller'],
    nav: accountingNavigation,
    access: (view) => {
      const has = (key: string) => view.permissions.includes(key);
      return (
        (has('retail.read') && (has('retail.invoices.manage') || has('retail.payments.accept'))) ||
        (has('commerce.read') && (has('commerce.invoices.manage') || has('commerce.payments.accept')))
      );
    },
    preferred: (view) => isPureAccountant(view.permissions),
  },
  { id: 'financing', prefix: '/finance', brand: 'Исламская линия', kinds: ['bank', 'mfo'], nav: financingNavigation },
  {
    id: 'insurance',
    prefix: '/insurance',
    brand: 'Страховая компания',
    kinds: ['insurance'],
    nav: insuranceNavigation,
    variant: 'insurance',
  },
  {
    id: 'admin',
    prefix: '/admin',
    brand: 'Администрирование',
    variant: 'admin',
    nav: [
      { to: '/admin/overview', label: 'Обзор', permission: 'platform.directory.read', element: <AdminOverviewPage /> },
      {
        to: '/admin/companies',
        label: 'Компании',
        permission: 'platform.directory.read',
        element: <CompaniesPage kind="seller" />,
      },
      {
        to: '/admin/catalog',
        label: 'Каталог автомобилей',
        permission: 'platform.catalog.manage',
        element: <CatalogPage />,
      },
      {
        to: '/admin/users',
        label: 'Сотрудники платформы',
        permission: 'platform.users.manage',
        element: <UsersPage />,
      },
      { to: '/admin/roles', label: 'Роли', permission: 'platform.roles.manage', element: <RolesPage /> },
      {
        to: '/admin/permissions',
        label: 'Разрешения',
        permission: 'platform.roles.manage',
        element: <PermissionsPage />,
      },
      {
        to: '/admin/integrations/mfo',
        label: 'МФО',
        group: 'Интеграции',
        permission: 'platform.directory.read',
        element: <CompaniesPage kind="mfo" />,
      },
      {
        to: '/admin/integrations/bank',
        label: 'Банки',
        group: 'Интеграции',
        permission: 'platform.directory.read',
        element: <CompaniesPage kind="bank" />,
      },
      {
        to: '/admin/integrations/insurance',
        label: 'Страховые компании',
        group: 'Интеграции',
        permission: 'platform.directory.read',
        element: <CompaniesPage kind="insurance" />,
      },
      {
        to: '/admin/audit',
        label: 'Журнал действий',
        bottom: true,
        permission: 'platform.audit.read',
        element: <AuditPage />,
      },
    ],
  },
];
