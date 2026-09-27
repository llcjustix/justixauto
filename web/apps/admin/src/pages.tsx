import { useState } from 'react';
import { useNavigate } from 'react-router-dom';
import {
  ActionButton,
  Badge,
  Button,
  Cell,
  Details,
  FilterSelect,
  Modal,
  Page,
  Panel,
  Stat,
  Stats,
  Table,
  Toolbar,
  dateTime,
  get,
  list,
  canonicalCountry,
  countries,
  matches,
  patch,
  permissionNames,
  permissionOptions,
  post,
  regionsFor,
  useData,
  useSearchQuery,
} from '@justixauto/kit';
import type { FieldSpec, Permission } from '@justixauto/kit';
import { kindLabel } from './labels';

// ---- types (see /api/v1/identity) ----
interface Label {
  key?: string;
  label: string;
}
interface Company {
  id: string;
  kind: string;
  name: string;
  legalName: string;
  country: Label;
  region: Label | null;
  registration: string;
  email: string;
  address: string;
  phone: string;
  access: string;
  accessReason: string;
  revision: string;
}
interface User {
  id: string;
  displayName: string;
  email: string;
  login: string | null;
  status: string;
  roles: { id: string; name: string }[];
  revision: string;
}
interface Role {
  id: string;
  name: string;
  system: boolean;
  scope: string;
  permissionKeys: string[];
  revision: string;
}
interface AuditEvent {
  id: string;
  occurredAt: string;
  actorId: string | null;
  action: string;
  resourceType: string;
  resourceId: string;
  companyId: string | null;
  reason: string;
  details: unknown;
}

const accessTone = (a: string) => (a === 'active' ? 'success' : a === 'suspended' ? 'danger' : 'warning');
const accessLabel: Record<string, string> = { draft: 'Черновик', active: 'Активна', suspended: 'Приостановлена' };

const companyFields = (c?: Company): FieldSpec[] => [
  {
    name: 'name',
    label: 'Название компании',
    type: 'text',
    required: true,
    initial: c?.name ?? '',
    group: 'Информация о компании',
  },
  {
    name: 'country',
    label: 'Страна',
    type: 'combobox',
    initial: c?.country.label ?? '',
    options: countries(),
    canonicalize: canonicalCountry,
    placeholder: 'Выберите или найдите страну',
    ariaLabel: 'Показать страны',
    group: 'Адрес',
  },
  {
    name: 'region',
    label: 'Регион',
    type: 'combobox',
    initial: c?.region?.label ?? '',
    dependsOn: 'country',
    optionsFor: regionsFor,
    placeholder: 'Выберите или найдите регион',
    disabledPlaceholder: 'Сначала выберите страну',
    ariaLabel: 'Показать регионы',
    group: 'Адрес',
  },
  { name: 'address', label: 'Адрес', type: 'text', initial: c?.address ?? '', group: 'Адрес' },
  {
    name: 'email',
    label: 'Электронная почта',
    type: 'email',
    initial: c?.email ?? '',
    group: 'Контакты',
  },
  { name: 'phone', label: 'Телефон', type: 'text', initial: c?.phone ?? '', group: 'Контакты' },
];
// registration is never entered in a form; on create it stays empty and on
// edit the company's current value is preserved unchanged (it only ever
// changes through the future government-source integration).
const companyInput = (v: Record<string, unknown>, legalName = '', registration = '') => ({
  name: v.name,
  legalName,
  country: { label: canonicalCountry(String(v.country)) ?? v.country },
  region: v.region ? { label: v.region } : null,
  registration,
  email: v.email,
  phone: v.phone,
  address: v.address,
});
const adminFields: FieldSpec[] = [
  {
    name: 'displayName',
    label: 'Имя администратора',
    type: 'text',
    hint: 'Если не заполнено, используется логин.',
    group: 'Данные для входа',
  },
  { name: 'login', label: 'Логин', type: 'text', required: true, group: 'Данные для входа' },
  {
    name: 'password',
    label: 'Пароль',
    type: 'password',
    required: true,
    hint: 'Не менее 12 символов.',
    group: 'Данные для входа',
  },
  {
    name: 'passwordConfirmation',
    label: 'Повторите пароль',
    type: 'password',
    required: true,
    group: 'Данные для входа',
  },
];
const firstAdmin = (v: Record<string, unknown>) => ({
  displayName: v.displayName,
  login: v.login,
  email: v.email,
  password: v.password,
  passwordConfirmation: v.passwordConfirmation,
});

