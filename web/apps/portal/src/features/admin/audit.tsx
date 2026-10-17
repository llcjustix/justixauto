import { useEffect, useRef, useState } from 'react';
import type { ReactNode } from 'react';
import { useInfiniteQuery } from '@tanstack/react-query';
import { Badge, Button, Cell, Modal, Notice, Page, Panel, Table, dateTime, get, listPage, useData, useSession } from '@justixauto/kit';

export interface AuditEvent {
  id: string;
  occurredAt: string;
  actorId: string | null;
  actorLogin?: string | null;
  actorDisplayName?: string | null;
  action: string;
  resourceType: string;
  resourceId: string | null;
  companyId: string | null;
  reason: string;
  outcome?: string | null;
  errorCode?: string | null;
  requestId?: string | null;
  clientIp?: string | null;
  sessionHash?: string | null;
  deviceHash?: string | null;
  details: unknown;
  metadata?: unknown;
  before?: unknown;
  after?: unknown;
  integrity?: {
    version: number | null;
    streamId: string | null;
    index: string | null;
    prevHash: string | null;
    hash: string | null;
    status: string;
  };
}

const actionLabels: Record<string, string> = {
  'auth.login_succeeded': 'Вход выполнен', 'auth.login_failed': 'Вход отклонён',
  'auth.logout': 'Выход из системы', 'auth.session_expired': 'Срок сеанса истёк',
  'auth.session_revoked': 'Сеанс отозван', 'context.company_changed': 'Рабочая компания изменена',
  'context.branch_scope_changed': 'Рабочие филиалы изменены', 'security.request_denied': 'Запрос отклонён',
  'company.created': 'Компания создана', 'company.updated': 'Реквизиты изменены',
  'company.activate': 'Доступ компании активирован', 'company.activated': 'Доступ компании активирован',
  'company.suspend': 'Доступ компании приостановлен', 'company.suspended': 'Доступ компании приостановлен',
  'company.restore': 'Доступ компании восстановлен', 'company.restored': 'Доступ компании восстановлен',
  'company.deleted': 'Компания удалена', 'company.seller_provisioned': 'Продавец подключён',
  'company.provider_provisioned': 'Подключена организация', 'company.user_type_set': 'Тип продавца назначен',
  'company.verification_submitted': 'Компания отправлена на проверку',
  'company.verification_verified': 'Компания проверена', 'company.verification_returned': 'Проверка компании возвращена',
  'branch.created': 'Филиал создан', 'branch.updated': 'Филиал изменён',
  'user.created': 'Пользователь создан', 'user.updated': 'Пользователь изменён',
  'user.suspended': 'Пользователь приостановлен', 'user.restored': 'Пользователь восстановлен',
  'user.password_changed': 'Пароль изменён пользователем', 'user.password_set': 'Пароль задан администратором',
  'user.sessions_revoked': 'Сеансы завершены', 'user.bootstrapped': 'Первый администратор платформы',
  'membership.granted': 'Доступ к компании выдан', 'membership.revoked': 'Доступ к компании отозван',
  'membership.branch_access_changed': 'Доступ к филиалам изменён',
  'role.created': 'Роль создана', 'role.updated': 'Роль изменена', 'role.deleted': 'Роль удалена',
  'permission.created': 'Разрешение добавлено', 'permission.updated': 'Разрешение изменено',
  'permission.deleted': 'Разрешение удалено',
  // Historical labels describe retained records, without implying current MFA support.
  'user.mfa_enrolled': 'Включена двухфакторная защита', 'mfa.enrolled': 'Включена двухфакторная защита',
  'password.changed': 'Пароль изменён',
};

