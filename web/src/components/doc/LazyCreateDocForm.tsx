import { lazy } from "react";

// Lazy: RichTextEditor/yjs are heavy, so this only loads when the "New doc" dialog opens.
export const LazyCreateDocForm = lazy(() =>
  import("@/components/doc/CreateDocForm").then((m) => ({ default: m.CreateDocForm })),
);
