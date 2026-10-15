import { errorText, list, patch, post } from './http';
import { useSession } from './session';
import { ActionButton, useData } from './shell';
import { Badge, Cell, Notice, Panel, Table } from './ui';
import type { FieldSpec } from './ui';

/** Permission that lets a company admin manage the company's employees. */
export const PERM_COMPANY_USERS = 'company.users.manage';
export const ACCOUNTANT_PERMISSION_KEYS = [
  'commerce.invoices.manage',
  'commerce.payments.accept',
  'commerce.read',
  'documents.read',
  'documents.sensitive.download',
  'documents.upload',
  'retail.invoices.manage',
  'retail.payments.accept',
  'retail.read',
] as const;

interface Employee {
  id: string;
  displayName: string;
  email: string;
  login: string | null;
  status: string;
  roles: { id: string; name: string }[];
  revision: string;
}
interface PreparedRole {
  id: string;
  name: string;
  permissionKeys: string[];
}

function isPreparedAccountant(role: PreparedRole): boolean {
  return (
    role.permissionKeys.length === ACCOUNTANT_PERMISSION_KEYS.length &&
    ACCOUNTANT_PERMISSION_KEYS.every((key) => role.permissionKeys.includes(key))
  );
}

const statusLabel: Record<string, string> = { pending: 'Без пароля', active: 'Активен', suspended: 'Приостановлен' };
const reason: FieldSpec[] = [{ name: 'reason', label: 'Основание', type: 'textarea', required: true }];

function AccountantRoleNotice({
  loading,
  error,
  role,
}: {
  loading: boolean;
  error: unknown;
  role: PreparedRole | undefined;
}) {
  if (error)
    return (
      <Notice kind="warning">
        Не удалось загрузить роли для бухгалтера: {errorText(error)}. Повторите загрузку и настройте роль перед
        добавлением.
      </Notice>
    );
  if (!loading && !role)
    return (
      <Notice kind="warning">
        Подготовленная роль бухгалтера недоступна. Настройте её в разделе «Роли» перед добавлением сотрудника.
      </Notice>
    );
  return null;
}

/**
 * Employees of the signed-in user's company. A company admin (with
 * company.users.manage) adds employees with a login and temporary password and
 * assigns roles prepared by the JustixAuto platform admin; one user belongs
 * to one company.
 */
