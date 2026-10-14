const OWNER = { role_name: "Owner", roles: [{ id: "role-owner", name: "Owner", is_owner: true }], permissions: [] };

const isViewerRead = (url: string) => /^\/api\/workspaces\/[^/]+\/me$/.test(url) || url === "/api/permissions/explain";

const viewerRead = (url: string) => Promise.resolve({ data: url === "/api/permissions/explain" ? { effective: [] } : OWNER });

// Answers the viewer's permission reads as an Owner with no rule on any item; every other read goes to `rest`.
export const asOwner =
  (rest: (url: string) => unknown = () => []) =>
  async (url: string) =>
    isViewerRead(url) ? viewerRead(url) : { data: await rest(url) };

// The same for a test that already serves its own reads as promises.
export const withOwner =
  <T,>(get: (url: string) => Promise<T>) =>
  (url: string) =>
    isViewerRead(url) ? viewerRead(url) : get(url);
