# An imported VS Code theme is the one exception to one design language

Amended by ADR 0188: branding also takes custom CSS, with no contrast floor, that can load nothing.

AGENTS.md rule 11 says one design language holds across `web/`, `website/` and `native/`, extended and never re-themed. A
host who resells an instance (ADR 0184) wants it to look like the client's own tool, and the client's colours already exist
as a VS Code color theme. The instance Owner may therefore import such a theme file for the whole instance, in the web app
only, and it re-themes the surface, text, accent and status roles of its own appearance. It is the only exception: the
design language's structure stays (panels, the type pair, the radius scale, the one accent held to its roles), the theme
supplies only its canvas, text, accent and error colours, every surface step between them is derived from the same ladder
the shipped palettes use, and any colour that would leave text under the contrast `themeContrast.test.ts` holds every
palette to is moved until it passes. The product name, logo and a lone accent (the `brand` token's value) are branding and
no exception to the rule. Why not leave themes to each person's palette: a client's staff do not choose their palette, and
the host wants one look; why not let the Owner set any token: that is a second design language with no contrast floor.
The phone app and the site stay as built, since a branded phone app is a build per client (map out of scope).

Decided: 2026-11-11. Map: `.scratch/enterprise-edition/map.md`.
