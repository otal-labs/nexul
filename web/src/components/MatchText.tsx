interface MatchTextProps {
  text: string;
  // The search being typed; its first match in the text is underlined so a filtered row shows why it is there.
  match?: string | undefined;
}

export const MatchText = ({ text, match = "" }: MatchTextProps) => {
  const needle = match.trim().toLowerCase();
  const at = needle ? text.toLowerCase().indexOf(needle) : -1;
  if (at < 0) return <>{text}</>;
  return (
    <>
      {text.slice(0, at)}
      <span className="underline decoration-foreground/50 underline-offset-2">{text.slice(at, at + needle.length)}</span>
      {text.slice(at + needle.length)}
    </>
  );
};
