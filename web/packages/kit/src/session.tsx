import { createContext, useCallback, useContext, useEffect, useMemo, useRef, useState } from 'react';
import type { ReactNode } from 'react';
import { flushSync } from 'react-dom';
import { useQueryClient } from '@tanstack/react-query';
import { ApiError, admitHttpContext, call, configureHttp, errorText, retireHttpContext, setCsrf, supersedeHttpRequests } from './http';
import { Button, Card, Field, Notice, css } from './ui';

export interface Company {
  id: string;
  name: string;
  kind: 'seller' | 'bank' | 'mfo' | 'insurance';
  access: string;
  userType: 'dealer' | 'distributor' | null;
  verification: 'unverified' | 'submitted' | 'verified' | 'returned';
  verificationReason: string;
}
export interface SessionView {
  user: { id: string; displayName: string; status: string; passwordChangeRequired: boolean };
  roles: { id: string; name: string }[];
  permissions: string[];
  context: {
    revision: string;
    companyId: string | null;
    branchScope: { mode: 'ALL' | 'SELECTED'; branchIds: string[] };
  };
  accessibleCompanies: Company[];
  setup: { next: 'company' | 'branch' | 'none' };
}

interface SessionApi {
  view: SessionView;
  /** Opaque identity of the admitted user, company and branch scope. */
  scopeKey: string;
  company: Company | undefined;
  can: (permission: string) => boolean;
  refresh: () => Promise<void>;
  logout: () => Promise<void>;
  selectCompany: (id: string | null) => Promise<void>;
  setBranchScope: (mode: 'ALL' | 'SELECTED', branchIds: string[]) => Promise<void>;
}

export interface CommercialAccess {
  allowed: boolean;
  reason: string;
}

const Ctx = createContext<SessionApi | null>(null);

export function useSession(): SessionApi {
  const s = useContext(Ctx);
  if (!s) throw new Error('useSession outside SessionGate');
  return s;
}

/** New commercial operations require a verified seller company. */
export function useCommercialAccess(): CommercialAccess {
  const session = useContext(Ctx);
  const company = session?.company;
  if (!company || company.kind !== 'seller' || company.verification === 'verified') {
    return { allowed: true, reason: '' };
  }
  if (!company.userType) {
    return { allowed: false, reason: 'Администратор платформы ещё не назначил тип вашей компании.' };
  }
  if (company.verification === 'submitted') {
    return { allowed: false, reason: 'Профиль компании отправлен на проверку. Новые коммерческие операции станут доступны после подтверждения.' };
  }
  if (company.verification === 'returned') {
    return { allowed: false, reason: `Проверка компании возвращена: ${company.verificationReason || 'исправьте данные профиля и отправьте его повторно.'}` };
  }
  return { allowed: false, reason: 'Заполните профиль компании и отправьте его на проверку, чтобы начать новые коммерческие операции.' };
}

type State =
  | { kind: 'loading' }
  | { kind: 'error'; message: string }
  | { kind: 'anonymous' }
  | { kind: 'ready'; view: SessionView; scopeKey: string; grantsKey: string; mountKey: number };

const scopeOf = (view: SessionView) => JSON.stringify([
  view.user.id,
  view.context.companyId,
  view.context.branchScope.mode,
  [...view.context.branchScope.branchIds].sort(),
]);
const grantsOf = (view: SessionView) => JSON.stringify([
  view.user.status,
  view.user.passwordChangeRequired,
  [...view.permissions].sort(),
]);
const readSession = async () => (await call<SessionView>('GET', '/identity/session')).data;
const failedState = (error: unknown): State => error instanceof ApiError && error.status === 401
  ? { kind: 'anonymous' }
  : { kind: 'error', message: errorText(error) };
const retiredError = () => new ApiError(0, 'context_retired', 'Рабочий контекст изменился. Обновите данные.');

/**
 * Loads the session and shows sign-in or the forced
 * password change before rendering the app. `kinds` limits which company
 * kinds this app works with (e.g. the insurer cabinet).
 */
