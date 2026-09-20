import type { SaveDocFormData } from "@/models/Doc";

// Split out of RichtextUtility.tsx so reading this constant doesn't drag the tiptap/yjs stack into the bundle.
export const emptyDocJson = `{"type":"doc","content":[{"type":"paragraph"}]}`;

export const emptyDocForm = (): SaveDocFormData => ({ project_id: "", title: "", body: emptyDocJson });