type Kind = 'seller' | 'bank' | 'mfo' | 'insurance';
const kindTitle: Record<Kind, [string, string]> = {
  seller: ['Компании', 'Продавцы автомобилей · JustixAuto Реализация'],
  mfo: ['МФО', 'Микрофинансовые организации · кабинет Финансирование'],
  bank: ['Банки', 'Банки · кабинет Финансирование'],
  insurance: ['Страховые компании', 'Страховые компании · кабинет Страховая'],
};
const cabinetLabel: Record<string, string> = {
  draft: 'Черновик',
  active: 'Доступ активен',
  suspended: 'Доступ приостановлен',
};

export function CompaniesPage({ kind }: { kind: Kind }) {
  const q = useData(['admin-companies', kind], () => list<Company>(`/identity/admin/companies?kind=${kind}&limit=100`));
  const [open, setOpen] = useState<Company | null>(null);
  const [query, setQuery] = useSearchQuery();
  const [access, setAccess] = useState('');
  const rows = (q.data ?? []).filter(
    (c) => (!access || c.access === access) && matches(query, c.name, c.registration, c.email, c.legalName),
  );
  const [title, subtitle] = kindTitle[kind];
  return (
    <Page
      title={title}
      subtitle={subtitle}
      actions={
        kind === 'seller' ? (
          <ActionButton
            label="+ Добавить компанию"
            title="Новая компания и администратор"
            submitLabel="Создать компанию"
            variant="primary"
            size="wide"
            fields={[...companyFields(), ...adminFields]}
            refresh={[['admin-companies']]}
            intro={
              <p>
                Контактная электронная почта также используется для первого администратора. Компания остаётся черновиком
                до активации.
              </p>
            }
            onSubmit={(v) =>
              post('/identity/admin/seller-companies', { company: companyInput(v), firstAdmin: firstAdmin(v) })
            }
          />
        ) : (
          <ActionButton
            label={`+ Подключить: ${kindLabel[kind]}`}
            title={`Новая компания и администратор: ${kindLabel[kind]}`}
            submitLabel="Создать компанию"
            variant="primary"
            size="wide"
            fields={[...companyFields(), ...adminFields]}
            refresh={[['admin-companies']]}
            intro={
              <p>
                Контактная электронная почта также используется для первого администратора. Подключение создаёт
                отдельный кабинет.
              </p>
            }
            onSubmit={(v) =>
              post('/identity/admin/provider-companies', { kind, company: companyInput(v), firstAdmin: firstAdmin(v) })
            }
          />
        )
      }
    >
      <Panel>
        <Toolbar
          query={query}
          onQuery={setQuery}
          placeholder="Название, БИН / регистрационный номер или email"
          onReset={() => setAccess('')}
        >
          <FilterSelect value={access} onChange={setAccess} all="Все статусы" options={Object.entries(cabinetLabel)} />
        </Toolbar>
        <Table
          rows={rows}
          loading={q.isLoading}
          error={q.error}
          rowKey={(c) => c.id}
          onRowClick={setOpen}
          empty="Компаний пока нет"
          columns={[
            {
              title: 'Компания',
              render: (c) => (
                <Cell
                  main={c.name}
                  sub={[c.registration, kind === 'seller' ? 'Реализация' : kindLabel[kind]].filter(Boolean).join(' · ')}
                />
              ),
            },
            { title: 'Страна / регион', render: (c) => <Cell main={c.country.label} sub={c.region?.label} /> },
            {
              title: 'Кабинет',
              render: (c) => <Badge tone={accessTone(c.access)}>{cabinetLabel[c.access] ?? c.access}</Badge>,
            },
            { title: 'Email', render: (c) => c.email },
            {
              title: '',
              render: (c) => (
                <Button size="sm" onClick={() => setOpen(c)}>
                  Открыть
                </Button>
              ),
            },
          ]}
        />
      </Panel>
      {open && <CompanyDialog id={open.id} onClose={() => setOpen(null)} />}
    </Page>
  );
}

