import { GithubMark } from "@/components/ProviderMarks";
import type { LoginMatch } from "@/models/User";

interface AllowlistSuggestionRowProps {
  match: LoginMatch;
  onPick: (match: LoginMatch) => void;
}

const AllowlistSuggestionRow = ({ match, onPick }: AllowlistSuggestionRowProps) => (
  <li>
    <button
      type="button"
      role="option"
      aria-selected={false}
      className="flex w-full min-w-0 items-center gap-2 px-3 py-2 text-left text-sm hover:bg-accent"
      onClick={() => onPick(match)}
    >
      <img src={match.avatar_url} alt="" className="size-5 shrink-0 rounded-full" />
      <span className="min-w-0 flex-1 truncate font-mono">{match.login}</span>
      <GithubMark className="size-4 shrink-0 text-muted-foreground" />
    </button>
  </li>
);

interface AllowlistSuggestionsListProps {
  show: boolean;
  matches: LoginMatch[];
  onPick: (match: LoginMatch) => void;
}

export const AllowlistSuggestionsList = ({ show, matches, onPick }: AllowlistSuggestionsListProps) => {
  if (!show) return null;

  return (
    <ul
      role="listbox"
      aria-label="GitHub username suggestions"
      className="divide-y divide-border overflow-hidden rounded-md border bg-card shadow-card"
    >
      {matches.map((match) => (
        <AllowlistSuggestionRow key={match.login} match={match} onPick={onPick} />
      ))}
    </ul>
  );
};
