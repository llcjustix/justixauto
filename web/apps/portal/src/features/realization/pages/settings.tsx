import { useState } from 'react';
import {
  ActionButton,
  Badge,
  Button,
  Cell,
  ChangePassword,
  CompanyEmployees,
  CompanyRoles,
  Details,
  Notice,
  Panel,
  Table,
  canonicalCountry,
  countries,
  regionsFor,
  get,
  patch,
  post,
  useData,
  errorText,
  useRefresh,
  useScopeLabel,
  useSession,
  legalFormLabel,
  legalFormOptions,
} from '@justixauto/kit';
import type { FieldSpec, LegalForm } from '@justixauto/kit';
import { useBranches, useWarehouses } from '../data';
import type { Branch } from '../data';
import { WarehouseDialog } from './inventory';

interface Company {
  id: string;
  name: string;
  legalName: string;
  country: { key?: string; label: string };
  region: { key?: string; label: string } | null;
  registration: string;
  email: string;
  address: string;
  phone: string;
  access: string;
  userType: 'dealer' | 'distributor' | null;
  verification: 'unverified' | 'submitted' | 'verified' | 'returned';
  verificationReason: string;
  legalForm: LegalForm;
  taxId: string;
  bankRequisites: string;
}

const branchFields = (b?: Branch): FieldSpec[] => [
  { name: 'name', label: 'Название', type: 'text', required: true, ...(b ? { initial: b.name } : {}) },
  { name: 'address', label: 'Адрес', type: 'text', required: true, ...(b ? { initial: b.address } : {}) },
];
type Section = 'companies' | 'members' | 'account';
const sections: [Section, string, string][] = [
  ['companies', 'Компании и филиалы', 'Юридические лица и точки продаж'],
  ['members', 'Пользователи и роли', 'Доступ сотрудников'],
  ['account', 'Личный аккаунт', 'Профиль и пароль'],
];
const initials = (n: string) =>
  n
    .split(/\s+/)
    .slice(0, 2)
    .map((w) => w[0]?.toUpperCase())
    .join('');

export function SettingsPage() {
  const [section, setSection] = useState<Section>('companies');
  return (
    <section className="page">
      <div className="page-header">
        <div>
          <h1 className="page-title">Настройки</h1>
          <div className="page-subtitle">Компания, филиалы, склады, доступ сотрудников и личная безопасность</div>
        </div>
      </div>
      <div className="settings-layout">
        <nav className="settings-nav">
          {sections.map(([k, title, sub]) => (
            <button
              key={k}
              className={`settings-nav-item${section === k ? ' active' : ''}`}
              onClick={() => setSection(k)}
            >
              <strong>{title}</strong>
              <span>{sub}</span>
            </button>
          ))}
        </nav>
        <div>{section === 'companies' ? <Companies /> : section === 'members' ? <Members /> : <Account />}</div>
      </div>
    </section>
  );
}

function CompanyEditAction({
  c,
  id,
  refresh,
  revision,
}: {
  c: Company;
  id: string;
  refresh: unknown[][];
  revision: string | undefined;
}) {
  return (
    <ActionButton
      small
      label="Изменить данные"
      refresh={refresh}
      fields={[
        {
          name: 'legalForm',
          label: 'Организационная форма',
          type: 'select',
          initial: c.legalForm,
          options: legalFormOptions,
          searchable: true,
        },
        { name: 'name', label: 'Название', type: 'text', required: true, initial: c.name },
        {
          name: 'legalName',
          label: 'Юридическое название',
          type: 'text',
          initial: c.legalName,
        },
        { name: 'taxId', label: 'ИНН / БИН', type: 'text', initial: c.taxId },
        { name: 'bankRequisites', label: 'Банковские реквизиты', type: 'textarea', initial: c.bankRequisites, full: true },
        { name: 'email', label: 'Email', type: 'email', initial: c.email },
        { name: 'phone', label: 'Телефон', type: 'text', initial: c.phone },
        { name: 'address', label: 'Юридический адрес', type: 'text', initial: c.address },
      ]}
      onSubmit={(v) =>
        patch(
          `/identity/companies/${id}`,
          { ...v, country: c.country, region: c.region, registration: c.registration },
          { ifMatch: revision! },
        )
      }
    />
  );
}