export function OverviewPage() {
  const navigate = useNavigate();
  const all = useData(['admin-companies', 'all'], () => list<Company>('/identity/admin/companies?limit=200'));
  const audit = useData(['admin-audit'], () => list<AuditEvent>('/identity/admin/audit?limit=100'));
  const users = useData(['admin-users'], () => list<User>('/identity/admin/users?limit=200'));
  const card = (kind: Kind, path: string) => {
    const cs = (all.data ?? []).filter((c) => c.kind === kind);
    return (
      <Stat
        key={kind}
        label={kindTitle[kind][0]}
        value={cs.length}
        note={`${cs.filter((c) => c.access === 'active').length} с активным доступом →`}
        onClick={() => navigate(path)}
      />
    );
  };
  return (
    <Page title="Обзор" subtitle="Управление доступом участников JustixAuto">
      <Stats>
        {card('mfo', '/integrations/mfo')}
        {card('bank', '/integrations/bank')}
        {card('insurance', '/integrations/insurance')}
        {card('seller', '/companies')}
      </Stats>
      <Panel title="Последние действия" actions={<Button onClick={() => navigate('/audit')}>Весь журнал</Button>}>
        <AuditTable
          rows={audit.data?.slice(0, 6)}
          loading={audit.isLoading}
          error={audit.error}
          users={users.data ?? []}
          companies={all.data ?? []}
        />
      </Panel>
      <div className="admin-note">
        Создание компании не подключает API и не публикует финансовые программы. Решения по заявкам остаются в кабинетах
        страховых и финансовых организаций.
      </div>
    </Page>
  );
}

function CompanyDialog({ id, onClose }: { id: string; onClose: () => void }) {
  const q = useData(['admin-company', id], () => get<Company>(`/identity/companies/${id}`));
  const c = q.data?.data;
  const refresh = [['admin-company', id], ['admin-companies']];
  const access = (action: string, label: string) => (
    <ActionButton
      label={label}
      refresh={refresh}
      fields={[{ name: 'reason', label: 'Основание', type: 'textarea', required: true }]}
      onSubmit={(v) =>
        post(`/identity/admin/companies/${id}/${action}`, { reason: v.reason }, { ifMatch: c!.revision })
      }
    />
  );
  return (
    <Modal
      title={c?.name ?? 'Компания'}
      onClose={onClose}
      size="wide"
      footer={
        c && (
          <>
            {c.access === 'draft' && access('activate', 'Активировать')}
            {c.access === 'active' && access('suspend', 'Приостановить')}
            {c.access === 'suspended' && access('restore', 'Восстановить')}
            <ActionButton
              label="Удалить"
              variant="danger"
              intro={
                <p>
                  Компания исчезнет из всех списков и справочников, её сотрудники потеряют к ней доступ. Данные и
                  история сохраняются.
                </p>
              }
              fields={[{ name: 'reason', label: 'Основание', type: 'textarea', required: true }]}
              refresh={[['admin-companies'], ['admin-memberships']]}
              onSubmit={async (v) => {
                const r = await post(
                  `/identity/admin/companies/${id}/delete`,
                  { reason: v.reason },
                  { ifMatch: c.revision },
                );
                onClose();
                return r;
              }}
            />
            <ActionButton
              label="Изменить реквизиты"
              fields={companyFields(c)}
              refresh={refresh}
              onSubmit={(v) =>
                patch(`/identity/companies/${id}`, companyInput(v, c.legalName, c.registration), {
                  ifMatch: c.revision,
                })
              }
            />
          </>
        )
      }
    >
      {c && (
        <Details
          items={[
            ['Тип', kindLabel[c.kind]],
            ['Доступ', <Badge tone={accessTone(c.access)}>{accessLabel[c.access]}</Badge>],
            ['Основание', c.accessReason || '—'],
            ['Юр. название', c.legalName || '—'],
            ['Страна / регион', `${c.country.label}${c.region ? ', ' + c.region.label : ''}`],
            ...(c.registration ? [['Рег. номер', c.registration] as [string, string]] : []),
            ['E-mail', c.email],
            ['Телефон', c.phone || '—'],
            ['Адрес', c.address || '—'],
          ]}
        />
      )}
      <p className="cell-sub">
        Активация открывает доступ к платформе; она не подтверждает лицензию, API банка или право продавать продукт.
      </p>
    </Modal>
  );
}

const statusLabel: Record<string, string> = { pending: 'Без пароля', active: 'Активен', suspended: 'Приостановлен' };

