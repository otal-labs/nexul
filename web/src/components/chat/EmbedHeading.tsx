import { httpUrl, type Embed } from "@/models/Embed";

const EmbedAuthor = ({ author }: { author: NonNullable<Embed["author"]> }) => {
  const icon = httpUrl(author.icon_url);
  const url = httpUrl(author.url);
  return (
    <p className="flex min-w-0 items-center gap-1.5 text-xs font-medium text-foreground/85">
      {icon && <img src={icon} alt="" loading="lazy" referrerPolicy="no-referrer" className="size-4 shrink-0 rounded-full" />}
      {url && (
        <a href={url} target="_blank" rel="noopener noreferrer" className="truncate hover:underline">
          {author.name}
        </a>
      )}
      {!url && <span className="truncate">{author.name}</span>}
    </p>
  );
};

// The author line over the title, which links out when the sender gave it a URL.
export const EmbedHeading = ({ embed: { author, title, url } }: { embed: Embed }) => {
  const href = httpUrl(url);
  return (
    <>
      {author && <EmbedAuthor author={author} />}
      {title && href && (
        <a href={href} target="_blank" rel="noopener noreferrer" className="block text-sm font-semibold wrap-anywhere text-foreground hover:underline">
          {title}
        </a>
      )}
      {title && !href && <p className="text-sm font-semibold wrap-anywhere text-foreground">{title}</p>}
    </>
  );
};
