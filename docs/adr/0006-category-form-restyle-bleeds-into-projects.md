# Restyling the shared category-create form is allowed to bleed into Projects

ADR 0003 frees `web/src/components/board/` and Board-specific pages/settings
from `design-language.md`, but keeps every other domain on the unchanged
spec. The Board design pass (`.scratch/board-design-pass/`) needs Board's
"+ New swimlane" dialog restyled to match its new black-and-white, no-pill-
fills language — but that dialog reuses `AddCategoryForm.tsx`, the same
component the Projects page uses to create categories. Restyling it in
place, rather than forking a Board-only copy, means the change reaches
Projects too, ahead of ADR 0003's promised app-wide unification pass.

Chose to restyle the shared component in place rather than fork it. Forking
means maintaining two near-identical forms for one field set. The owner
also intends to redesign Projects next, so having its category-create form
already carry Board's direction is a head start, not drift — this is a
deliberate widening of ADR 0003's boundary for this one component, not a
reopening of the boundary generally. Every other Projects surface still
follows `design-language.md` unchanged.