export function UsersPage() {
  const q = useData(['admin-users'], () => list<User>('/identity/admin/users?limit=200'));
  const roles = useData(['admin-roles'], () => list<Role>('/identity/admin/roles'));
  const staffRoles = (roles.data ?? []).filter((r) => r.scope === 'platform');
  // New staff start as platform administrators (user decision 2026-09-26).
  const platformAdmin = staffRoles.find((r) => r.system);
  const [open, setOpen] = useState<string | null>(null);
  const [query, setQuery] = useSearchQuery();
  const [status, setStatus] = useState('');
  const rows = (q.data ?? []).filter(
    (u) => (!status || u.status === status) && matches(query, u.displayName, u.email, u.login),
  );
  return (
    <Page
      title="Сотрудники платформы"
      subtitle="Администраторы и операторы JustixAuto. Сотрудников компаний добавляет администратор компании в своём кабинете."
      actions={
        <ActionButton
          label="+ Добавить сотрудника"
          title="Новый сотрудник платформы"
          submitLabel="Добавить"
          variant="primary"
          refresh={[['admin-users']]}
          fields={[
            { name: 'displayName', label: 'Имя', type: 'text', required: true },
            { name: 'email', label: 'E-mail', type: 'email' },
            { name: 'login', label: 'Логин', type: 'text', required: true },
            {
              name: 'password',
              label: 'Временный пароль',
              type: 'password',
              required: true,
              hint: 'Не менее 12 символов',
            },
          ]}
          intro={
            <p>
              Сотрудник получает роль «Platform administrator»; её можно изменить в карточке. При первом входе временный
              пароль нужно сменить. Письма не отправляются.
            </p>
          }
          onSubmit={(v) => post('/identity/admin/users', { ...v, roleIds: platformAdmin ? [platformAdmin.id] : [] })}
        />
      }
    >
      <Panel>
        <Toolbar query={query} onQuery={setQuery} placeholder="Имя, логин или email" onReset={() => setStatus('')}>
          <FilterSelect value={status} onChange={setStatus} all="Все статусы" options={Object.entries(statusLabel)} />
        </Toolbar>
        <Table
          rows={rows}
          loading={q.isLoading}
          error={q.error}
          rowKey={(u) => u.id}
          onRowClick={(u) => setOpen(u.id)}
          empty="Сотрудников нет"
          columns={[
            {
              title: 'Сотрудник',
              render: (u) => <Cell main={u.displayName} sub={[u.login, u.email].filter(Boolean).join(' · ')} />,
            },
            { title: 'Роли', render: (u) => u.roles.map((r) => r.name).join(', ') || '—' },
            {
              title: 'Доступ',
              render: (u) => (
                <Badge tone={u.status === 'active' ? 'success' : u.status === 'suspended' ? 'danger' : 'warning'}>
                  {statusLabel[u.status] ?? u.status}
                </Badge>
              ),
            },
            {
              title: '',
              render: (u) => (
                <Button size="sm" onClick={() => setOpen(u.id)}>
                  Открыть
                </Button>
              ),
            },
          ]}
        />
      </Panel>
      <div className="admin-note">Пароли в журнал не попадают.</div>
      {open && <UserDialog id={open} roles={staffRoles} onClose={() => setOpen(null)} />}
    </Page>
  );
}

function UserDialog({ id, roles, onClose }: { id: string; roles: Role[]; onClose: () => void }) {
  const q = useData(['admin-user', id], () => get<User>(`/identity/admin/users/${id}`));
  const u = q.data?.data;
  const refresh = [['admin-user', id], ['admin-users']];
  const reason: FieldSpec[] = [{ name: 'reason', label: 'Основание', type: 'textarea', required: true }];
  return (
    <Modal
      title={u?.displayName ?? 'Сотрудник'}
      onClose={onClose}
      size="wide"
      footer={
        u && (
          <>
            <ActionButton
              label="Изменить"
              refresh={refresh}
              fields={[
                { name: 'displayName', label: 'Имя', type: 'text', required: true, initial: u.displayName },
                {
                  name: 'roleIds',
                  label: 'Роли',
                  type: 'multiselect',
                  options: roles.map((r) => [r.id, r.name]),
                  initial: u.roles.map((r) => r.id),
                },
              ]}
              onSubmit={(v) => patch(`/identity/admin/users/${id}`, v, { ifMatch: u.revision })}
            />
            <ActionButton
              label={u.login ? 'Сбросить пароль' : 'Выдать доступ'}
              refresh={refresh}
              intro={<p>Сотрудник сменит временный пароль при первом входе.</p>}
              fields={[
                ...(u.login ? [] : [{ name: 'login', label: 'Логин', type: 'text', required: true } as FieldSpec]),
                { name: 'password', label: 'Временный пароль', type: 'password', required: true },
                { name: 'passwordConfirmation', label: 'Повторите пароль', type: 'password', required: true },
              ]}
              onSubmit={(v) => post(`/identity/admin/users/${id}/password`, v, { ifMatch: u.revision })}
            />
            {u.status === 'suspended' ? (
              <ActionButton
                label="Восстановить"
                fields={reason}
                refresh={refresh}
                onSubmit={(v) => post(`/identity/admin/users/${id}/restore`, v, { ifMatch: u.revision })}
              />
            ) : (
              <ActionButton
                label="Приостановить"
                variant="danger"
                fields={reason}
                refresh={refresh}
                onSubmit={(v) => post(`/identity/admin/users/${id}/suspend`, v, { ifMatch: u.revision })}
              />
            )}
            <ActionButton
              label="Завершить сеансы"
              fields={reason}
              onSubmit={(v) => post(`/identity/admin/users/${id}/revoke-sessions`, v)}
            />
          </>
        )
      }
    >
      {u && (
        <Details
          items={[
            ['E-mail', u.email || '—'],
            ['Логин', u.login ?? '—'],
            ['Статус', statusLabel[u.status]],
            ['Роли', u.roles.map((r) => r.name).join(', ') || '—'],
          ]}
        />
      )}
    </Modal>
  );
}