export function SessionGate({
  children,
  title,
  brand,
  kinds,
}: {
  children: ReactNode;
  title: string;
  brand?: string;
  kinds?: Company['kind'][];
}) {
  const [state, setState] = useState<State>({ kind: 'loading' });
  const stateRef = useRef<State>(state);
  const generation = useRef(0);
  const mountSerial = useRef(0);
  const mounted = useRef(false);
  const transitioning = useRef(false);
  const queryClient = useQueryClient();

  const commit = useCallback((next: State) => {
    stateRef.current = next;
    setState(next);
  }, []);

  const retire = useCallback((next: State, synchronous = true) => {
    generation.current += 1;
    transitioning.current = true;
    retireHttpContext();
    // Unmount pages and dialogs before their queries can be recreated.
    if (synchronous) flushSync(() => commit(next));
    else commit(next);
    void queryClient.cancelQueries();
    queryClient.clear();
    mountSerial.current += 1;
    return generation.current;
  }, [commit, queryClient]);

  const admit = useCallback((view: SessionView) => {
    const next: State = {
      kind: 'ready', view, scopeKey: scopeOf(view), grantsKey: grantsOf(view),
      mountKey: mountSerial.current,
    };
    admitHttpContext();
    transitioning.current = false;
    commit(next);
  }, [commit]);

  const refresh = useCallback(async () => {
    if (transitioning.current) throw retiredError();
    const request = ++generation.current;
    supersedeHttpRequests();
    try {
      const view = await readSession();
      if (!mounted.current || request !== generation.current) throw retiredError();
      const previous = stateRef.current;
      if (previous.kind === 'ready' && (previous.scopeKey !== scopeOf(view) || previous.grantsKey !== grantsOf(view))) {
        retire({ kind: 'loading' });
      }
      admit(view);
    } catch (error) {
      if (!mounted.current || request !== generation.current) throw error;
      retire(failedState(error));
      if (error instanceof ApiError && error.status === 401) setCsrf('');
      transitioning.current = false;
      throw error;
    }
  }, [admit, retire]);

  useEffect(() => {
    mounted.current = true;
    const dispose = configureHttp({ onUnauthenticated: () => {
      retire({ kind: 'anonymous' });
      setCsrf('');
      transitioning.current = false;
    } });
    const request = retire({ kind: 'loading' }, false);
    void readSession().then((view) => {
      if (mounted.current && request === generation.current) admit(view);
    }).catch((error) => {
      if (mounted.current && request === generation.current) {
        commit(failedState(error));
        if (error instanceof ApiError && error.status === 401) setCsrf('');
        transitioning.current = false;
      }
    });
    return () => {
      mounted.current = false;
      generation.current += 1;
      retireHttpContext();
      dispose();
    };
  }, [admit, commit, retire]);

  const api = useMemo<SessionApi | null>(() => {
    if (state.kind !== 'ready') return null;
    const view = state.view;
    return {
      view,
      scopeKey: state.scopeKey,
      company: view.accessibleCompanies.find((c) => c.id === view.context.companyId),
      can: (p) => view.permissions.includes(p),
      refresh,
      logout: async () => {
        if (!mounted.current || stateRef.current !== state) throw retiredError();
        const request = retire({ kind: 'loading' });
        try {
          await call('POST', '/identity/session/logout');
        } finally {
          if (mounted.current && request === generation.current) {
            setCsrf('');
            commit({ kind: 'anonymous' });
            transitioning.current = false;
          }
        }
      },
      selectCompany: async (id) => {
        if (!mounted.current || stateRef.current !== state) throw retiredError();
        const request = retire({ kind: 'loading' });
        try {
          await call('PUT', '/identity/session/context', { companyId: id }, { ifMatch: view.context.revision });
          const next = await readSession();
          if (!mounted.current || request !== generation.current) throw retiredError();
          if (next.context.companyId !== id) throw new Error('Не удалось подтвердить выбранную компанию. Повторите вход.');
          admit(next);
        } catch (error) {
          if (mounted.current && request === generation.current) {
            commit(failedState(error));
            transitioning.current = false;
          }
          throw error;
        }
      },
      setBranchScope: async (mode, branchIds) => {
        if (!mounted.current || stateRef.current !== state) throw retiredError();
        const request = retire({ kind: 'loading' });
        try {
          await call('PUT', '/identity/session/branch-scope', { mode, branchIds }, { ifMatch: view.context.revision });
          const next = await readSession();
          if (!mounted.current || request !== generation.current) throw retiredError();
          const actual = next.context.branchScope;
          if (actual.mode !== mode || JSON.stringify([...actual.branchIds].sort()) !== JSON.stringify([...branchIds].sort())) {
            throw new Error('Не удалось подтвердить выбранные филиалы. Повторите вход.');
          }
          admit(next);
        } catch (error) {
          if (mounted.current && request === generation.current) {
            commit(failedState(error));
            transitioning.current = false;
          }
          throw error;
        }
      },
    };
  }, [state, refresh, retire, admit, commit]);

  if (state.kind === 'loading')
    return (
      <Centered>
        <p>Загрузка…</p>
      </Centered>
    );
  if (state.kind === 'error') {
    return (
      <Centered>
        <Card title="Нет связи">
          <Notice kind="danger">{state.message}</Notice>
          <Button onClick={() => void refresh()}>Повторить</Button>
        </Card>
      </Centered>
    );
  }
  if (state.kind === 'anonymous') {
    return <Login title={title} brand={brand} onDone={refresh} />;
  }
  if (state.view.user.passwordChangeRequired) return <ChangePassword forced onDone={refresh} />;
  return (
    <Ctx.Provider value={api}>
      <CompanyGate key={state.mountKey} kinds={kinds}>{children}</CompanyGate>
    </Ctx.Provider>
  );
}

