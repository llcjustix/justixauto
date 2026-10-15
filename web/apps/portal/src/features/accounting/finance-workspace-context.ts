import { createContext, useContext } from 'react';

/** Financial decisions are available only from the Accounting cabinet. */
export const FinanceWorkspaceContext = createContext(false);

export function useFinanceWorkspace() {
  return useContext(FinanceWorkspaceContext);
}