const fieldLabels: Record<string, string> = {
  id: 'ID записи', occurredAt: 'Время события (UTC)', actorId: 'ID сотрудника',
  actorLogin: 'Логин сотрудника на момент события', actorDisplayName: 'Имя сотрудника на момент события',
  action: 'Действие', resourceType: 'Тип объекта', resourceId: 'ID объекта', companyId: 'ID компании',
  reason: 'Основание', outcome: 'Результат', errorCode: 'Код отказа / ошибки', requestId: 'ID запроса',
  clientIp: 'IP клиента', sessionHash: 'Хеш сеанса', deviceHash: 'Хеш корреляции браузера',
  method: 'HTTP метод', route: 'Шаблон маршрута', network: 'Сетевые сведения', backendHttp: 'HTTP соединения с API',
  backendTls: 'TLS соединения с API', metadataTruncated: 'Общий объём метаданных ограничен',
  peerIp: 'IP узла соединения с API', peerPort: 'Порт узла соединения с API', clientPort: 'Порт клиента',
  clientIpSource: 'Источник IP клиента', extractionStatus: 'Результат извлечения IP',
  version: 'Версия', cipher: 'Шифр', alpn: 'ALPN', http: 'HTTP на внешнем узле',
  tlsVersion: 'Версия TLS', tlsCipher: 'Шифр TLS', ja3: 'JA3', ja4: 'JA4', sourcePort: 'Исходный порт',
  status: 'Состояние', source: 'Источник', value: 'Значение', truncated: 'Значение усечено',
  userAgent: 'User-Agent', uaBrands: 'Бренды браузера (Sec-CH-UA)', uaMobile: 'Признак мобильного браузера',
  uaPlatform: 'Платформа (Sec-CH-UA-Platform)', uaFullVersionList: 'Полные версии браузера',
  uaPlatformVersion: 'Версия платформы', uaModel: 'Модель со слов клиента', uaArch: 'Архитектура со слов клиента',
  acceptLanguage: 'Языки клиента', origin: 'Origin', referer: 'Referer (без query и fragment)',
  fetchSite: 'Sec-Fetch-Site', fetchMode: 'Sec-Fetch-Mode', fetchDest: 'Sec-Fetch-Dest', fetchUser: 'Sec-Fetch-User',
  timezone: 'Часовой пояс', appVersion: 'Версия приложения со слов клиента',
  screen: 'Экран: ширина × высота @ DPR', forwardedFor: 'X-Forwarded-For (непроверенная цепочка)',
  login: 'Логин', displayName: 'Имя', sessionStartedAt: 'Начало сеанса (UTC)',
  authenticationMethod: 'Способ аутентификации', knownDevice: 'Корреляция ранее встречалась у пользователя',
  streamId: 'ID потока', index: 'Индекс цепочки', prevHash: 'Предыдущий хеш', hash: 'Хеш записи',
  failureReason: 'Причина отказа входа', attemptedLogin: 'Введённый логин (со слов клиента)',
  attemptedLoginSource: 'Источник введённого логина', attemptedLoginTruncated: 'Введённый логин усечён',
  targetLogin: 'Логин целевого пользователя', sessionEndReason: 'Причина завершения сеанса',
  name: 'Название', legalName: 'Юридическое название', country: 'Страна', countryKey: 'Код страны',
  region: 'Регион', regionKey: 'Код региона', registration: 'Регистрационные сведения', legalForm: 'Организационная форма',
  taxId: 'ИНН / БИН', bankRequisites: 'Банковские реквизиты', email: 'Электронная почта', address: 'Адрес',
  phone: 'Телефон', kind: 'Вид компании', statusReason: 'Основание изменения состояния',
  userType: 'Тип продавца', verification: 'Проверка компании', verificationReason: 'Основание проверки',
  verificationActorId: 'ID проверяющего', deleted: 'Удалено', roleIds: 'ID ролей', roleId: 'ID роли',
  permissionKeys: 'Ключи разрешений', permissions: 'Разрешения', permissionsBefore: 'Разрешения до',
  permissionsAfter: 'Разрешения после', permissionsAdded: 'Добавленные разрешения', permissionsRemoved: 'Удалённые разрешения',
  changedFields: 'Изменённые поля', membershipId: 'ID доступа к компании', userId: 'ID пользователя',
  branchAccess: 'Доступ к филиалам', branchIds: 'ID филиалов', branchScope: 'Рабочие филиалы',
  access: 'Доступ', mode: 'Режим', key: 'Ключ', label: 'Название', scope: 'Область', assignable: 'Можно назначать',
  rolesBefore: 'Роли до', rolesAfter: 'Роли после', adminUserId: 'ID администратора', adminLogin: 'Логин администратора',
  before: 'До', after: 'После',
};

