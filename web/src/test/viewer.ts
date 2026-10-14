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

// Answers the viewer's permission reads as `me` with no rule on any item; every other read goes to `rest`.
export const asViewer =
  (me: object, rest: (url: string) => unknown = () => []) =>
  async (url: string) =>
    isViewerRead(url) ? viewerRead(url, me) : { data: await rest(url) };

// Answers the viewer's permission reads as an Owner with no rule on any item; every other read goes to `rest`.
export const asOwner = (rest: (url: string) => unknown = () => []) => asViewer(OWNER, rest);

// The same for a test that already serves its own reads as promises.
export const withOwner =
  (get: (url: string, ...rest: never[]) => unknown) =>
  async (url: string, ...rest: unknown[]) =>
    isViewerRead(url) ? viewerRead(url, OWNER) : get(url, ...(rest as never[]));
