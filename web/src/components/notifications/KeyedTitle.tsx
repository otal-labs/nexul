import { Fragment } from "react";

// A ticket key inside a notice's title ("Fix with AI failed on ATL-12") is technical data, so it is set in mono.
const KEY = /\b([A-Z][A-Z0-9]{1,9}-\d+)\b/;

export const KeyedTitle = ({ title }: { title: string }) => (
  <>
    {title.split(KEY).map((part, i) => (
      <Fragment key={i}>
        {i % 2 === 1 && <span className="font-mono">{part}</span>}
        {i % 2 === 0 && part}
      </Fragment>
    ))}
  </>
);