const valueLabels: Record<string, string> = {
  success: 'Успешно', denied: 'Отклонено', error: 'Ошибка', client_supplied: 'Со слов клиента',
  network_peer: 'Узел соединения с API', trusted_proxy_xff: 'Граница доверенного прокси по XFF',
  not_configured: 'Не настроено', invalid_peer: 'Некорректный адрес узла', untrusted_peer: 'Узел не является доверенным прокси',
  missing_forwarded_header: 'Заголовок XFF отсутствует', forwarded_too_large: 'Превышен размер XFF',
  too_many_forwarded_hops: 'Превышено число узлов XFF', malformed_forwarded_header: 'Некорректный XFF',
  first_untrusted_hop: 'Выбран первый недоверенный узел от API', no_untrusted_hop: 'Недоверенный узел не найден; адрес узла API',
  captured: 'Получено', sanitized: 'Очищено', invalid: 'Некорректно', truncated: 'Усечено', omitted_budget: 'Пропущено из-за лимита',
  combined: 'Объединены несколько заголовков', combined_sanitized: 'Заголовки объединены и очищены',
  chained_unverified: 'Локальная цепочка; проверка не выполнена', prechain_unverified: 'Историческая запись вне цепочки; не проверена',
  unknown_login: 'Логин не найден', no_credentials: 'Нет учётных данных', bad_password: 'Неверный пароль',
  locked: 'Вход заблокирован', suspended: 'Доступ приостановлен', internal_error: 'Внутренняя ошибка',
  unauthenticated: 'Не выполнен вход', permission_denied: 'Недостаточно разрешений', origin_denied: 'Origin отклонён',
  csrf_invalid: 'Проверка CSRF не пройдена', password_change_required: 'Требуется смена пароля',
  invalid_credentials: 'Неверные учётные данные', account_locked: 'Учётная запись заблокирована',
  account_suspended: 'Доступ пользователя приостановлен', not_a_member: 'Нет доступа к компании',
  company_suspended: 'Доступ компании приостановлен', no_active_company: 'Рабочая компания не выбрана',
  context_changed: 'Рабочий контекст изменился', branch_scope_denied: 'Нет доступа к филиалам',
  password: 'Пароль', unknown: 'Неизвестно', dev: 'Сборка для разработки; коммит не указан',
  active: 'Активно', draft: 'Черновик', unverified: 'Не проверено', submitted: 'На проверке',
  verified: 'Проверено', returned: 'Возвращено', all: 'Все', selected: 'Выбранные', ALL: 'Все', SELECTED: 'Выбранные',
};
const enumFields = new Set(['outcome', 'errorCode', 'clientIpSource', 'extractionStatus', 'status', 'source',
  'failureReason', 'attemptedLoginSource', 'sessionEndReason', 'authenticationMethod', 'verification', 'mode', 'access']);

const object = (value: unknown): Record<string, unknown> =>
  value !== null && typeof value === 'object' && !Array.isArray(value) ? value as Record<string, unknown> : {};
const claim = (event: AuditEvent, key: string) => {
  const text = object(object(object(event.metadata).clientSupplied)[key]).value;
  return typeof text === 'string' ? text.slice(0, 512) : '';
};

function deviceSummary(event: AuditEvent) {
  const ua = claim(event, 'userAgent');
  const brands = claim(event, 'uaBrands');
  const browser = /Microsoft Edge|Edg\//i.test(`${brands} ${ua}`) ? 'Edge'
    : /Opera|OPR\//i.test(`${brands} ${ua}`) ? 'Opera'
    : /Firefox\//i.test(ua) ? 'Firefox'
    : /Google Chrome|Chromium|Chrome\//i.test(`${brands} ${ua}`) ? 'Chrome / Chromium'
    : /Version\/[\d.]+.*Safari\//i.test(ua) ? 'Safari' : 'Браузер неизвестен';
  const platform = claim(event, 'uaPlatform').replace(/^"|"$/g, '');
  const os = ['Windows', 'macOS', 'Android', 'iOS', 'Linux', 'Chrome OS'].find((name) => platform === name)
    ?? (/Android/i.test(ua) ? 'Android' : /iPhone|iPad|iPod/i.test(ua) ? 'iOS'
      : /Windows NT/i.test(ua) ? 'Windows' : /Macintosh|Mac OS X/i.test(ua) ? 'macOS'
      : /CrOS/i.test(ua) ? 'Chrome OS' : /Linux/i.test(ua) ? 'Linux' : 'ОС неизвестна');
  return `${browser} · ${os}`;
}

function actorLabel(event: AuditEvent) {
  if (event.actorLogin) return event.actorLogin;
  if (event.actorDisplayName) return event.actorDisplayName;
  if (event.actorId) return event.actorId;
  if (event.action === 'user.bootstrapped') return 'Системная инициализация';
  return event.action.startsWith('auth.') || event.action.startsWith('security.') || event.metadata
    ? 'Не установлен' : 'Неизвестен (историческая запись)';
}

