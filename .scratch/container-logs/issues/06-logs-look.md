# 06 — Where logs open and how they look

**Type:** prototype
**Status:** resolved
**Blocked by:** 01

## Question

Where does someone reach a container's logs: a row action in the stack page's Services card, a click on a service node on the topology canvas (today it only opens the stack page), a logs tab on the stack page, or a page per container? How the view looks: reuse the deploy log panel (follow the tail, copy, download) or something richer (a stream filter, search, pause, wrap). Mono Console, built at 768px first. Settled with `design-mode`.

## Answer

Decided 2026-09-30 by the owner, from real references.

- **Where:**
  - A **Logs** entry in the stack page's sub-navigation, beside Overview,
    Exposures and Deploy history, with one tab per service.
  - A **Logs** link on each row of the Services card opens it on that
    service.
  - Clicking a service node on the topology canvas opens that service's logs
    instead of the stack overview.
- **Body:** one continuous terminal-style block per service, not a list of
  expandable event rows.
  - A timestamp column sits on the left and the raw line on the right, with
    long lines wrapping under the text rather than under the timestamp.
  - The row rhythm is dense, and a copy button sits in the panel's corner.
- **Toolbar**, above the block:
  - service tabs;
  - an **All / Errors** switch (errors means stderr);
  - **Pause**, **Copy** and **Download**;
  - a small **Live** marker while the view follows new lines.
- **Left out:** search, time ranges, "show more", level badges, and row
  expand. Nothing is stored, and container output has no levels.
- **Color:** stderr lines get a thin status-colored gutter mark, the only
  color in the view (Mono Console).
- **Scrolling:** scrolling up pauses following; a "Jump to live" control
  resumes it.

