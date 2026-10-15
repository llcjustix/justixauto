import { createContext } from 'react';
import type { ComponentProps } from 'react';
import type { ActionButton } from '@justixauto/kit';

export type RetailAction = ComponentProps<typeof ActionButton>;
export const RetailActionContext = createContext<((action: RetailAction) => void) | null>(null);
