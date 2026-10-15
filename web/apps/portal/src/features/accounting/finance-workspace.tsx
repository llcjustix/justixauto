import type { ReactNode } from 'react';
import { FinanceWorkspaceContext } from './finance-workspace-context';

export function FinanceWorkspace({ children }: { children: ReactNode }) {
  return <FinanceWorkspaceContext.Provider value>{children}</FinanceWorkspaceContext.Provider>;
}
