import type { XlsxTemplateColumnSpec } from '@/lib/export-data';
import type { DictionaryKey } from './dictionary-helpers';

type Row = Record<string, unknown>;

/* 模板列的输入辅助规格：下拉选项或选中提示（写入 Excel 数据验证） */
export function dictionaryImportTemplateSpecs(
  key: DictionaryKey
): Record<number, XlsxTemplateColumnSpec> {
  if (key === 'itemtypes') return { 1: { dropdownValues: ['是', '否'] } };
  if (key === 'contracttypes') return { 1: { note: '多个合同子类型请使用顿号（、）分隔，可为空' } };
  if (key === 'statustypes')
    return { 1: { note: '十六进制颜色（如 #2f7fba），可为空，为空时使用默认颜色' } };
  return {};
}

/* 模板列与导出结构对齐：名称列 + 各类型特有列；
   标记不考虑“硬件N/软件N”明细列；合同子类型单列、多条用“、”分隔 */
export function dictionaryImportHeaders(key: DictionaryKey, label: string): string[] {
  if (key === 'itemtypes') return [label, '支持软件'];
  if (key === 'statustypes') return [label, '状态颜色'];
  if (key === 'contracttypes') return [label, '合同子类型'];
  return [label];
}

/* 解析 .xlsx：用浏览器原生 DecompressionStream 解压 zip 条目，无需第三方依赖。
   仅读取第一个工作表与共享字符串表，返回按行分组的字符串矩阵 */
export async function parseXlsxMatrix(file: File): Promise<string[][]> {
  const buffer = new Uint8Array(await file.arrayBuffer());
  const entries = await readZipEntries(buffer);
  const sheetEntry = entries.find(entry => /^xl\/worksheets\/sheet\d+\.xml$/i.test(entry.name));
  if (!sheetEntry) throw new Error('表格文件缺少工作表，请使用下载的 Excel 模板填写');
  const sharedEntry = entries.find(entry => entry.name.toLowerCase() === 'xl/sharedstrings.xml');
  const shared = sharedEntry ? parseSharedStrings(await inflateEntry(sharedEntry)) : [];
  const sheetXml = await inflateEntry(sheetEntry);
  return parseSheetRows(sheetXml, shared);
}

type ZipEntry = { name: string; method: number; data: Uint8Array };

async function readZipEntries(buffer: Uint8Array): Promise<ZipEntry[]> {
  const view = new DataView(buffer.buffer, buffer.byteOffset, buffer.byteLength);
  let eocd = -1;
  for (
    let offset = buffer.length - 22;
    offset >= 0 && offset > buffer.length - 65558;
    offset -= 1
  ) {
    if (view.getUint32(offset, true) === 0x06054b50) {
      eocd = offset;
      break;
    }
  }
  if (eocd < 0) throw new Error('无法识别的表格文件，请使用下载的 Excel 模板');
  const entryCount = view.getUint16(eocd + 10, true);
  let offset = view.getUint32(eocd + 16, true);
  const entries: ZipEntry[] = [];
  const decoder = new TextDecoder();
  for (let index = 0; index < entryCount; index += 1) {
    if (view.getUint32(offset, true) !== 0x02014b50) break;
    const method = view.getUint16(offset + 10, true);
    const compressedSize = view.getUint32(offset + 20, true);
    const nameLength = view.getUint16(offset + 28, true);
    const extraLength = view.getUint16(offset + 30, true);
    const commentLength = view.getUint16(offset + 32, true);
    const localOffset = view.getUint32(offset + 42, true);
    const name = decoder.decode(buffer.subarray(offset + 46, offset + 46 + nameLength));
    if (view.getUint32(localOffset, true) !== 0x04034b50) {
      offset += 46 + nameLength + extraLength + commentLength;
      continue;
    }
    const localNameLength = view.getUint16(localOffset + 26, true);
    const localExtraLength = view.getUint16(localOffset + 28, true);
    const dataStart = localOffset + 30 + localNameLength + localExtraLength;
    entries.push({
      name,
      method,
      data: buffer.subarray(dataStart, dataStart + compressedSize),
    });
    offset += 46 + nameLength + extraLength + commentLength;
  }
  return entries;
}

async function inflateEntry(entry: ZipEntry): Promise<string> {
  if (entry.method === 0) return new TextDecoder().decode(entry.data);
  const bytes = new Uint8Array(entry.data.length);
  bytes.set(entry.data);
  const stream = new Blob([bytes]).stream().pipeThrough(new DecompressionStream('deflate-raw'));
  return new TextDecoder().decode(await new Response(stream).arrayBuffer());
}

function parseSharedStrings(xml: string): string[] {
  const doc = new DOMParser().parseFromString(xml, 'application/xml');
  return Array.from(doc.getElementsByTagName('si')).map(item =>
    Array.from(item.getElementsByTagName('t'))
      .map(node => node.textContent ?? '')
      .join('')
  );
}

function parseSheetRows(xml: string, shared: string[]): string[][] {
  const doc = new DOMParser().parseFromString(xml, 'application/xml');
  const matrix: string[][] = [];
  for (const rowNode of Array.from(doc.getElementsByTagName('row'))) {
    const cells: string[] = [];
    for (const cellNode of Array.from(rowNode.getElementsByTagName('c'))) {
      const ref = cellNode.getAttribute('r') ?? '';
      const columnIndex = columnLettersToIndex(ref);
      const type = cellNode.getAttribute('t') ?? '';
      let text = '';
      if (type === 's') {
        const value = cellNode.getElementsByTagName('v')[0]?.textContent ?? '';
        text = shared[Number(value)] ?? '';
      } else if (type === 'inlineStr') {
        text = Array.from(cellNode.getElementsByTagName('t'))
          .map(node => node.textContent ?? '')
          .join('');
      } else {
        text = cellNode.getElementsByTagName('v')[0]?.textContent ?? '';
      }
      while (cells.length < columnIndex) cells.push('');
      cells[columnIndex] = text.trim();
    }
    matrix.push(cells);
  }
  while (matrix.length > 0 && matrix.at(-1)!.every(cell => cell === '')) matrix.pop();
  return matrix;
}

function columnLettersToIndex(ref: string) {
  let index = 0;
  for (const char of ref) {
    if (char < 'A' || char > 'Z') break;
    index = index * 26 + (char.charCodeAt(0) - 64);
  }
  return Math.max(0, index - 1);
}

/* 一行模板数据 → 创建请求；合同类型的子类型单独返回，待类型创建拿到编号后再提交 */
export type DictionaryImportRequest = {
  path: string;
  body: Row;
  subtypes?: string[];
};

export function dictionaryImportPayloads(
  key: DictionaryKey,
  cells: string[]
): DictionaryImportRequest {
  const name = (cells[0] ?? '').trim();
  if (key === 'itemtypes') {
    const support = (cells[1] ?? '').trim();
    return { path: 'itemtypes', body: { typedesc: name, hassoftware: support === '是' ? 1 : 0 } };
  }
  if (key === 'statustypes') {
    const color = (cells[1] ?? '').trim();
    return { path: 'statustypes', body: { statusdesc: name, color: color || '#2f7fba' } };
  }
  if (key === 'contracttypes') {
    return {
      path: 'contracttypes',
      body: { name },
      subtypes: (cells[1] ?? '')
        .split(/[、，,;；\n]+/)
        .map(item => item.trim())
        .filter(Boolean),
    };
  }
  if (key === 'filetypes') return { path: 'filetypes', body: { typedesc: name } };
  if (key === 'dpttypes') return { path: 'dpttypes', body: { dptname: name } };
  return { path: 'tags', body: { name } };
}
