# A run stores the memories it read as the steps land

A run is told to read some memories and finds the rest itself by their when-to-use line (ADR 0111). Its steps showed
those reads, but nothing on the screen did: a run listed only what it was told to read, and a memory showed only its
Always included switch. A memory every run opened looked unused, so nobody could tell a when-to-use line that works from
one no run ever opens.

Decision: when a finished `memory_get` step lands on a run, the runner stores one row for that run and memory, with the
title the memory had then and the time, in `trail_memory_reads`. The tool counts by its last name, so every harness's
prefix matches, and only a memory of the run's own project is stored, so a run never names a title its reader may not
see. A second read changes nothing. Deleting a memory keeps its rows, so a run still says what it read; deleting a run
takes them with it. One trail's view lists the rows it was not told to read as "read by itself", and a memory's own read
carries the newest row's time as its last read, in the same statement as the memory. The runs stored before this were
filled from their steps once, in the migration.

Recomputing both from the steps on every read was rejected. A trail's steps are one JSON column capped at 300 entries,
so a memory read early in a long run falls off it, and a memory's last read would mean scanning every run's steps of its
project on each Memories page. The cost is a write per memory a run reads, and a title that can only be as current as the
memory: a memory deleted before the migration ran leaves no row, since there is no title to keep.

Decided 2026-11-18.
