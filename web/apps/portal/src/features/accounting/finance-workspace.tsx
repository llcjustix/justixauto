import { createContext, useContext } from 'react';
import type { ReactNode } from 'react';

/** Financial decisions are available only from the Accounting cabinet. */
const FinanceWorkspaceContext = createContext(false);

export function FinanceWorkspace({ children }: { children: ReactNode }) {
  return <FinanceWorkspaceContext.Provider value>{children}</FinanceWorkspaceContext.Provider>;
}

export function useFinanceWorkspace() {
  return useContext(FinanceWorkspaceContext);
}
