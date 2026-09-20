# Board drag-and-drop moves to `@dnd-kit`, off native HTML5 DnD

Board's column-to-column ticket movement used the native HTML5 Drag & Drop
API, which has no touch support at all. Replaced with `@dnd-kit` (pointer +
touch + keyboard sensors) so moving a ticket works on any input device, not
just a mouse. This also fixes an unrelated flicker bug that came from
`dragenter`/`dragleave` bubbling through child elements under native DnD,
since `@dnd-kit` owns its own hover state instead of relying on that
bubbling.
