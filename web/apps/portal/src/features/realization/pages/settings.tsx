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
import './settings.css';

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
type Section = 'company' | 'branches' | 'members' | 'account';
const sections: [Section, string, string][] = [
  ['company', 'Компания', 'Реквизиты и верификация'],
  ['branches', 'Филиалы и склады', 'Точки продаж и склады'],
  ['members', 'Сотрудники и роли', 'Доступ сотрудников'],
  ['account', 'Личный профиль', 'Профиль и безопасность'],
];
const initials = (n: string) =>
  n
    .split(/\s+/)
    .slice(0, 2)
    .map((w) => w[0]?.toUpperCase())
    .join('');

export function SettingsPage() {
  const [section, setSection] = useState<Section>('company');
  return (
    <section className="page realization-settings">
      <div className="page-header">
        <div>
          <h1 className="page-title">Настройки</h1>
          <div className="page-subtitle">Компания, филиалы, склады, доступ сотрудников и личная безопасность</div>
        </div>
      </div>
      <div className="realization-settings__layout">
        <nav className="realization-settings__nav" aria-label="Разделы настроек">
          {sections.map(([k, title, sub]) => (
            <button
              key={k}
              type="button"
              className={`realization-settings__nav-item${section === k ? ' is-active' : ''}`}
              aria-current={section === k ? 'page' : undefined}
              onClick={() => setSection(k)}
            >
              <strong>{title}</strong>
              <span>{sub}</span>
            </button>
          ))}
        </nav>
        <div className="realization-settings__content">
          {section === 'company' ? <CompanySection /> : section === 'branches' ? <BranchesSection /> : section === 'members' ? <Members /> : <Account />}
        </div>
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
const valueOrEmpty = (value: string | undefined | null) => value?.trim() || 'Не указано';

function CompanyFacts({ c }: { c: Company | undefined }) {
  const countryAndRegion = [c?.country?.label, c?.region?.label].filter(Boolean).join(', ');
  return (
    <div className="realization-settings__fact-groups">
      <section className="realization-settings__fact-group" aria-labelledby="company-requisites-title">
        <h3 id="company-requisites-title">Реквизиты</h3>
        <dl>
          <div><dt>Юридическое название</dt><dd>{valueOrEmpty(c?.legalName)}</dd></div>
          <div><dt>Регистрационный номер</dt><dd>{valueOrEmpty(c?.registration)}</dd></div>
          <div><dt>ИНН / БИН</dt><dd>{valueOrEmpty(c?.taxId)}</dd></div>
          <div><dt>Банковские реквизиты</dt><dd>{valueOrEmpty(c?.bankRequisites)}</dd></div>
        </dl>
      </section>
      <section className="realization-settings__fact-group" aria-labelledby="company-contacts-title">
        <h3 id="company-contacts-title">Контакты</h3>
        <dl>
          <div><dt>Страна и регион</dt><dd>{countryAndRegion || 'Не указано'}</dd></div>
          <div><dt>Юридический адрес</dt><dd>{valueOrEmpty(c?.address)}</dd></div>
          <div><dt>Телефон</dt><dd>{valueOrEmpty(c?.phone)}</dd></div>
          <div><dt>Email</dt><dd>{valueOrEmpty(c?.email)}</dd></div>
        </dl>
      </section>
    </div>
  );
}

function CompanyInfoSection({
  c,
  canEdit,
  id,
  refresh,
  revision,
  onRefreshStatus,
  loading,
  loadError,
}: {
  c: Company | undefined;
  canEdit: boolean;
  id: string;
  refresh: unknown[][];
  revision: string | undefined;
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
    ? 'Заявка принята. Дождитесь результата проверки платформы и обновите статус позже.'
    : c?.verification === 'verified'
      ? 'Компания проверена'
      : c?.verification === 'returned'
        ? 'Проверьте причину возврата, исправьте данные и отправьте заявку повторно.'
        : 'Заполните данные компании и отправьте заявку на проверку.';
  return (
    <section className="surface realization-settings__company-card">
      <div className="section-head realization-settings__company-head">
        <div className="partner-id realization-settings__company-identity">
          <div className="partner-logo">{c ? initials(c.name) : ''}</div>
          <div>
            <span className="realization-settings__legal-form">{c ? legalFormLabel(c.legalForm) : 'Организационная форма не указана'}</span>
            <strong>{c?.name || 'Компания не выбрана'}</strong>
            <span>{c?.legalName || 'Юридическое название не указано'}</span>
          </div>
        </div>
        <div className="partner-actions realization-settings__company-actions">
          <span className="realization-settings__status-label">Доступ к кабинету <Badge tone={accessTone(c?.access)}>{(c?.access && accessLabel[c.access]) || 'Черновик'}</Badge></span>
          <span className="realization-settings__status-label">Тип работы <strong>{c?.userType ? sellerTypeLabel[c.userType] : 'Не назначен'}</strong></span>
          {c && canEdit && <CompanyEditAction c={c} id={id} refresh={refresh} revision={revision} />}
        </div>
      </div>
      <div className="section-body realization-settings__company-body">
        <Panel title={c?.verification === 'verified' ? 'Верификация пройдена' : 'Верификация компании'} padded actions={<Button size="sm" onClick={() => void refreshStatus()} busy={busy}>Обновить статус</Button>}>
          {loading ? <p>Загружаем статус верификации…</p> : <p className={c?.verification === 'verified' ? 'realization-settings__verified-message' : undefined}>{verificationMessage}</p>}
          {loadError && <Notice kind="danger">Не удалось загрузить данные компании: {errorText(loadError)}. Обновите статус, чтобы повторить загрузку.</Notice>}
          {c?.verification === 'verified' && !c.userType && <Notice kind="warning">Бизнес-тип ещё не назначен платформой.</Notice>}
          {c?.verification === 'verified' && c.access !== 'active' && <Notice kind="warning">Доступ к кабинету сейчас не активен.</Notice>}
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
        <CompanyFacts c={c} />
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

function CompanySection() {
  const s = useSession();
  const id = s.company?.id ?? '';
  const company = useData(['company', id], () => get<Company>(`/identity/companies/${id}`), !!id);
  const c = company.data?.data;
  const refresh = [['company', id], ['branches']];
  return (
    <div className="realization-settings__section">
      <div className="realization-settings__title-row">
        <div>
          <h2>Компания</h2>
          <p>Юридические данные, контакты и статус верификации выбранной компании.</p>
        </div>
        {s.can('company.create') && <AddCompanyAction />}
      </div>
      <CompanyInfoSection
        c={c}
        canEdit={s.can('company.edit')}
        id={id}
        refresh={refresh}
        revision={company.data?.revision}
        loading={company.isLoading}
        loadError={company.error}
        onRefreshStatus={async () => {
          await s.refresh();
          return company.refetch({ throwOnError: true });
        }}
      />
    </div>
  );
}

function BranchesSection() {
  const s = useSession();
  const id = s.company?.id ?? '';
  const scope = useScopeLabel();
  const company = useData(['company', id], () => get<Company>(`/identity/companies/${id}`), !!id);
  const branches = useBranches();
  const warehouses = useWarehouses();
  const reload = useRefresh();
  const [open, setOpen] = useState<string | null>(null);
  return (
    <div className="realization-settings__section">
      <div className="realization-settings__title-row">
        <div>
          <h2>Филиалы и склады</h2>
          <p>Рабочий контекст: {scope}. Управляйте точками продаж и их складами отдельно от реквизитов компании.</p>
        </div>
      </div>
      <BranchesPanel
        c={company.data?.data}
        s={s}
        id={id}
        refresh={[['company', id], ['branches']]}
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
    <div className="realization-settings__section">
      <div className="realization-settings__title-row">
        <div>
          <h2>Сотрудники и роли</h2>
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
    <div className="realization-settings__section">
      <div className="realization-settings__title-row">
        <div>
          <h2>Личный профиль</h2>
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
