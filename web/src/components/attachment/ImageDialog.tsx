import { LinkIcon } from "lucide-react";
import { toast } from "sonner";

import { Button } from "@/components/ui/button";
import { Dialog, DialogContent, DialogHeader, DialogTitle, DialogTrigger } from "@/components/ui/dialog";

interface ImageDialogProps {
  // What renders: an object URL for an attachment, the raw URL otherwise.
  src: string;
  alt: string;
  // The image's own address, the one the link copies.
  link: string;
  className?: string;
  onCloseAutoFocus?: (event: Event) => void;
}

const copyLink = async (link: string) => {
  try {
    await navigator.clipboard.writeText(new URL(link, window.location.origin).href);
    toast.success("Link copied");
  } catch {
    toast.error("Couldn't copy the link");
  }
};

export const ImageDialog = ({ src, alt, link, className, onCloseAutoFocus }: ImageDialogProps) => (
  <Dialog>
    <DialogTrigger asChild>
      <button type="button" aria-label={alt ? `Open ${alt}` : "Open image"} className="block max-w-full cursor-zoom-in">
        <img src={src} alt={alt} className={className} />
      </button>
    </DialogTrigger>
    <DialogContent aria-describedby={undefined} onCloseAutoFocus={onCloseAutoFocus} className="gap-3 p-4 sm:max-w-[min(64rem,calc(100%-2rem))]">
      <DialogHeader className="flex-row items-center gap-3 pr-8">
        <DialogTitle className="min-w-0 flex-1 truncate text-sm font-medium">{alt || "Image"}</DialogTitle>
        <Button type="button" variant="outline" size="sm" onClick={() => void copyLink(link)}>
          <LinkIcon aria-hidden />
          Copy link
        </Button>
      </DialogHeader>
      <img src={src} alt={alt} className="mx-auto max-h-[80vh] max-w-full rounded-md object-contain" />
    </DialogContent>
  </Dialog>
);
