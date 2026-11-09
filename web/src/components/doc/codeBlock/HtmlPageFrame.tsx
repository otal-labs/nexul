import { useLayoutEffect, useRef, useState } from "react";

import { HTML_PAGE_MIN_HEIGHT, readHtmlPageMessage } from "@nexul/client-core/htmlPage";
import { useHtmlPageDocument } from "@/hooks/HtmlPageHooks";

// Opens a link only while the reader is using the frame: a page can post on load, but can't fake the reader's click.
const readerIsInFrame = (frame: HTMLIFrameElement) =>
  document.activeElement === frame && navigator.userActivation?.isActive !== false;

// Never allow-same-origin: the page runs on an opaque origin, out of reach of the reader's session and the doc around it.
export const HtmlPageFrame = ({ code }: { code: string }) => {
  const { data } = useHtmlPageDocument(code);
  const frameRef = useRef<HTMLIFrameElement>(null);
  const [height, setHeight] = useState(HTML_PAGE_MIN_HEIGHT);

  // At commit, because a fast page posts its first height before passive effects run.
  useLayoutEffect(() => {
    const onMessage = (event: MessageEvent) => {
      const frame = frameRef.current;
      if (!frame || event.source !== frame.contentWindow) return;
      const message = readHtmlPageMessage(event.data);
      if (message?.type === "height") setHeight(message.height);
      if (message?.type === "link" && readerIsInFrame(frame)) window.open(message.url, "_blank", "noopener,noreferrer");
    };
    window.addEventListener("message", onMessage);
    return () => window.removeEventListener("message", onMessage);
  }, []);

  if (!data) return null;
  return (
    <iframe
      ref={frameRef}
      srcDoc={data.html}
      title="HTML page"
      sandbox="allow-scripts"
      allow="fullscreen"
      loading="lazy"
      className="block w-full border-0"
      style={{ height, colorScheme: data.appearance }}
      contentEditable={false}
    />
  );
};
