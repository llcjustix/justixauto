import { Button, ChangePassword, Details, Page, Panel, useSession } from '@justixauto/kit';
import { useState } from 'react';
import type { NavItem } from '@justixauto/kit';
import { BillingPage } from '../realization/pages/billing';

function AccountantProfilePage() {
  const session = useSession();
  const [password, setPassword] = useState(false);
  return <Page title="Профиль" subtitle="Личный доступ к кабинету бухгалтерии">
    <Panel title="Профиль" padded>
      <Details items={[
        ['Имя', session.view.user.displayName],
        ['Статус', session.view.user.status === 'active' ? 'Активен' : session.view.user.status],
        ['Роли', session.view.roles.map((role) => role.name).join(', ') || '—'],
      ]} />
    </Panel>
    <Panel title="Безопасность" padded>
      <Button onClick={() => setPassword(true)}>Сменить пароль</Button>
      {password && <div style={{ marginTop: 16, maxWidth: 420 }}><ChangePassword onDone={() => {
        void session.refresh();
        setPassword(false);
      }} /></div>}
    </Panel>
  </Page>;
}

export const accountingNavigation: NavItem[] = [
  { to: '/accounting/billing', label: 'Счета и оплаты', icon: 'wallet', permission: ['retail.read', 'commerce.read'], element: <BillingPage /> },
  { to: '/accounting/profile', label: 'Профиль', icon: 'settings', bottom: true, element: <AccountantProfilePage /> },
];
