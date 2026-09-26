import { list, patch, post } from './http';
import { PERM_COMPANY_USERS } from './company-employees';
import { permissionNames, permissionOptions } from './permissions';
import type { Permission } from './permissions';
import { useSession } from './session';
import { ActionButton, useData } from './shell';
import { Cell, Panel, Table } from './ui';
import type { FieldSpec } from './ui';

interface CompanyRole {
  id: string;
  name: string;
  system: boolean;
  permissionKeys: string[];
  revision: string;
}

/**
 * The company's own roles (user decisions 2026-09-26): a company admin creates
 * private roles from the company permissions kept in PostgreSQL and gives
 * them to employees. The built-in company administrator is listed read-only.
 */
export function CompanyRoles() {
  const s = useSession();
  const id = s.company?.id ?? '';
  const manage = !!id && s.can(PERM_COMPANY_USERS);
  const base = `/identity/companies/${id}`;
  const roles = useData(['company-roles', id], () => list<CompanyRole>(`${base}/roles`), manage);
  const perms = useData(['company-permissions', id], () => list<Permission>(`${base}/permissions`), manage);
  const refresh = [['company-roles', id]];
  const options = permissionOptions(perms.data ?? []);
  const fields = (r?: CompanyRole): FieldSpec[] => [
    { name: 'name', label: 'Название', type: 'text', required: true, full: true, initial: r?.name ?? '' },
    { name: 'permissionKeys', label: 'Разрешения', type: 'multiselect', options, initial: r?.permissionKeys ?? [] },
  ];
  if (!manage) return null;
  return (
    <Panel
      title="Роли"
      actions={
        <ActionButton
          label="+ Создать роль"
          title="Новая роль"
          submitLabel="Создать"
          variant="primary"
          size="wide"
          refresh={refresh}
          fields={fields()}
          onSubmit={(v) => post(`${base}/roles`, v)}
        />
      }
    >
      <Table
        rows={roles.data}
        loading={roles.isLoading}
        error={roles.error}
        rowKey={(r) => r.id}
        empty="Ролей пока нет"
        columns={[
          { title: 'Роль', render: (r) => <Cell main={r.name} sub={r.system ? 'Встроенная' : undefined} /> },
          {
            title: 'Разрешения',
            render: (r) =>
              r.system ? 'Все разрешения компании' : permissionNames(perms.data ?? [], r.permissionKeys) || '—',
          },
          {
            title: '',
            render: (r) =>
              !r.system && (
                <ActionButton
                  small
                  label="Изменить"
                  title="Изменить роль"
                  submitLabel="Сохранить"
                  size="wide"
                  refresh={refresh}
                  fields={fields(r)}
                  onSubmit={(v) => patch(`${base}/roles/${r.id}`, v, { ifMatch: r.revision })}
                />
              ),
          },
        ]}
      />
    </Panel>
  );
}
