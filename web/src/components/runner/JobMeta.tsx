import { usePersonNamer } from "@/hooks/usePersonNamer";
import { formatAge } from "@/utils/TimeUtility";

interface JobMetaProps {
  // When the job started running, or began to wait.
  at?: string | undefined;
  // A user id or login, with ":mcp" when an agent started it.
  by?: string | undefined;
  // Set where the meta runs on inline after other text, rather than in a flex row with its own gap.
  inline?: boolean;
}

// How long a job has run or waited, and, where its cell is 22rem wide, who started it; a job from before the server sent either shows neither.
export const JobMeta = ({ at, by, inline = false }: JobMetaProps) => {
  const nameOf = usePersonNamer();
  const age = at && formatAge(at);
  const who = by && nameOf(by.replace(/:mcp$/, ""));
  return (
    <>
      {age && inline && " "}
      {age && <span className="shrink-0 whitespace-nowrap text-muted-foreground">· {age === "now" ? "<1m" : age}</span>}
      {who && inline && " "}
      {who && (
        <span className="hidden max-w-[16ch] shrink-0 truncate whitespace-nowrap text-muted-foreground @[22rem]/doing:inline" title={`Started by ${who}`}>
          · {who}
        </span>
      )}
    </>
  );
};
