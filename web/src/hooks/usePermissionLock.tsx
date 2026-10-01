import { createContext, use } from "react";

// Locks every level control inside it, so a read-only list doesn't hand its flag down each row.
export const PermissionLockContext = createContext(false);

export const usePermissionLock = (): boolean => use(PermissionLockContext);
