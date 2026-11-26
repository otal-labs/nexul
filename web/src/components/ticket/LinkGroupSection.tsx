import { Children, useState, type ReactNode } from "react";

interface LinkGroupSectionProps {
  title: string;
  children: ReactNode;
}

const CAP = 5;

// One titled group of linked tickets inside the Links or Bugs rail section; past five rows the rest wait behind Show N more.
export const LinkGroupSection = ({ title, children }: LinkGroupSectionProps) => {
  const [all, setAll] = useState(false);
  const rows = Children.toArray(children);
  // A single leftover row is shown rather than hidden behind a button of the same height.
  const hidden = !all && rows.length > CAP + 1 ? rows.length - CAP : 0;

  return (
    <div className="space-y-0.5">
      <h3 className="px-2 text-xs text-muted-foreground">{title}</h3>
      <ul className="flex flex-col">{hidden > 0 ? rows.slice(0, CAP) : rows}</ul>
      {hidden > 0 && (
        <button
          type="button"
          onClick={() => setAll(true)}
          className="w-full rounded-md px-2 py-1.5 text-left text-xs text-muted-foreground transition-colors duration-150 ease-standard hover:bg-accent hover:text-foreground"
        >
          {`Show ${hidden} more`}
        </button>
      )}
    </div>
  );
};