export function CompanyEmployees() {
  const s = useSession();
  const id = s.company?.id ?? '';
  const manage = !!id && s.can(PERM_COMPANY_USERS);
  const base = `/identity/companies/${id}`;
  const users = useData(['company-users', id], () => list<Employee>(`${base}/users`), manage);
  const roles = useData(['company-roles', id], () => list<PreparedRole>(`${base}/roles`), manage);
  const refresh = [['company-users', id]];
  const roleOptions = (roles.data ?? []).map((r): [string, string] => [r.id, r.name]);
  const accountantRole = (roles.data ?? []).find(isPreparedAccountant);

  if (!manage) {
    return (
      <Notice>
        Сотрудников компании добавляет её администратор. Обратитесь к нему, если вам нужен доступ к другим разделам.
      </Notice>
    );
  }
  return (
    <Panel
      title="Сотрудники"
      actions={
        <div className="kit-row">
          <ActionButton
            label="+ Добавить сотрудника"
            title="Новый сотрудник"
            submitLabel="Добавить"
            variant="primary"
            refresh={refresh}
            fields={[
              { name: 'displayName', label: 'Имя', type: 'text', required: true },
              { name: 'email', label: 'E-mail', type: 'email' },
              { name: 'login', label: 'Логин', type: 'text', required: true },
              {
                name: 'password',
                label: 'Временный пароль',
                type: 'password',
                required: true,
                hint: 'Не менее 8 символов',
              },
              { name: 'roleIds', label: 'Роли', type: 'multiselect', options: roleOptions },
            ]}
            intro={
              <p>
                Сотрудник сменит временный пароль при первом входе; письма не отправляются.
                {roles.data && roleOptions.length === 0 && ' Ролей пока нет — создайте их в разделе «Роли».'}
              </p>
            }
            onSubmit={(v) => post(`${base}/users`, v)}
          />
          {s.company?.kind === 'seller' && (
            <ActionButton
              label="+ Добавить бухгалтера"
              title="Новый бухгалтер"
              submitLabel="Добавить бухгалтера"
              variant="secondary"
              disabled={roles.isLoading || !!roles.error || !accountantRole}
              refresh={refresh}
              fields={[
                { name: 'displayName', label: 'Имя', type: 'text', required: true },
                { name: 'email', label: 'E-mail', type: 'email' },
                { name: 'login', label: 'Логин', type: 'text', required: true },
                {
                  name: 'password',
                  label: 'Временный пароль',
                  type: 'password',
                  required: true,
                  hint: 'Не менее 8 символов',
                },
              ]}
              intro={
                <p>
                  Будет назначена подготовленная роль бухгалтера с доступом к счетам и проверке оплат. Сотрудник сменит
                  временный пароль при первом входе.
                </p>
              }
              onSubmit={(v) => {
                if (!accountantRole || roles.error)
                  return Promise.reject(new Error('Подготовленная роль бухгалтера недоступна.'));
                return post(`${base}/users`, { ...v, roleIds: [accountantRole.id] });
              }}
            />
          )}
        </div>
      }
    >
      {s.company?.kind === 'seller' && (
        <AccountantRoleNotice loading={roles.isLoading} error={roles.error} role={accountantRole} />
      )}
      <Table
        rows={users.data}
        loading={users.isLoading}
        error={users.error}
        rowKey={(u) => u.id}
        empty="Сотрудников пока нет"
        columns={[
          {
            title: 'Сотрудник',
            render: (u) => <Cell main={u.displayName} sub={[u.login, u.email].filter(Boolean).join(' · ')} />,
          },
          { title: 'Роли', render: (u) => u.roles.map((r) => r.name).join(', ') || '—' },
          {
            title: 'Статус',
            render: (u) => (
              <Badge tone={u.status === 'active' ? 'success' : u.status === 'suspended' ? 'danger' : 'warning'}>
                {statusLabel[u.status] ?? u.status}
              </Badge>
            ),
          },
          {
            title: '',
            render: (u) => (
              <div className="row-actions">
                <ActionButton
                  small
                  label="Изменить"
                  refresh={refresh}
                  fields={[
                    { name: 'displayName', label: 'Имя', type: 'text', required: true, initial: u.displayName },
                    { name: 'email', label: 'E-mail', type: 'email', initial: u.email },
                    {
                      name: 'roleIds',
                      label: 'Роли',
                      type: 'multiselect',
                      options: roleOptions,
                      initial: u.roles.map((r) => r.id),
                    },
                  ]}
                  onSubmit={(v) => patch(`${base}/users/${u.id}`, v, { ifMatch: u.revision })}
                />
                {u.id !== s.view.user.id && (
                  <>
                    <ActionButton
                      small
                      label="Сбросить пароль"
                      refresh={refresh}
                      intro={<p>Сотрудник сменит временный пароль при первом входе.</p>}
                      fields={[
                        {
                          name: 'password',
                          label: 'Временный пароль',
                          type: 'password',
                          required: true,
                          hint: 'Не менее 8 символов',
                        },
                      ]}
                      onSubmit={(v) => post(`${base}/users/${u.id}/password`, v, { ifMatch: u.revision })}
                    />
                    {u.status === 'suspended' ? (
                      <ActionButton
                        small
                        label="Восстановить"
                        refresh={refresh}
                        fields={reason}
                        onSubmit={(v) => post(`${base}/users/${u.id}/restore`, v, { ifMatch: u.revision })}
                      />
                    ) : (
                      <ActionButton
                        small
                        label="Приостановить"
                        variant="danger"
                        refresh={refresh}
                        fields={reason}
                        onSubmit={(v) => post(`${base}/users/${u.id}/suspend`, v, { ifMatch: u.revision })}
                      />
                    )}
                  </>
                )}
              </div>
            ),
          },
        ]}
      />
    </Panel>
  );
}
