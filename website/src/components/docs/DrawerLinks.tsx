import { siteLinks } from '../SiteHeader';

// The header drops the site links on narrow docs pages, so they sit at the foot of the sidebar there instead.
export function DrawerLinks() {
	return (
		<nav aria-label="Site" className="flex flex-col gap-0.5 pt-2 text-sm lg:hidden">
			{siteLinks.map((link) => (
				<a key={link.href} href={link.href} className="rounded-md px-2 py-2 text-fd-muted-foreground hover:bg-fd-accent hover:text-fd-foreground">
					{link.text}
				</a>
			))}
			<a href="/#install" className="mt-2 rounded-md bg-fd-primary px-3 py-2 text-center font-medium text-fd-primary-foreground md:hidden">
				Get started
			</a>
		</nav>
	);
}