const accessLabel: Record<string, string> = { active: 'Активна', suspended: 'Приостановлена' };
const accessTone = (access: string | undefined) => (access === 'active' ? 'success' : 'warning');
const sellerTypeLabel: Record<string, string> = { dealer: 'Дилер', distributor: 'Дистрибьютор' };
const verificationLabel: Record<string, string> = {
  unverified: 'Не отправлена на проверку',
  submitted: 'На проверке платформы',
  verified: 'Проверена',
  returned: 'Возвращена на доработку',
};

function CompanyFacts({ c, scope }: { c: Company | undefined; scope: string }) {
  return (
    <div className="facts-grid">
      <div className="fact">
        <span>Организационная форма</span>
        <strong>{legalFormLabel(c?.legalForm)}</strong>
      </div>
      <div className="fact">
        <span>Страна и регион</span>
        <strong>{c ? [c.country.label, c.region?.label].filter(Boolean).join(', ') : '—'}</strong>
      </div>
      <div className="fact">
        <span>Юридический адрес</span>
        <strong>{c?.address || '—'}</strong>
      </div>
      <div className="fact">
        <span>Контакты</span>
        <strong>{[c?.email, c?.phone].filter(Boolean).join(' · ') || '—'}</strong>
      </div>
      <div className="fact">
        <span>Рабочий контекст</span>
        <strong>{scope}</strong>
      </div>
      <div className="fact">
        <span>Бизнес-тип</span>
        <strong>{c?.userType ? sellerTypeLabel[c.userType] : 'Назначается платформой'}</strong>
      </div>
      <div className="fact">
        <span>Проверка компании</span>
        <strong>{c ? verificationLabel[c.verification] : '—'}</strong>
      </div>
    </div>
  );
}

function CompanyInfoSection({
  c,
  canEdit,
  id,
  refresh,
  revision,
  scope,
  onRefreshStatus,
  loading,
  loadError,
}: {
  c: Company | undefined;
  canEdit: boolean;
  id: string;
  refresh: unknown[][];
  revision: string | undefined;
  scope: string;
  onRefreshStatus: () => Promise<unknown>;
  loading: boolean;
  loadError: unknown;
}) {
  const [busy, setBusy] = useState(false);
  const [error, setError] = useState('');
  const [submittedNeedsRefresh, setSubmittedNeedsRefresh] = useState(false);
  const refreshStatus = async () => {
    setBusy(true);
    setError('');
    try {
      await onRefreshStatus();
      setSubmittedNeedsRefresh(false);
    } catch (err) {
      setError(errorText(err));
    } finally {
      setBusy(false);
    }
  };
  const submitVerification = async () => {
    if (!revision) {
      setError('Не удалось определить актуальную версию компании. Обновите статус и попробуйте снова.');
      return;
    }
    setBusy(true);
    setError('');
    try {
      await post(`/identity/companies/${id}/submit-verification`, {}, { ifMatch: revision });
      try {
        await onRefreshStatus();
      } catch (err) {
        setSubmittedNeedsRefresh(true);
        setError(`Заявка отправлена, но статус не обновился: ${errorText(err)}. Обновите статус перед повторной отправкой.`);
      }
    } catch (err) {
      setError(errorText(err));
    } finally {
      setBusy(false);
    }
  };
  const verificationMessage = c?.verification === 'submitted'
    ? 'Заявка отправлена на проверку платформой. Настройки компании, филиалы, склады и сотрудники остаются доступны.'
    : c?.verification === 'verified'
      ? 'Компания верифицирована. Для коммерческих операций также нужны назначенный бизнес-тип и соответствующие разрешения.'
      : c?.verification === 'returned'
        ? 'Заявка возвращена на доработку. Исправьте или дополните данные компании и отправьте её повторно.'
        : 'Заполните доступные реквизиты компании при необходимости и отправьте её на проверку. Реквизиты остаются необязательными.';
  return (
    <section className="surface">
      <div className="section-head">
        <div className="partner-id">
          <div className="partner-logo">{c ? initials(c.name) : ''}</div>
          <div>
            <strong>{c?.name}</strong>
            <span>{[c?.legalName, c?.registration && `Рег. № ${c.registration}`].filter(Boolean).join(' · ')}</span>
          </div>
        </div>
        <div className="partner-actions">
          <Badge tone={accessTone(c?.access)}>{(c?.access && accessLabel[c.access]) || 'Черновик'}</Badge>
          {c && canEdit && <CompanyEditAction c={c} id={id} refresh={refresh} revision={revision} />}
        </div>
      </div>
      <div className="section-body">
        <Panel title="Верификация компании" padded actions={<Button size="sm" onClick={() => void refreshStatus()} busy={busy}>Обновить статус</Button>}>
          {loading ? <p>Загружаем статус верификации…</p> : <p>{verificationMessage}</p>}
          {loadError && <Notice kind="danger">Не удалось загрузить данные компании: {errorText(loadError)}. Обновите статус, чтобы повторить загрузку.</Notice>}
          {c?.verification === 'returned' && c.verificationReason && <Notice kind="warning">Причина возврата: {c.verificationReason}</Notice>}
          {error && <Notice kind="danger">{error}</Notice>}
          {(c?.verification === 'unverified' || c?.verification === 'returned') && (canEdit ? (
            <div className="kit-row">
              <Button variant="primary" onClick={() => void submitVerification()} busy={busy} disabled={submittedNeedsRefresh || !revision || loading || !!loadError}>
                {c.verification === 'returned' ? 'Отправить повторно' : 'Отправить на верификацию'}
              </Button>
            </div>
          ) : <Notice kind="warning">Для отправки на верификацию нужно разрешение «Редактирование компании».</Notice>)}
        </Panel>
        <CompanyFacts c={c} scope={scope} />
      </div>
    </section>
  );
}

