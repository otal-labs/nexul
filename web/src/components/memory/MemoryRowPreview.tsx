interface MemoryRowPreviewProps {
  alwaysIncluded: boolean;
  snippet: string;
}

// A row's one line under its title: sent in every turn leads it in words, so the state reads without a control and every row keeps two lines.
export const MemoryRowPreview = ({ alwaysIncluded, snippet }: MemoryRowPreviewProps) => (
  <>
    {alwaysIncluded && <span className="shrink-0 text-foreground">Always included</span>}
    {alwaysIncluded && snippet !== "" && <span aria-hidden>·</span>}
    {snippet !== "" && (
      <span dir="auto" className="min-w-0 truncate">
        {snippet}
      </span>
    )}
  </>
);