export function RolesPage() {
  const q = useData(['admin-roles'], () => list<Role>('/identity/admin/roles'));
  const perms = useData(['admin-permissions'], () => list<Permission>('/identity/admin/permissions'));
  const options = permissionOptions(perms.data ?? []);
  // Built-in company administrator: every company permission (managed in companies).
  const permList = (r: Role) =>
    r.system && r.scope === 'company' ? 'Все разрешения компании' : permissionNames(perms.data ?? [], r.permissionKeys);
  const fields = (r?: Role): FieldSpec[] => [
    { name: 'name', label: 'Название', type: 'text', required: true, full: true, initial: r?.name ?? '' },
    { name: 'permissionKeys', label: 'Разрешения', type: 'multiselect', options, initial: r?.permissionKeys ?? [] },
  ];
  return (
    <Page
      title="Роли"
      subtitle="Роли сотрудников платформы: название и набор разрешений платформы. Роли компаний создают сами компании в своём кабинете."
      actions={
        <ActionButton
          label="+ Создать роль"
          title="Новая роль"
          submitLabel="Создать"
          variant="primary"
          size="wide"
          fields={fields()}
          refresh={[['admin-roles']]}
          onSubmit={(v) => post('/identity/admin/roles', v)}
        />
      }
    >
      <Panel>
        <Table
          rows={q.data}
          loading={q.isLoading}
          error={q.error}
          rowKey={(r) => r.id}
          columns={[
            { title: 'Роль', render: (r) => <Cell main={r.name} sub={r.system ? 'Встроенная' : undefined} /> },
            {
              title: 'Разрешения',
              render: (r) => <span title={permList(r)}>{r.permissionKeys.length}</span>,
            },
            {
              title: '',
              render: (r) =>
                r.system ? (
                  <ActionButton
                    small
                    label="Просмотреть"
                    title={r.name}
                    fields={[]}
                    submitLabel="Закрыть"
                    intro={<p>{permList(r) || 'Права платформы'}</p>}
                    onSubmit={async () => undefined}
                  />
                ) : (
                  <div className="row-actions">
                    <ActionButton
                      small
                      label="Изменить"
                      title="Изменить роль"
                      submitLabel="Сохранить"
                      size="wide"
                      fields={fields(r)}
                      refresh={[['admin-roles']]}
                      onSubmit={(v) => patch(`/identity/admin/roles/${r.id}`, v, { ifMatch: r.revision })}
                    />
                    <ActionButton
                      small
                      label="Удалить"
                      title="Удалить роль"
                      submitLabel="Удалить"
                      variant="danger"
                      refresh={[['admin-roles'], ['admin-users']]}
                      intro={<p>«{r.name}» исчезнет из списков, а сотрудники с этой ролью потеряют её права.</p>}
                      onSubmit={() => post(`/identity/admin/roles/${r.id}/delete`, {}, { ifMatch: r.revision })}
                    />
                  </div>
                ),
            },
          ]}
        />
      </Panel>
      <div className="admin-note">
        Встроенные роли не изменяются. «Company administrator» получает все разрешения компании. Разрешения и их
        названия хранятся в PostgreSQL (identity.permissions).
      </div>
    </Page>
  );
}

interface CarSpec {
  make: string;
  model: string;
  variant: string;
  year: number;
  bodyType: string;
  exteriorColor: string;
  interiorColor: string;
  powertrain: string;
  drivetrain: string;
  version: number;
}
interface CarModel {
  id: string;
  specification: CarSpec;
  revision: string;
}

