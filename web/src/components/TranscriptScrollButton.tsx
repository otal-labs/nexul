import { MessageScrollerButton } from "@/components/ui/message-scroller";

// Scroll to end, in the scroller's bottom-right corner, where it leaves the line it would cover readable.
export const TranscriptScrollButton = () => (
  <MessageScrollerButton className="inset-s-auto end-3 size-8 translate-x-0 bg-card shadow-card data-[direction=end]:bottom-3 rtl:translate-x-0" />
);
