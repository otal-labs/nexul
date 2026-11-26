import { ArrowRightIcon } from "lucide-react";
import { Link } from "react-router";

// What a section leaves out, as a way to the full list rather than a muted aside.
export const HomeMoreLink = ({ to, count, label }: { to: string; count: number; label: string }) => (
  <Link
    to={to}
    className="group mt-1 inline-flex items-center gap-1.5 rounded-md py-1 text-xs text-muted-foreground transition-colors duration-150 ease-standard hover:text-foreground"
  >
    <span className="font-mono tabular-nums">{count}</span> {label}
    <ArrowRightIcon className="size-3 transition-transform duration-150 ease-standard group-hover:translate-x-0.5" aria-hidden />
  </Link>
);
