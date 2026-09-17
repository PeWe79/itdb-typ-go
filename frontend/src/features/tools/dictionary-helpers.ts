import type { ExportColumn } from '@/lib/export-data';

export type DictionaryKey =
  'itemtypes' | 'contracttypes' | 'statustypes' | 'filetypes' | 'dpttypes' | 'tags';

type DictionaryRow = Record<string, unknown>;

/* 搜索文本覆盖表格全部显示列：编号、名称，以及各类型特有的显示内容 */
export function dictionaryRowSearchText(
  key: DictionaryKey,
  row: DictionaryRow,
  field: string
): string[] {
  const parts = [row.id, row[field]];
  if (key === 'itemtypes') parts.push(Number(row.hassoftware ?? 0) === 1 ? '是' : '否');
  if (key === 'statustypes') parts.push(row.color);
  if (key === 'tags') parts.push(row.itemCount, row.softwareCount);
  return parts.map(part =>
    String(part ?? '')
      .trim()
      .toLowerCase()
  );
}

/* 导出列与各类型的表格显示列保持一致：
   - 合同类型按筛选数据中最多子类型数量生成“合同子类型N”列，不足的填“-”
   - 标记按最多关联数生成“硬件N/软件N”明细列（名称与原版关联面板同格式），不足的填“-” */
export type TagRelatedLists = {
  items: Array<{ id: number; txt: string }>;
  software: Array<{ id: number; txt: string }>;
};

export function dictionaryExportColumns(
  key: DictionaryKey,
  field: string,
  label: string,
  rows: DictionaryRow[] = [],
  subtypes: DictionaryRow[] = [],
  related: Map<number, TagRelatedLists> = new Map()
): ExportColumn<DictionaryRow>[] {
  const columns: ExportColumn<DictionaryRow>[] = [
    { id: 'id', header: '编号', value: row => Number(row.id ?? 0) },
    { id: 'name', header: label, value: row => String(row[field] ?? '-') },
  ];
  if (key === 'itemtypes') {
    columns.push({
      id: 'hassoftware',
      header: '支持软件',
      value: row => (Number(row.hassoftware ?? 0) === 1 ? '是' : '否'),
    });
  }
  if (key === 'statustypes') {
    columns.push({
      id: 'color',
      header: '状态颜色',
      value: row => String(row.color ?? '-'),
    });
  }
  if (key === 'contracttypes') {
    const namesByType = new Map<number, string[]>();
    for (const subtype of subtypes) {
      const typeId = Number(subtype.contypeid ?? 0);
      const names = namesByType.get(typeId) ?? [];
      names.push(String(subtype.name ?? '-'));
      namesByType.set(typeId, names);
    }
    const exportedIds = new Set(rows.map(row => Number(row.id ?? 0)));
    const maxCount = Math.max(
      0,
      ...[...namesByType.entries()]
        .filter(([typeId]) => exportedIds.has(typeId))
        .map(([, names]) => names.length)
    );
    for (let index = 0; index < maxCount; index += 1) {
      columns.push({
        id: `subtype-${index + 1}`,
        header: `合同子类型${index + 1}`,
        value: row => (namesByType.get(Number(row.id ?? 0)) ?? [])[index] ?? '-',
      });
    }
  }
  if (key === 'tags') {
    columns.push(
      { id: 'itemCount', header: '关联硬件', value: row => Number(row.itemCount ?? 0) },
      { id: 'softwareCount', header: '关联软件', value: row => Number(row.softwareCount ?? 0) }
    );
    const exportedIds = rows.map(row => Number(row.id ?? 0));
    const maxItems = Math.max(0, ...exportedIds.map(id => related.get(id)?.items.length ?? 0));
    const maxSoftware = Math.max(
      0,
      ...exportedIds.map(id => related.get(id)?.software.length ?? 0)
    );
    for (let index = 0; index < maxItems; index += 1) {
      columns.push({
        id: `item-${index + 1}`,
        header: `硬件${index + 1}`,
        value: row => related.get(Number(row.id ?? 0))?.items[index]?.txt ?? '-',
      });
    }
    for (let index = 0; index < maxSoftware; index += 1) {
      columns.push({
        id: `software-${index + 1}`,
        header: `软件${index + 1}`,
        value: row => related.get(Number(row.id ?? 0))?.software[index]?.txt ?? '-',
      });
    }
  }
  return columns;
}