/** Makes sure a suitable company is active; offers a choice if several. */
function CompanyGate({ children, kinds }: { children: ReactNode; kinds: Company['kind'][] | undefined }) {
  const s = useSession();
  const usable = s.view.accessibleCompanies.filter((c) => !kinds || kinds.includes(c.kind));
  const [error, setError] = useState('');
  const tried = useRef(false);
  const only = usable.length === 1 ? usable[0]!.id : undefined;
  const hasCompany = !!s.company;
  useEffect(() => {
    if (!kinds || hasCompany || !only || tried.current) return;
    let live = true;
    queueMicrotask(() => {
      if (!live || tried.current) return;
      tried.current = true;
      void s.selectCompany(only).catch((e) => { if (live) setError(errorText(e)); });
    });
    return () => { live = false; };
  }, [s, kinds, hasCompany, only]);
  if (!kinds) return <>{children}</>; // platform admin app works without a company
  if (s.company && kinds.includes(s.company.kind)) return <>{children}</>;
  if (usable.length === 0) {
    return (
      <Centered>
        <Card title="Нет доступа">
          <p>У вашей учётной записи нет компании для этого кабинета.</p>
          <Button onClick={() => void s.logout()}>Выйти</Button>
        </Card>
      </Centered>
    );
  }
  return (
    <Centered>
      <Card title="Выберите компанию">
        {error && <Notice kind="danger">{error}</Notice>}
        <div className={css.stack}>
          {usable.map((c) => (
            <Button key={c.id} onClick={() => s.selectCompany(c.id).catch((e) => setError(errorText(e)))}>
              {c.name}
            </Button>
          ))}
        </div>
      </Card>
    </Centered>
  );
}

function Centered({ children }: { children: ReactNode }) {
  return <div className={css.centered}>{children}</div>;
}

function Login({ brand, onDone }: { title: string; brand: string | undefined; onDone: () => void | Promise<void> }) {
  const [login, setLogin] = useState('');
  const [password, setPassword] = useState('');
  const [error, setError] = useState('');
  const [busy, setBusy] = useState(false);
  return (
    <Centered>
      <Card title="Вход" subtitle={brand}>
        <form
          className={css.stack}
          onSubmit={async (e) => {
            e.preventDefault();
            setBusy(true);
            setError('');
            try {
              await call('POST', '/identity/session/login', { login, password });
              await onDone();
            } catch (err) {
              setError(
                err instanceof ApiError && err.status === 429
                  ? 'Слишком много попыток. Попробуйте позже.'
                  : errorText(err),
              );
            } finally {
              setBusy(false);
            }
          }}
        >
          {error && <Notice kind="danger">{error}</Notice>}
          <Field label="Логин" value={login} onChange={setLogin} autoComplete="username" required />
          <Field
            label="Пароль"
            type="password"
            value={password}
            onChange={setPassword}
            autoComplete="current-password"
            required
          />
          <Button type="submit" variant="primary" busy={busy}>
            Войти
          </Button>
          <p className="cell-sub">
            Нет доступа? Компанию и её первого администратора подключает администратор платформы JustixAuto.
          </p>
        </form>
      </Card>
    </Centered>
  );
}

export function ChangePassword({ forced, onDone }: { forced?: boolean; onDone: () => void | Promise<void> }) {
  const [current, setCurrent] = useState('');
  const [next, setNext] = useState('');
  const [confirm, setConfirm] = useState('');
  const [error, setError] = useState('');
  const form = (
    <form
      className={css.stack}
      onSubmit={async (e) => {
        e.preventDefault();
        try {
          await call('POST', '/identity/session/password', {
            currentPassword: current,
            newPassword: next,
            newPasswordConfirmation: confirm,
          });
          await onDone();
        } catch (err) {
          setError(errorText(err));
        }
      }}
    >
      {forced && <p>Администратор задал вам временный пароль. Придумайте собственный (не менее 8 символов).</p>}
      {error && <Notice kind="danger">{error}</Notice>}
      <Field
        label="Текущий пароль"
        type="password"
        value={current}
        onChange={setCurrent}
        autoComplete="current-password"
        required
      />
      <Field
        label="Новый пароль"
        type="password"
        value={next}
        onChange={setNext}
        autoComplete="new-password"
        required
      />
      <Field
        label="Повторите новый пароль"
        type="password"
        value={confirm}
        onChange={setConfirm}
        autoComplete="new-password"
        required
      />
      <Button type="submit" variant="primary">
        Сохранить пароль
      </Button>
    </form>
  );
  return forced ? (
    <Centered>
      <Card title="Смена пароля">{form}</Card>
    </Centered>
  ) : (
    form
  );
}
