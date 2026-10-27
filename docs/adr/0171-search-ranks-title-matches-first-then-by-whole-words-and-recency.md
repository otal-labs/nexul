# Search ranks title matches first, then by whole words and recency

Ticket and doc search returned one list in bm25 order, so a doc that said a word often in its body ranked above the
ticket named for it, a prefix typed into the palette (`log`) ranked `Login times out` level with `Log viewer`, and a
year-old hit sat level with one changed this morning. The palette showed that list flat, without saying where a hit's
words were.

Decision: `mentions.Search`, which backs the command palette, the `@` picker and `mention_search`, sorts tickets and
docs into two sections and ranks within each.

- **Section.** A hit whose title holds every typed word, as a word or the start of one, is a title match; any other
  hit is a text match, found in a ticket's body or note or a doc's body. Title matches come first. An exact ticket key
  still comes before both, and people keep their places around them.
- **Score.** Within a section, score = −bm25 × 2 when every word is a whole word of the title (or of the snippet, for
  a text match) × (1 + 7 / (7 + days since the last change)). Recent activity at most doubles a score and halves its
  boost after a week, so a far better match still wins over a fresher one.
- **Where the words are.** A text match carries a snippet of its matched words, a doc's the heading above them and a
  ticket's the note they are in, so the palette opens the doc at that heading and the ticket at that note.
- **Workspace.** With a workspace, tickets and docs come from its projects only, in SQL, so another workspace's hits
  never take a place in the limit.

The browser and MCP read the same use-case, so an agent sees the order a person sees. Stacks, automations and
memories stay matched by name in the palette from the lists their pages already read; they have no index to rank.

Rejected: ranking in SQL with bm25 column weights, which still mixes body and title hits in one list and cannot say
which section a hit belongs to; a recency boost without a ceiling, which buries an exact title under anything edited
today.

Decided 2026-10-27.
