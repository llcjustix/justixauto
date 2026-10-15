import { useContext } from 'react';
import { ActionButton, Button } from '@justixauto/kit';
import { OrderActionContext } from './order-action-context';
import type { OrderAction } from './order-action-context';
export type { OrderAction } from './order-action-context';

/** Order-owned forms replace the active surface; standalone invoice callers keep their dialog. */
export function OrderActionButton(props: OrderAction) {
  const open = useContext(OrderActionContext);
  if (!open) return <ActionButton {...props} />;
  return (
    <Button variant={props.variant ?? 'secondary'} disabled={props.disabled} onClick={() => open(props)}>
      {props.label}
    </Button>
  );
}
