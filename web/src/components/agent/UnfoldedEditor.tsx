import type { ReactNode } from "react";

interface UnfoldedEditorProps {
  label: string;
  // A small muted line over the fields saying whose settings these are.
  title?: string;
  children: ReactNode;
}

// What a row unfolds into: one panel on `surface-2` indented to the auto plays' column, its fields as label rows,
// inside a container so they stack in a narrow pane. Its foot (EditorFooter) sticks to the screen's bottom while the panel is taller, on the panel's opaque fill under a hairline.
export const UnfoldedEditor = ({ label, title, children }: UnfoldedEditorProps) => (
  <div className="settle-in border-b border-border py-3 pr-3 pl-4 sm:pl-12">
    <div role="group" aria-label={label} className="@container rounded-lg bg-surface-2 px-4 ring-1 ring-border sm:px-6">
      {title && <p className="pt-3 text-xs text-muted-foreground">{title}</p>}
      <div>{children}</div>
    </div>
  </div>
);
