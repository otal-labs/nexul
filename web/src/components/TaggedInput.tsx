import type { ReactNode } from "react";

interface TaggedInputProps {
  // The tag drawn inside the field's right edge; the input keeps its width and only its text makes room while a tag shows.
  tag: ReactNode;
  children: ReactNode;
}

export const TaggedInput = ({ tag, children }: TaggedInputProps) => (
  <div className="relative has-[[data-slot=no-fill-badge]]:[&_[data-slot=input]]:pr-24">
    {children}
    <span className="pointer-events-none absolute inset-y-0 right-2 flex items-center">{tag}</span>
  </div>
);
