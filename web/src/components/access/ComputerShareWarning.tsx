import { TriangleAlert } from "lucide-react";

// What sharing a computer hands over, said before anyone is given it: everything runs as the owner's own account there.
export const ComputerShareWarning = () => (
  <p className="flex items-start gap-2 rounded-md border border-input px-3 py-2.5 text-xs text-muted-foreground">
    <TriangleAlert className="mt-px size-3.5 shrink-0 text-warning" aria-hidden />
    <span className="text-pretty">
      <span className="font-medium text-foreground">Agents and commands run as your own user on this computer.</span> Whoever you let run
      them can read what your account can there, your SSH keys, git credentials and logins included. Share it only with people you trust
      with that.
    </span>
  </p>
);