function BranchesPanel({
  c,
  s,
  id,
  refresh,
  branches,
  warehouses,
  setOpen,
  reload,
}: {
  c: Company | undefined;
  s: ReturnType<typeof useSession>;
  id: string;
  refresh: unknown[][];
  branches: ReturnType<typeof useBranches>;
  warehouses: ReturnType<typeof useWarehouses>;
  setOpen: (id: string | null) => void;
  reload: ReturnType<typeof useRefresh>;
}) {
  return (
    <Panel
      title={`Филиалы · ${c?.name ?? ''}`}
      actions={
        s.can('branches.create') && (
          <ActionButton
            small
            variant="primary"
            label="Добавить филиал"
            refresh={refresh}
            fields={branchFields()}
            onSubmit={(v) => post(`/identity/companies/${id}/branches`, v)}
          />
        )
      }
    >
      <Table
        rows={branches.data}
        loading={branches.isLoading}
        error={branches.error}
        rowKey={(b) => b.id}
        empty="Филиалов пока нет"
        columns={[
          { title: 'Филиал', render: (b) => <Cell main={b.name} /> },
          { title: 'Адрес', render: (b) => b.address },
          {
            title: 'Склад',
            render: (b) => {
              const w = warehouses.data?.find((x) => x.branchId === b.id);
              return w ? (
                <>
                  <Cell main={w.name} sub={`${w.occupied} из ${w.capacity} мест`} />
                  <div className="kit-row" style={{ marginTop: 6 }}>
                    <Button size="sm" onClick={() => setOpen(w.id)}>
                      Открыть
                    </Button>
                    <ActionButton
                      small
                      label="Отвязать"
                      onSubmit={async () => {
                        await post(
                          `/inventory/warehouses/${w.id}/branch-attachment`,
                          { branchId: null },
                          { ifMatch: w.revision },
                        );
                        await reload(['warehouses']);
                      }}
                    />
                  </div>
                </>
              ) : (
                <span className="cell-sub">Основной склад не привязан</span>
              );
            },
          },
          {
            title: 'Действия',
            render: (b) =>
              s.can('branches.edit') && (
                <ActionButton
                  small
                  label="Изменить"
                  fields={branchFields(b)}
                  refresh={refresh}
                  onSubmit={(v) => patch(`/identity/companies/${id}/branches/${b.id}`, v, { ifMatch: b.revision })}
                />
              ),
          },
        ]}
      />
    </Panel>
  );
}

