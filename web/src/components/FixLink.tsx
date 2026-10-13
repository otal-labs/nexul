import { Link } from "react-router";

interface FixLinkProps {
  to: string;
  label: string;
}

// The way forward at the end of a sentence that says what is missing.
export const FixLink = ({ to, label }: FixLinkProps) => (
  <Link to={to} className="text-foreground underline underline-offset-2">
    {label}
  </Link>
);
