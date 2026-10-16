# 10: The stack page, redesigned to hold what this effort adds

Type: prototype
Status: resolved
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

## Answer

Locked in `practices/design-language.md` ("Stack detail page"); the header and logs toolbar
changes shipped with this ticket, the rest is placed for the build tickets.

- **One header band.** The facts row is gone: slug, runner, strategy, repository (or network)
  and a run stack's image move into the title's mono meta line. The live status well becomes one
  strip: Health, Last deploy, Services (count over "N up"), Hostnames. The services list left
  the strip, since the Services card under it lists them.
- **Health is the first column** and the only incident summary on the page: "No issues", or
  one line per problem in its status hue (outage red, error rate amber, recovered green). No
  incident list or banner between the strip and the cards (a prototype with one re-created the
  two-headers problem).
- **Overview order:** Services, Requests, Deploy. Ticket 14 puts an open outage as one line
  under the service's name in its Services row. Ticket 18 adds the Requests card (hour totals,
  5xx share, per-minute bars in a well with the 5xx part amber, the rule in the footer), drawn
  as plain SVG bars so no chart dependency is added.
- **Alerts section** after Logs in the section nav: a Crash tickets card (ticket 16) and an
  Error rate card (ticket 18), each one switch row; the error-rate row is greyed with its reason
  when the stack has no hostname or the token lacks analytics.
- **Watching** (ticket 15) is an outline eye button in the header's actions, left of the menu.
- **Logs toolbar:** Copy and Download became icon buttons with tooltips, so the toolbar and its
  status marker fit one row; the service tab row no longer shows a stray vertical scrollbar.