/** Opens a new seller company; its creator becomes the company administrator. */
export function AddCompanyAction() {
  const s = useSession();
  return (
    <ActionButton
      label="+ Добавить компанию"
      title="Новая компания"
      submitLabel="Создать"
      variant="primary"
      fields={[
        { name: 'legalForm', label: 'Организационная форма', type: 'select', options: legalFormOptions, searchable: true },
        { name: 'name', label: 'Название компании', type: 'text', required: true },
        {
          name: 'country',
          label: 'Страна',
          type: 'combobox',
          options: countries(),
          canonicalize: canonicalCountry,
          placeholder: 'Выберите или найдите страну',
          ariaLabel: 'Показать страны',
        },
        {
          name: 'region',
          label: 'Регион',
          type: 'combobox',
          dependsOn: 'country',
          optionsFor: regionsFor,
          placeholder: 'Выберите или найдите регион',
          disabledPlaceholder: 'Сначала выберите страну',
          ariaLabel: 'Показать регионы',
        },
        { name: 'address', label: 'Адрес', type: 'text' },
        { name: 'phone', label: 'Телефон', type: 'text' },
        { name: 'email', label: 'Электронная почта', type: 'email' },
      ]}
      intro={<p>Остальные реквизиты можно заполнить позже. Переключиться на компанию можно в шапке кабинета.</p>}
      onSubmit={async (v) => {
        const country = String(v.country ?? '');
        await post('/identity/companies', {
          company: {
            name: v.name,
            legalForm: v.legalForm ?? '',
            country: { label: canonicalCountry(country) ?? country },
            region: v.region ? { label: v.region } : null,
            address: v.address,
            phone: v.phone,
            email: v.email,
          },
        });
        await s.refresh(); // the selector lists the new company
      }}
    />
  );
}

function Companies() {
  const s = useSession();
  const id = s.company?.id ?? '';
  const company = useData(['company', id], () => get<Company>(`/identity/companies/${id}`), !!id);
  const branches = useBranches();
  const warehouses = useWarehouses();
  const scope = useScopeLabel();
  const reload = useRefresh();
  const [open, setOpen] = useState<string | null>(null);
  const c = company.data?.data;
  const refresh = [['company', id], ['branches']];
  return (
    <div className="settings-content">
      <div className="settings-title-row">
        <div>
          <h2>Компания и филиалы</h2>
          <p>
            Изменение данных не меняет рабочий контекст. Новая компания сразу активна и появится в переключателе
            компаний, вы станете её администратором.
          </p>
        </div>
        {s.can('company.create') && <AddCompanyAction />}
      </div>
      <CompanyInfoSection
        c={c}
        canEdit={s.can('company.edit')}
        id={id}
        refresh={refresh}
        revision={company.data?.revision}
        scope={scope}
        loading={company.isLoading}
        loadError={company.error}
        onRefreshStatus={async () => {
          await s.refresh();
          return company.refetch({ throwOnError: true });
        }}
      />
      <BranchesPanel
        c={c}
        s={s}
        id={id}
        refresh={refresh}
        branches={branches}
        warehouses={warehouses}
        setOpen={setOpen}
        reload={reload}
      />
      {open && <WarehouseDialog id={open} onClose={() => setOpen(null)} />}
    </div>
  );
}

function Members() {
  const s = useSession();
  return (
    <div className="settings-content">
      <div className="settings-title-row">
        <div>
          <h2>Пользователи и роли</h2>
          <p>Роли компании (набор разрешений) и сотрудники, которым они назначены.</p>
        </div>
      </div>
      <CompanyRoles />
      <CompanyEmployees />
      <Panel title="Ваш доступ" padded>
        <Details
          items={[
            ['Роли', s.view.roles.map((r) => r.name).join(', ') || '—'],
            ['Разрешений', String(s.view.permissions.length)],
          ]}
        />
      </Panel>
    </div>
  );
}

function Account() {
  const s = useSession();
  const [tab, setTab] = useState<'password' | null>(null);
  return (
    <div className="settings-content">
      <div className="settings-title-row">
        <div>
          <h2>Личный аккаунт</h2>
          <p>Профиль и пароль</p>
        </div>
      </div>
      <Panel title="Профиль" padded>
        <Details
          items={[
            ['Имя', s.view.user.displayName],
            ['Статус', s.view.user.status === 'active' ? 'Активен' : s.view.user.status],
          ]}
        />
      </Panel>
      <Panel title="Безопасность" padded>
        <div className="kit-row">
          <Button onClick={() => setTab('password')}>Сменить пароль</Button>
        </div>
        {tab === 'password' && (
          <div style={{ marginTop: 16, maxWidth: 420 }}>
            <ChangePassword
              onDone={() => {
                void s.refresh();
                setTab(null);
              }}
            />
          </div>
        )}
      </Panel>
    </div>
  );
}
