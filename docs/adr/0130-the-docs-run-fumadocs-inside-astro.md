# The docs run Fumadocs inside Astro

nexul.io's docs moved off Starlight to Fumadocs UI, rendered as React islands
inside the existing Astro site, rather than to a Next.js app, which is where
Fumadocs is most used. Starlight's chrome could only be restyled one override
at a time, and the docs read as a separate product from the home page.
Staying on Astro keeps the home, roadmap, changelog and installer scripts as
static pages with no React, and keeps the content files and URLs unchanged.
The cost is running Fumadocs on its less travelled path: page tree, search
index and markdown copies are wired by hand in `website/src/lib/` instead of
coming from `fumadocs-mdx`. If that path stops being maintained, the content
moves to a Next.js static export as is.