function Value({ value, field }: { value: unknown; field?: string }) {
  if (value === null || value === undefined) return <span className="cell-sub">Нет данных</span>;
  if (Array.isArray(value)) return value.length ? <ul>{value.map((v, i) => <li key={i}><Value value={v} {...(field ? { field } : {})} /></li>)}</ul> : <span>Пустой список</span>;
  if (typeof value === 'object') return <Fields value={value} />;
  if (typeof value === 'boolean') return <span>{value ? 'Да' : 'Нет'}</span>;
  const raw = String(value);
  const label = field === 'action' ? actionLabels[raw] : field === 'changedFields' ? fieldLabels[raw]
    : field && enumFields.has(field) ? valueLabels[raw] : undefined;
  return <span style={{ whiteSpace: 'pre-wrap', overflowWrap: 'anywhere' }}>{label ? `${label} (${raw})` : raw || 'Пустое значение'}</span>;
}

function Fields({ value, empty = 'Нет данных' }: { value: unknown; empty?: string }) {
  const entries = Object.entries(object(value));
  return entries.length ? <dl style={{ margin: 0 }}>{entries.map(([key, v]) => (
    <div key={key} style={{ padding: '8px 0', borderBottom: '1px solid var(--border, #e5e7eb)' }}>
      <dt className="cell-sub">{fieldLabels[key] ?? key} <code>({key})</code></dt>
      <dd style={{ margin: '4px 0 0', overflowWrap: 'anywhere' }}><Value value={v} field={key} /></dd>
    </div>
  ))}</dl> : <p className="cell-sub">{empty}</p>;
}

function Group({ title, children }: { title: string; children: ReactNode }) {
  return <section style={{ marginBottom: 24 }}><h3>{title}</h3>{children}</section>;
}