const carMakes = [
  'BYD',
  'Changan',
  'Chery',
  'Chevrolet',
  'Exeed',
  'Geely',
  'Haval',
  'Honda',
  'Hongqi',
  'Hyundai',
  'Jetour',
  'Kia',
  'Lada',
  'Leapmotor',
  'Lexus',
  'Li Auto',
  'Mazda',
  'Mercedes-Benz',
  'BMW',
  'Nissan',
  'Ravon',
  'Tesla',
  'Toyota',
  'Volkswagen',
  'Voyah',
  'Zeekr',
].sort((a, b) => a.localeCompare(b));
const carBodies = [
  'Седан',
  'Хэтчбек',
  'Лифтбек',
  'Универсал',
  'Кроссовер',
  'Внедорожник',
  'Минивэн',
  'Купе',
  'Кабриолет',
  'Пикап',
  'Фургон',
];
const carPowertrains = ['Бензин', 'Дизель', 'Гибрид', 'Подключаемый гибрид', 'Электро', 'Газ / бензин'];
const carDrivetrains = ['Передний', 'Задний', 'Полный'];
const carColors = [
  'Белый',
  'Чёрный',
  'Серый',
  'Серебристый',
  'Синий',
  'Голубой',
  'Красный',
  'Бордовый',
  'Бежевый',
  'Коричневый',
  'Зелёный',
  'Жёлтый',
  'Оранжевый',
];
const thisYear = new Date().getFullYear();
const carYears = Array.from({ length: thisYear + 2 - 1990 }, (_, i) => String(thisYear + 1 - i));

/** Select options that always contain the current value (older free-text data). */
const choices = (list: string[], current?: string): [string, string][] =>
  (current && !list.includes(current) ? [current, ...list] : list).map((x) => [x, x]);

const carSpecFields = (models: CarModel[], s?: CarSpec): FieldSpec[] => [
  {
    name: 'make',
    label: 'Марка',
    type: 'combobox',
    required: true,
    options: [...new Set([...carMakes, ...models.map((m) => m.specification.make)])].sort((a, b) => a.localeCompare(b)),
    placeholder: 'Выберите или введите марку',
    initial: s?.make ?? '',
  },
  {
    name: 'model',
    label: 'Модель',
    type: 'combobox',
    required: true,
    dependsOn: 'make',
    optionsFor: (make) => [
      ...new Set(
        models
          .filter((m) => m.specification.make.toLowerCase() === make.trim().toLowerCase())
          .map((m) => m.specification.model),
      ),
    ],
    placeholder: 'Выберите или введите модель',
    disabledPlaceholder: 'Сначала выберите марку',
    initial: s?.model ?? '',
  },
  { name: 'variant', label: 'Комплектация', type: 'text', required: true, initial: s?.variant ?? '' },
  {
    name: 'year',
    label: 'Год',
    type: 'select',
    required: true,
    options: choices(carYears, s && String(s.year)),
    initial: s ? String(s.year) : String(thisYear),
  },
  {
    name: 'bodyType',
    label: 'Кузов',
    type: 'select',
    required: true,
    options: choices(carBodies, s?.bodyType),
    initial: s?.bodyType ?? '',
  },
  {
    name: 'powertrain',
    label: 'Двигатель',
    type: 'select',
    required: true,
    options: choices(carPowertrains, s?.powertrain),
    initial: s?.powertrain ?? '',
  },
  {
    name: 'drivetrain',
    label: 'Привод',
    type: 'select',
    required: true,
    options: choices(carDrivetrains, s?.drivetrain),
    initial: s?.drivetrain ?? '',
  },
  {
    name: 'exteriorColor',
    label: 'Цвет кузова',
    type: 'combobox',
    required: true,
    options: carColors,
    placeholder: 'Выберите или введите цвет',
    initial: s?.exteriorColor ?? '',
  },
  {
    name: 'interiorColor',
    label: 'Цвет салона',
    type: 'combobox',
    required: true,
    options: carColors,
    placeholder: 'Выберите или введите цвет',
    initial: s?.interiorColor ?? '',
  },
];
const carSpecBody = (v: Record<string, unknown>) => ({ specification: { ...v, year: Number(v.year) } });

