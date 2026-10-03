import { createContext, useContext, useRef, useState } from 'react';
import type { ComponentProps } from 'react';
import { ActionButton, ApiError, Button, FormDialog } from '@justixauto/kit';

export type RetailAction = ComponentProps<typeof ActionButton>;
export const RetailActionContext = createContext<((action: RetailAction) => void) | null>(null);

export function RetailActionButton(props: RetailAction) {
  const open = useContext(RetailActionContext);
  if (!open) return <ActionButton {...props} />;
  return <Button variant={props.variant ?? 'secondary'} disabled={props.disabled}
    onClick={() => open(props)}>{props.label}</Button>;
}

/** The deal owns this form instead of mounting it over another dialog. */
export function RetailActionForm({ action, context, onClose, onDone, backLabel = 'Назад к сделке' }: {
  action: RetailAction; context: string; onClose: () => void; onDone: () => Promise<void>; backLabel?: string;
}) {
  const sending = useRef(false);
  const completed = useRef(false);
  const surface = useRef<HTMLDivElement>(null);
  const [busy, setBusy] = useState(false);
  // FormDialog also uploads files before invoking onSubmit. Its submit button
  // exposes aria-busy throughout that phase, which must guard modal dismissal too.
  const close = () => {
    if (completed.current || (!sending.current && !surface.current?.querySelector('[aria-busy="true"]'))) onClose();
  };
  return <div className="sale-dialog" ref={surface}><FormDialog title={action.title ?? action.label} fields={action.fields ?? []}
    submitLabel={action.submitLabel ?? action.label} size={action.size}
    intro={<><p>{context}</p><Button variant="back" disabled={busy} onClick={close}>{backLabel}</Button>{action.intro}</>}
    onClose={close}
    onSubmit={async values => {
      if (sending.current) return;
      sending.current = true;
      setBusy(true);
      try {
        await action.onSubmit(values);
        await onDone();
        completed.current = true;
      } catch (error) {
        if (error instanceof ApiError && (error.status === 409 || error.status === 412)) {
          throw new ApiError(error.status, error.code,
            `${error.message} Данные изменились. Вернитесь к сделке и обновите данные перед повтором.`, error.fields);
        }
        throw error;
      } finally { sending.current = false; setBusy(false); }
    }} /></div>;
}
