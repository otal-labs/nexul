import type { ReactNode } from "react";
import { renderSVG } from "uqr";

import { cn } from "@/lib/utils";

// Generated locally from our own string, so a data URL carries it without touching innerHTML.
const qrDataUrl = (link: string) =>
  `data:image/svg+xml,${encodeURIComponent(renderSVG(link, { border: 0, whiteColor: "#ffffff", blackColor: "#000000" }))}`;

interface QrFrameProps {
  link?: string;
  dimmed: boolean;
  action?: ReactNode;
  children: ReactNode;
}

// Without a link the frame shows a faint stand-in so the card keeps its size until a code exists.
export const QrFrame = ({ link, dimmed, action, children }: QrFrameProps) => (
  <div className="@container">
    {/* Two cards share the row from 1024px, so a card can be 240px wide: the text drops under the QR until the card has room. */}
    <div className="flex flex-col items-start gap-5 @sm:flex-row">
      <div className="relative shrink-0 rounded-md bg-white p-2.5">
        <img
          key={link}
          src={qrDataUrl(link ?? "nexul")}
          alt={link ? "Sign-in code for the Nexul app" : ""}
          className={cn(
            "size-32 animate-in fade-in-0 animation-duration-150 ease-out transition-opacity duration-200",
            dimmed && "opacity-10",
          )}
        />
        {action}
      </div>
      <div className="min-w-0 space-y-3 text-sm">{children}</div>
    </div>
  </div>
);
