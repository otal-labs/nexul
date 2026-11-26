import { Fragment } from "react";

// A ticket key inside a notice's title ("Fix with AI failed on ATL-12") is technical data, so it is set in mono.
const KEY = /\b([A-Z][A-Z0-9]{1,9}-\d+)\b/;

// A play's notice repeats its wording on every row ("Fix with AI finished on"), so quiet mutes the words and leaves the key in ink.
export const KeyedTitle = ({ title, quiet = false }: { title: string; quiet?: boolean }) => (
  <>
    {title.split(KEY).map((part, i) => (
      <Fragment key={i}>
        {i % 2 === 1 && <span className="font-mono text-foreground">{part}</span>}
        {i % 2 === 0 && quiet && <span className="text-muted-foreground">{part}</span>}
        {i % 2 === 0 && !quiet && part}
      </Fragment>
    ))}
  </>
);
