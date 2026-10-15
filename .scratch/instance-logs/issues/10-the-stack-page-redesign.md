# 10: The stack page, redesigned to hold what this effort adds

Type: prototype
Status: open
Blocked by: 07, 09

## Question

The stack page (`web/src/pages/StackPage.tsx` and `web/src/components/stack/`) has to carry new
things: an open outage line per service, an error-rate line and its rule, the per-person
Watching toggle, the crash ticket switch, and a small request chart. Today it already reads
crowded:

- The facts header (last deploy, services up, hostnames) and the row under it (machine,
  strategy, repository) read as two headers.
- The Logs section's toolbar (service tabs, All/Errors, Pause, Copy, Download, Live) and its
  long wrapped lines push everything else down.

Where does each piece go, and what moves?

- Run through auto design mode: score the top references, pick, build the prototype on the
  real page with real data at 768, 1024 and 1440px, and report once with the runner-up to swap
  to.
- Keep within `practices/design-language.md` and the settings-shell sub-nav the page already
  uses. Status hues stay status hues: an outage is red, an error rate over its rule is amber,
  and recovered is green.
- The build tickets for tickets 07 and 09 place their pieces where this lands, so it goes
  first among the UI work.
