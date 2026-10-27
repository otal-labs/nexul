export interface CsvTable {
  rows: string[][];
  // True when rows or columns past the limits were left out.
  truncated: boolean;
}

// RFC 4180: a quoted field may hold the delimiter, line breaks and doubled quotes. Stops after maxRows, so a huge file stays cheap.
export const parseCsv = (input: string, delimiter: string, maxRows: number, maxColumns: number): CsvTable => {
  const text = input.replace(/^\uFEFF/, "");
  const rows: string[][] = [];
  let row: string[] = [];
  let field = "";
  let quoted = false;
  let truncated = false;
  const endField = () => {
    truncated ||= row.length >= maxColumns;
    if (row.length < maxColumns) row.push(field);
    field = "";
  };
  for (let i = 0; i < text.length; i++) {
    const c = text.charAt(i);
    if (quoted && c === '"' && text.charAt(i + 1) === '"') {
      field += '"';
      i++;
      continue;
    }
    if (quoted && c === '"') {
      quoted = false;
      continue;
    }
    if (quoted) {
      field += c;
      continue;
    }
    if (c === '"' && field === "") {
      quoted = true;
      continue;
    }
    if (c === delimiter) {
      endField();
      continue;
    }
    if (c !== "\n" && c !== "\r") {
      field += c;
      continue;
    }
    if (c === "\r" && text.charAt(i + 1) === "\n") i++;
    endField();
    rows.push(row);
    row = [];
    if (rows.length > maxRows) return { rows: rows.slice(0, maxRows), truncated: true };
  }
  if (field !== "" || row.length > 0) {
    endField();
    rows.push(row);
  }
  return { rows: rows.slice(0, maxRows), truncated: truncated || rows.length > maxRows };
};

export const csvDelimiter = (name: string): string => (name.toLowerCase().endsWith(".tsv") ? "\t" : ",");
