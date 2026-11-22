import { read, utils, type WorkBook } from 'xlsx';

const maxRows = 50;
const maxColumns = 24;

type SheetGrid = {
  name: string;
  columns: string[];
  rows: string[][];
  limit: { rows: number; columns: number } | null;
};

export type SheetRequest = { url: string } | { sheet: string };
export type SheetResponse = { sheets: string[]; grid: SheetGrid } | { failed: true };

let workbook: WorkBook | null = null;
let sheets: string[] = [];

function readGrid(workbook: WorkBook, name: string): SheetGrid {
  const sheet = workbook.Sheets[name];
  const reference = sheet['!fullref'] ?? sheet['!ref'];
  if (!reference) return { name, columns: [], rows: [], limit: null };

  const bounds = utils.decode_range(reference);
  const rows = Math.min(maxRows, bounds.e.r + 1);
  const columns = Math.min(maxColumns, bounds.e.c + 1);
  const range = { s: { r: 0, c: 0 }, e: { r: rows - 1, c: columns - 1 } };

  return {
    name,
    columns: Array.from({ length: columns }, (_, index) => utils.encode_col(index)),
    rows: utils.sheet_to_json<string[]>(sheet, { header: 1, range, raw: false, defval: '', blankrows: true }),
    limit: bounds.e.r >= maxRows || bounds.e.c >= maxColumns ? { rows, columns } : null,
  };
}

self.onmessage = async ({ data }: MessageEvent<SheetRequest>) => {
  try {
    let selected;
    if ('url' in data) {
      const response = await fetch(data.url);
      if (!response.ok) throw new Error(`Preview HTTP ${response.status}`);
      workbook = read(await response.arrayBuffer(), { dense: true, sheetRows: maxRows + 1, cellHTML: false });
      sheets = workbook.SheetNames.filter((_, index) => !workbook?.Workbook?.Sheets?.[index]?.Hidden);
      selected = sheets[0];
    } else {
      selected = data.sheet;
    }
    if (!workbook || !sheets.includes(selected)) throw new Error('No worksheet to preview');
    self.postMessage({ sheets, grid: readGrid(workbook, selected) } satisfies SheetResponse);
  } catch {
    self.postMessage({ failed: true } satisfies SheetResponse);
  }
};