function AuditDetail({ event, onClose }: { event: AuditEvent; onClose: () => void }) {
  const session = useSession();
  const q = useData(['admin-audit-detail', session.scopeKey, event.id], () => get<AuditEvent>(`/identity/admin/audit/${encodeURIComponent(event.id)}`));
  const root = useRef<HTMLDivElement>(null);
  const [opener] = useState(() => typeof document === 'undefined' ? null : document.activeElement);
  useEffect(() => {
    const dialog = root.current?.closest<HTMLElement>('[role="dialog"]');
    const trap = (e: KeyboardEvent) => {
      if (e.key !== 'Tab' || !dialog) return;
      const controls = [...dialog.querySelectorAll<HTMLElement>('button:not(:disabled),a[href],input:not(:disabled),select:not(:disabled),textarea:not(:disabled),[tabindex="0"]')];
      const first = controls[0], last = controls[controls.length - 1];
      if (e.shiftKey && (document.activeElement === first || !dialog.contains(document.activeElement))) { e.preventDefault(); last?.focus(); }
      else if (!e.shiftKey && (document.activeElement === last || !dialog.contains(document.activeElement))) { e.preventDefault(); first?.focus(); }
    };
    document.addEventListener('keydown', trap);
    return () => { document.removeEventListener('keydown', trap); if (opener instanceof HTMLElement && opener.isConnected) opener.focus(); };
  }, [opener]);
  const e = q.data?.data ?? event;
  const metadata = object(e.metadata);
  const { clientSupplied, auth, edge, ...server } = metadata;
  const before = object(e.before), after = object(e.after);
  const changed = [...new Set([...Object.keys(before), ...Object.keys(after)])];
  return <Modal title="Сведения о действии" size="wide" onClose={onClose} footer={<Button onClick={onClose}>Закрыть</Button>}>
    <div ref={root} style={{ minWidth: 0 }}>
      {q.isLoading && <p role="status">Загрузка полной записи…</p>}
      {q.error && <><Notice kind="danger">Не удалось обновить запись. Ниже показаны сведения из журнала.</Notice><Button onClick={() => void q.refetch()}>Повторить</Button></>}
      <Group title="Событие и участник">
        <p><strong>{actionLabels[e.action] ?? e.action}</strong> · {actorLabel(e)}</p>
        <p>Точное время UTC: <time dateTime={e.occurredAt} style={{ overflowWrap: 'anywhere' }}>{e.occurredAt}</time></p>
        <Fields value={{ id: e.id, action: e.action, outcome: e.outcome, errorCode: e.errorCode, reason: e.reason,
          resourceType: e.resourceType, resourceId: e.resourceId, companyId: e.companyId, actorId: e.actorId,
          actorLogin: e.actorLogin, actorDisplayName: e.actorDisplayName }} />
      </Group>
      <Group title="Сервер и сеть">
        <p className="cell-sub">Адрес узла и протокол относятся к соединению с API. IP клиента зависит от указанного источника и настройки доверенных прокси.</p>
        {!Object.keys(metadata).length && <p className="cell-sub">Метаданные не собирались для этой записи.</p>}
        <Fields value={{ requestId: e.requestId, clientIp: e.clientIp }} />
        <h4>Снимок запроса</h4>
        <Fields value={server} empty="Метаданные не собирались для этой записи." />
      </Group>
      <Group title="Со слов клиента">
        <p>{deviceSummary(e)}</p>
        <p className="cell-sub">Заголовки могут быть подменены, усечены или отсутствовать. Сводка браузера и ОС приблизительная; это не идентификация физического устройства. Версия приложения — заявление клиента.</p>
        <Fields value={clientSupplied} empty="Клиентские сведения не собирались или не были переданы." />
      </Group>
      <Group title="Сеанс и корреляция браузера">
        <p className="cell-sub">Хеши позволяют сопоставлять записи. Cookie браузера можно удалить или скопировать; распознавание не является фактором аутентификации.</p>
        <Fields value={{ sessionHash: e.sessionHash, deviceHash: e.deviceHash }} />
        <h4>Серверный снимок аутентификации</h4>
        <Fields value={auth} empty="Снимок аутентификации не собирался или участник не установлен." />
      </Group>
      <Group title="Изменённые значения">
        {changed.length ? <div className="table-wrap"><table><thead><tr><th>Поле</th><th>До</th><th>После</th></tr></thead><tbody>
          {changed.map((key) => <tr key={key}><td>{fieldLabels[key] ?? key}<br /><code>{key}</code></td><td><Value value={before[key]} field={key} /></td><td><Value value={after[key]} field={key} /></td></tr>)}
        </tbody></table></div> : <p className="cell-sub">Значения до и после не собирались или неприменимы.</p>}
        {(e.before !== null && e.before !== undefined && !Object.keys(before).length) && <><h4>До</h4><Value value={e.before} /></>}
        {(e.after !== null && e.after !== undefined && !Object.keys(after).length) && <><h4>После</h4><Value value={e.after} /></>}
      </Group>
      <Group title="Дополнительные сведения события"><Value value={e.details} /></Group>
      <Group title="Целостность записи">
        <Notice>Наличие хешей не означает успешную проверку. Проверка цепочки и независимого внешнего контрольного значения здесь не выполнялась.</Notice>
        <Fields value={e.integrity} empty="Историческая запись: сведения о цепочке не собирались; целостность не проверена." />
      </Group>
      <Group title="Внешний узел и обогащение">
        <Fields value={edge} empty="Измерения внешнего узла не настроены." />
        <p className="cell-sub">Геолокация, ASN, VPN/proxy/Tor-оценка, внешнее архивирование и уведомления не настроены. Данных о доставке во внешнюю систему нет.</p>
      </Group>
    </div>
  </Modal>;
}

export function AuditTable({ rows, loading, error, compact = false, onRetry }: {
  rows: AuditEvent[] | undefined; loading: boolean; error: unknown; compact?: boolean; onRetry?: () => void;
}) {
  const { scopeKey } = useSession();
  const [selected, select] = useState<{ event: AuditEvent; scope: string } | null>(null);
  const open = (event: AuditEvent) => select({ event, scope: scopeKey });
  return <>
    <Table rows={rows} loading={loading} error={error} rowKey={(e) => e.id} onRowClick={open} empty="Действий пока нет"
      columns={[
        { title: 'Время', render: (e) => <time dateTime={e.occurredAt} title={e.occurredAt}>{dateTime(e.occurredAt)}</time> },
        { title: 'Сотрудник', render: (e) => <Cell main={actorLabel(e)} sub={e.actorLogin ? e.actorDisplayName ?? undefined : undefined} /> },
        { title: 'Действие', render: (e) => <Cell main={actionLabels[e.action] ?? e.action} sub={e.reason ? `Основание: ${e.reason}` : e.action} /> },
        ...(compact ? [] : [
          { title: 'Результат', render: (e: AuditEvent) => <Badge tone={e.outcome === 'success' ? 'success' : e.outcome === 'denied' || e.outcome === 'error' ? 'danger' : undefined}>{e.outcome ? valueLabels[e.outcome] ?? e.outcome : 'Нет данных'}</Badge> },
          { title: 'IP', render: (e: AuditEvent) => e.clientIp ?? 'Нет данных' },
          { title: 'Браузер / ОС', render: (e: AuditEvent) => <Cell main={deviceSummary(e)} sub={`Со слов клиента · корреляция: ${e.deviceHash ? e.deviceHash.slice(0, 12) + '…' : 'нет данных'}`} /> },
        ]),
        { title: 'Компания', render: (e) => <span title={e.companyId ?? undefined}>{e.companyId ? e.companyId.slice(0, 8) + '…' : '—'}</span> },
        { title: '', render: (e) => <span onClick={(ev) => ev.stopPropagation()} onKeyDown={(ev) => ev.stopPropagation()}><Button size="sm" title={`Подробности: ${actionLabels[e.action] ?? e.action}`} onClick={() => open(e)}>Подробнее</Button></span> },
      ]} />
    {Boolean(error) && onRetry && <Button onClick={onRetry}>Повторить</Button>}
    {selected?.scope === scopeKey && <AuditDetail key={selected.event.id} event={selected.event} onClose={() => select(null)} />}
  </>;
}