/** The shared car catalog: only the platform admin maintains it (user decision 2026-09-27). */
export function CatalogPage() {
  const [query, setQuery] = useSearchQuery();
  const q = useData(['admin-car-models', query], () =>
    list<CarModel>(`/inventory/vehicle-models?limit=100&q=${encodeURIComponent(query)}`),
  );
  const all = useData(['admin-car-models', ''], () => list<CarModel>('/inventory/vehicle-models?limit=100'));
  const known = all.data ?? [];
  const refresh = [['admin-car-models']];
  return (
    <Page
      title="Каталог автомобилей"
      subtitle="Общий каталог марок, моделей и комплектаций. Компании выбирают из него модели при приёмке автомобилей."
      actions={
        <ActionButton
          label="+ Добавить модель"
          title="Новая модель"
          submitLabel="Добавить"
          variant="primary"
          size="wide"
          fields={carSpecFields(known)}
          refresh={refresh}
          onSubmit={(v) => post('/inventory/vehicle-models', carSpecBody(v))}
        />
      }
    >
      <Panel>
        <Toolbar query={query} onQuery={setQuery} placeholder="Марка, модель или комплектация" />
        <Table
          rows={q.data}
          loading={q.isLoading}
          error={q.error}
          rowKey={(m) => m.id}
          empty="В каталоге пока нет моделей"
          columns={[
            {
              title: 'Модель',
              render: (m) => (
                <Cell main={`${m.specification.make} ${m.specification.model}`} sub={m.specification.variant} />
              ),
            },
            { title: 'Год', render: (m) => m.specification.year },
            {
              title: 'Характеристики',
              render: (m) =>
                [m.specification.bodyType, m.specification.powertrain, m.specification.drivetrain]
                  .filter(Boolean)
                  .join(' · '),
            },
            { title: 'Версия', render: (m) => m.specification.version },
            {
              title: '',
              render: (m) => (
                <ActionButton
                  small
                  label="Новая версия"
                  title="Новая версия характеристик"
                  submitLabel="Сохранить"
                  size="wide"
                  intro={<p>Уже принятые автомобили сохраняют прежнюю версию характеристик.</p>}
                  fields={carSpecFields(known, m.specification)}
                  refresh={refresh}
                  onSubmit={(v) =>
                    post(`/inventory/vehicle-models/${m.id}/specification-versions`, carSpecBody(v), {
                      ifMatch: m.revision,
                    })
                  }
                />
              ),
            },
          ]}
        />
      </Panel>
    </Page>
  );
}

const permissionScopeLabel: Record<string, string> = { platform: 'Платформа', company: 'Компания' };

export function PermissionsPage() {
  const q = useData(['admin-permission-catalog'], () => list<Permission>('/identity/admin/permission-catalog'));
  const [scope, setScope] = useState('');
  const [query, setQuery] = useSearchQuery();
  const rows = (q.data ?? []).filter((p) => (!scope || p.scope === scope) && matches(query, p.name, p.key));
  const refresh = [['admin-permission-catalog'], ['admin-permissions']];
  const assignable: FieldSpec = {
    name: 'assignable',
    label: 'Можно давать в роли',
    type: 'checkbox',
    initial: true,
  };
  return (
    <Page
      title="Разрешения"
      subtitle="Каталог разрешений в PostgreSQL. Разрешения платформы входят в роли сотрудников платформы, разрешения компании — в роли, которые создают компании."
      actions={
        <ActionButton
          label="+ Добавить разрешение"
          title="Новое разрешение"
          submitLabel="Добавить"
          variant="primary"
          refresh={refresh}
          fields={[
            {
              name: 'name',
              label: 'Название',
              type: 'text',
              required: true,
              full: true,
              hint: 'Например: Продажи: отчёты',
            },
            { name: 'key', label: 'Ключ', type: 'text', required: true, hint: 'Например: retail.reports.read' },
            {
              name: 'scope',
              label: 'Область',
              type: 'select',
              required: true,
              options: Object.entries(permissionScopeLabel),
            },
            assignable,
          ]}
          intro={
            <p>
              Ключ и область после создания не меняются. Разрешение начинает действовать, когда код проверяет этот ключ.
            </p>
          }
          onSubmit={(v) => post('/identity/admin/permission-catalog', v)}
        />
      }
    >
      <Panel>
        <Toolbar query={query} onQuery={setQuery} placeholder="Название или ключ" onReset={() => setScope('')}>
          <FilterSelect
            value={scope}
            onChange={setScope}
            all="Все области"
            options={Object.entries(permissionScopeLabel)}
          />
        </Toolbar>
        <Table
          rows={rows}
          loading={q.isLoading}
          error={q.error}
          rowKey={(p) => p.key}
          empty="Разрешений нет"
          columns={[
            { title: 'Разрешение', render: (p) => <Cell main={p.name} sub={p.key} /> },
            { title: 'Область', render: (p) => permissionScopeLabel[p.scope] ?? p.scope },
            {
              title: 'В ролях',
              render: (p) => (
                <Badge tone={p.assignable ? 'success' : undefined}>{p.assignable ? 'Можно' : 'Нельзя'}</Badge>
              ),
            },
            {
              title: '',
              render: (p) => (
                <div className="row-actions">
                  <ActionButton
                    small
                    label="Изменить"
                    title="Изменить разрешение"
                    submitLabel="Сохранить"
                    refresh={refresh}
                    intro={
                      <p>
                        {p.key} · {permissionScopeLabel[p.scope]}
                      </p>
                    }
                    fields={[
                      { name: 'name', label: 'Название', type: 'text', required: true, full: true, initial: p.name },
                      { ...assignable, initial: p.assignable } as FieldSpec,
                    ]}
                    onSubmit={(v) => patch(`/identity/admin/permission-catalog/${encodeURIComponent(p.key)}`, v)}
                  />
                  {p.assignable && (
                    <ActionButton
                      small
                      label="Удалить"
                      title="Удалить разрешение"
                      submitLabel="Удалить"
                      variant="danger"
                      refresh={refresh}
                      intro={
                        <p>
                          «{p.name}» исчезнет из каталога и перестанет действовать во всех ролях. История сохраняется.
                        </p>
                      }
                      onSubmit={() => post(`/identity/admin/permission-catalog/${encodeURIComponent(p.key)}/delete`)}
                    />
                  )}
                </div>
              ),
            },
          ]}
        />
      </Panel>
    </Page>
  );
}

