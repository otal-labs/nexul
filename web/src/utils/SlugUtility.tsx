// Mirrors deploy.Slug (internal/deploy/model.go): lowercase, non-alphanumeric runs collapsed to one dash,
// trimmed. The server derives the stack's own slug this way; the wizard only needs the same shape client-side
// to default a run stack's docker network before the stack (and its real slug) exist.
export const slugify = (text: string): string =>
  text
    .toLowerCase()
    .replace(/[^a-z0-9]+/g, "-")
    .replace(/^-+|-+$/g, "");
