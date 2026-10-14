import { act } from "@testing-library/react";

const OWNER = { role_name: "Owner", roles: [{ id: "role-owner", name: "Owner", is_owner: true }], permissions: [] };

// A member who only reads: a role holding the listed permissions, with Show locked actions on or off.
export const memberWho = (permissions: string[], showLocked: boolean) => ({
  role_name: "Member",
  roles: [{ id: "role-member", name: "Member", is_owner: false }],
  permissions,
  show_locked_actions: showLocked,
});

const isViewerRead = (url: string) => /^\/api\/workspaces\/[^/]+\/me$/.test(url) || url === "/api/permissions/explain";

const viewerRead = (url: string, me: object) => Promise.resolve({ data: url === "/api/permissions/explain" ? { effective: [] } : me });

interface InstanceHolding {
  instance_permissions: string[];
  instance_owner: boolean;
}

// The auth /me says who the viewer is and what they hold on the instance; a test's own answer wins field by field.
const accountOf = async (rest: (url: string) => unknown, instance: InstanceHolding) => {
  const own = await Promise.resolve()
    .then(() => rest("/api/auth/me"))
    .catch(() => undefined);
  const user = { id: "u-viewer", login: "alice", name: "Alice", email: "alice@example.com", avatar_url: "" };
  return { data: { user, needs_owner_wizard: false, needs_first_login_wizard: false, ...instance, ...(own && !Array.isArray(own) ? own : {}) } };
};

// Answers the viewer's permission reads as `me` with no rule on any item, and the auth /me as what they hold on the instance; every other read goes to `rest`.
export const asViewer =
  (me: object, rest: (url: string) => unknown = () => [], instance: InstanceHolding = { instance_permissions: [], instance_owner: false }) =>
  async (url: string) => {
    if (url === "/api/auth/me") return accountOf(rest, instance);
    return isViewerRead(url) ? viewerRead(url, me) : { data: await rest(url) };
  };

// Answers the viewer's permission reads as an Owner with no rule on any item; every other read goes to `rest`.
export const asOwner = (rest: (url: string) => unknown = () => []) => asViewer(OWNER, rest, { instance_permissions: [], instance_owner: true });

// The same for a test that already serves its own reads as promises.
export const withOwner =
  (get: (url: string, ...rest: never[]) => unknown) =>
  async (url: string, ...rest: unknown[]) =>
    isViewerRead(url) ? viewerRead(url, OWNER) : get(url, ...(rest as never[]));

// A viewer holding `permissions` and nothing else; `showLocked` is their role's Show locked actions switch.
export const asReader = ({ showLocked = false, permissions = [] as string[] }: { showLocked?: boolean; permissions?: string[] } = {}, rest: (url: string) => unknown = () => []) =>
  asViewer(memberWho(permissions, showLocked), rest);

// Lets the viewer's permission reads land, for asserting that something stays absent for them.
export const settle = () =>
  act(async () => {
    await new Promise((resolve) => setTimeout(resolve, 30));
  });
