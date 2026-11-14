# Branding carries custom CSS that loads nothing

Amends ADR 0187, which turned down letting the Owner set any token because it would be a second design language with
no contrast floor. A host matching a client's own tool needs more than colours (a font stack, square corners, a
backdrop image), and an agent asked to "make it look like our site" can only do that with CSS. So branding takes the
Owner's own CSS, added after Nexul's styles on every page, sign-in included, behind the same key and permission as the
rest of branding, and `instance_get` with `styling_guide` gives agents the tokens and the selectors Nexul keeps for it
(`StableSelectors`, held to the web source by a test). The trade-off is the contrast floor: an imported theme is still
corrected, custom CSS is not, and keeping it readable is the Owner's job; the design language stays the default every
instance without branding shows. The hard rule is that the CSS can load nothing, since a page's CSS that reaches
another address can send what the page shows there (attribute selectors that each fetch a different URL). The server
refuses rather than strips: `url()` other than `data:` or `#id`, `@import`, `image-set()`, `image()`, `src()`, any
backslash (an escape can spell each of those) and `<` (the CSS sits in a `<style>` element of the served page). The
cost is that an image must be inline and the CSS stays under 20,000 characters. Why not a sanitiser in the browser:
the server is the only writer, refusing tells the author which rule to change, and the page head carries the CSS
before the app loads. The phone app keeps its own look, as ADR 0187 says.

Decided: 2026-11-14.