const auditLabel: Record<string, string> = {
  'company.created': 'Компания создана',
  'company.activate': 'Доступ компании активирован',
  'company.suspend': 'Доступ компании приостановлен',
  'company.restore': 'Доступ компании восстановлен',
  'company.provider_provisioned': 'Подключена организация',
  'user.mfa_enrolled': 'Включена двухфакторная защита',
  'user.password_changed': 'Пароль изменён пользователем',
  'company.updated': 'Реквизиты изменены',
  'company.activated': 'Доступ компании активирован',
  'company.suspended': 'Доступ компании приостановлен',
  'company.restored': 'Доступ компании восстановлен',
  'company.deleted': 'Компания удалена',
  'branch.created': 'Филиал создан',
  'branch.updated': 'Филиал изменён',
  'user.created': 'Пользователь создан',
  'user.updated': 'Пользователь изменён',
  'user.suspended': 'Пользователь приостановлен',
  'user.restored': 'Пользователь восстановлен',
  'user.password_set': 'Пароль задан администратором',
  'user.sessions_revoked': 'Сеансы завершены',
  'membership.granted': 'Доступ к компании выдан',
  'membership.revoked': 'Доступ к компании отозван',
  'membership.branch_access_changed': 'Доступ к филиалам изменён',
  'role.created': 'Роль создана',
  'permission.created': 'Разрешение добавлено',
  'permission.updated': 'Разрешение изменено',
  'permission.deleted': 'Разрешение удалено',
  'role.deleted': 'Роль удалена',
  'role.updated': 'Роль изменена',
  'user.bootstrapped': 'Первый администратор платформы',
  'mfa.enrolled': 'Включена двухфакторная защита',
  'password.changed': 'Пароль изменён',
};

function AuditTable({
  rows,
  loading,
  error,
  users,
  companies,
}: {
  rows: AuditEvent[] | undefined;
  loading: boolean;
  error: unknown;
  users: User[];
  companies: Company[];
}) {
  return (
    <Table
      rows={rows}
      loading={loading}
      error={error}
      rowKey={(e) => e.id}
      empty="Действий пока нет"
      columns={[
        { title: 'Время', render: (e) => dateTime(e.occurredAt) },
        {
          title: 'Сотрудник',
          render: (e) =>
            users.find((u) => u.id === e.actorId)?.displayName ?? (e.actorId ? e.actorId.slice(0, 8) : 'Система'),
        },
        {
          title: 'Действие',
          render: (e) => (
            <Cell main={auditLabel[e.action] ?? e.action} sub={e.reason ? `Основание: ${e.reason}` : undefined} />
          ),
        },
        { title: 'Компания', render: (e) => companies.find((c) => c.id === e.companyId)?.name ?? '—' },
      ]}
    />
  );
}

export function AuditPage() {
  const q = useData(['admin-audit'], () => list<AuditEvent>('/identity/admin/audit?limit=100'));
  const users = useData(['admin-users'], () => list<User>('/identity/admin/users?limit=200'));
  const companies = useData(['admin-companies', 'all'], () => list<Company>('/identity/admin/companies?limit=200'));
  return (
    <Page title="Журнал действий" subtitle="История изменений без возможности редактирования">
      <Panel>
        <AuditTable
          rows={q.data}
          loading={q.isLoading}
          error={q.error}
          users={users.data ?? []}
          companies={companies.data ?? []}
        />
      </Panel>
    </Page>
  );
}
