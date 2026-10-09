export interface Role {
  id: string;
  workspace_id: string;
  name: string;
  permissions: string[];
  is_owner_role: boolean;
  created_at: string;
  updated_at: string;
}

// A duplicate's name: "Admin (copy)", then "Admin (copy 2)" and on while those are taken.
export const copyName = (name: string, taken: readonly string[]): string => {
  const base = `${name} (copy)`;
  if (!taken.includes(base)) return base;
  let n = 2;
  while (taken.includes(`${name} (copy ${n})`)) n++;
  return `${name} (copy ${n})`;
};