function Journal() {
  const { scopeKey } = useSession();
  const [draft, setDraft] = useState({ actor: '', ip: '', action: '' });
  const [filters, setFilters] = useState(draft);
  const query = useInfiniteQuery({
    queryKey: ['admin-audit-journal', scopeKey, filters], initialPageParam: '',
    queryFn: ({ pageParam }) => {
      const params = new URLSearchParams({ limit: '50' });
      for (const [key, value] of Object.entries(filters)) if (value.trim()) params.set(key, value.trim());
      if (pageParam) params.set('cursor', pageParam);
      return listPage<AuditEvent>(`/identity/admin/audit?${params}`);
    },
    getNextPageParam: (page) => page.nextCursor || undefined,
    // An existing cursor belongs to its immutable first-page fence.
    refetchOnWindowFocus: false,
  });
  const rows = [...new Map(query.data?.pages.flatMap((page) => page.items).map((event) => [event.id, event] as const) ?? []).values()];
  return <Page title="Журнал действий" subtitle="История изменений и событий безопасности. Откройте строку для всех сведений.">
    <Panel>
      <form className="toolbar" onSubmit={(e) => { e.preventDefault(); setFilters({ ...draft }); }}>
        <label className="search-field">Сотрудник<input aria-label="Сотрудник: логин или имя" placeholder="Логин или имя" maxLength={128} value={draft.actor} onChange={(e) => setDraft({ ...draft, actor: e.target.value })} /></label>
        <label className="search-field">IP<input aria-label="IP клиента" placeholder="Точный IPv4 или IPv6" value={draft.ip} onChange={(e) => setDraft({ ...draft, ip: e.target.value })} /></label>
        <label className="search-field">Действие<input aria-label="Действие" list="audit-actions" placeholder="Все действия / ключ" maxLength={128} value={draft.action} onChange={(e) => setDraft({ ...draft, action: e.target.value })} /><datalist id="audit-actions">{Object.entries(actionLabels).map(([key, label]) => <option key={key} value={key}>{label}</option>)}</datalist></label>
        <Button type="submit">Применить</Button>
        <Button onClick={() => { const empty = { actor: '', ip: '', action: '' }; setDraft(empty); setFilters(empty); }}>Сбросить</Button>
      </form>
      <AuditTable rows={query.data ? rows : undefined} loading={query.isLoading} error={query.isFetchNextPageError ? null : query.error} onRetry={() => void query.refetch()} />
      {query.isFetchNextPageError && <Notice kind="danger">Не удалось загрузить следующую страницу. Уже загруженные записи сохранены.</Notice>}
      {query.hasNextPage && <div style={{ padding: 16 }}><Button busy={query.isFetchingNextPage} onClick={() => void query.fetchNextPage()}>{query.isFetchNextPageError ? 'Повторить загрузку' : 'Загрузить ещё'}</Button></div>}
      {query.data?.pages[0]?.asOf && <p className="cell-sub" style={{ padding: 16 }}>Срез журнала: {query.data.pages[0].asOf}</p>}
    </Panel>
  </Page>;
}

export function AuditPage() {
  const { scopeKey } = useSession();
  return <Journal key={scopeKey} />;
}
