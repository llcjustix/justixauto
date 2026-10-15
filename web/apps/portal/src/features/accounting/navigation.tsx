import { AccountantProfilePage } from './profile-page';
import type { NavItem } from '@justixauto/kit';
import { BillingPage } from '../realization/pages/billing';
import { FinanceWorkspace } from './finance-workspace';

export const accountingNavigation: NavItem[] = [
  {
    to: '/accounting/billing',
    label: 'Счета и оплаты',
    icon: 'wallet',
    permission: ['retail.read', 'commerce.read'],
    element: (
      <FinanceWorkspace>
        <BillingPage />
      </FinanceWorkspace>
    ),
  },
  { to: '/accounting/profile', label: 'Профиль', icon: 'settings', bottom: true, element: <AccountantProfilePage /> },
];
