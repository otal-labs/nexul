import { cn } from "@/lib/utils";

const Tile = ({ src, className, caption }: { src: string; className: string; caption: string }) => (
  <figure className="flex flex-col items-center gap-1.5">
    <span className={cn("grid place-items-center overflow-hidden bg-surface-2 ring-1 ring-border", className)}>
      <img src={src} alt="" className="size-full object-contain" />
    </span>
    <figcaption className="text-xs text-muted-foreground">{caption}</figcaption>
  </figure>
);

// The logo at the two sizes the instance draws its mark: the sidebar's and the sign-in corner's.
export const BrandingLogoPreview = ({ src }: { src: string }) => (
  <div className="flex items-end gap-6">
    <Tile src={src} className="size-8 rounded-lg" caption="Sidebar" />
    <Tile src={src} className="size-5 rounded-sm" caption="Sign-in" />
  </div>
);
