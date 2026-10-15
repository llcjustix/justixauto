import { createContext } from 'react';
import type { ComponentProps } from 'react';
import type { ActionButton } from '@justixauto/kit';

export type OrderAction = ComponentProps<typeof ActionButton>;
export const OrderActionContext = createContext<((action: OrderAction) => void) | null>(null);
