import { createContext, useContext } from 'react';
import type { ComponentProps } from 'react';
import { ActionButton, Button } from '@justixauto/kit';

export type OrderAction = ComponentProps<typeof ActionButton>;
export const OrderActionContext = createContext<((action: OrderAction) => void) | null>(null);

/** Order-owned forms replace the active surface; standalone invoice callers keep their dialog. */
export function OrderActionButton(props: OrderAction) {
  const open = useContext(OrderActionContext);
  if (!open) return <ActionButton {...props} />;
  return <Button variant={props.variant ?? 'secondary'} disabled={props.disabled}
    onClick={() => open(props)}>{props.label}</Button>;
}
