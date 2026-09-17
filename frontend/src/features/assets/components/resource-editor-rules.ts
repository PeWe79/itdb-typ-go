import { type ResourceConfig, type ResourceField } from '../resource-config';
import { normalizeDateInput, serializeContractRenewals } from '../resource-helpers';

type Row = Record<string, unknown>;
type Lookups = Record<string, Row[]>;

// 发票文件仅允许图片或 PDF，与后端存储校验一致
export const INVOICE_FILE_ACCEPT = '.jpg,.jpeg,.png,.gif,.bmp,.webp,.svg,.avif,.pdf';

const INVOICE_FILE_EXTENSIONS = INVOICE_FILE_ACCEPT.split(',').map(ext => ext.replace(/^\./, ''));

// 发票类型（内置编号 3）仅允许图片或 PDF，与后端存储校验一致，提前拦截避免整轮上传后才失败
export function validateInvoiceFileExtension(values: Row) {
  const file = values.file;
  if (!(file instanceof File)) return '';
  if (Number(values.typeId) !== 3) return '';
  const ext = file.name.includes('.') ? (file.name.split('.').pop() ?? '').toLowerCase() : '';
  if (INVOICE_FILE_EXTENSIONS.includes(ext)) return '';
  return '发票类型的文件仅支持图片或 PDF 格式';
}

export function validateRequiredFields(fields: ResourceField[], values: Row, isEdit = false) {
  const missing = fields.filter(field => {
    if (!field.required) return false;
    const value = values[field.key];
    if (field.type === 'file') return !isEdit && !(value instanceof File);
    if (Array.isArray(value)) return value.length === 0;
    return value === undefined || value === null || String(value).trim() === '';
  });
  return missing.length ? `请完善必填项：${missing.map(field => field.label).join('、')}` : '';
}

// JSON 载荷按原项目 toRequestPayload 逻辑归一类型：选择/数字空值转 0、多选转数字数组，
// 避免空字符串传入后端数值字段导致 JSON 解码失败（“请求体 JSON 格式无效”）
export function buildJsonPayload(
  resource: ResourceConfig,
  values: Row,
  renewals: Row[],
  cleanupFileLinks: number[]
) {
  const payload: Row = {};
  resource.fields.forEach(field => {
    const value = values[field.key];
    if (field.type === 'file') return;
    if (field.type === 'multiselect') {
      payload[field.key] = Array.isArray(value) ? value.map(Number) : [];
      return;
    }
    if (field.type === 'number') {
      const num = Number(value);
      const blank = value === '' || value === null || value === undefined;
      payload[field.key] = blank ? null : Number.isFinite(num) ? num : 0;
      return;
    }
    if (field.type === 'select') {
      if (field.textValue) {
        payload[field.key] = String(value ?? '').trim();
        return;
      }
      const num = Number(value);
      payload[field.key] =
        value === '' || value === null || value === undefined || !Number.isFinite(num) ? 0 : num;
      return;
    }
    if (Array.isArray(value)) {
      payload[field.key] = value;
      return;
    }
    if (field.type === 'date') {
      const display = normalizeDateInput(value);
      payload[field.key] = display;
      return;
    }
    payload[field.key] = value === null || value === undefined ? '' : String(value);
  });
  if (resource.key === 'contracts') payload.renewals = serializeContractRenewals(renewals);
  if (cleanupFileLinks.length) payload.cleanupFileLinks = cleanupFileLinks;
  return payload;
}

function compareDateText(left: string, right: string) {
  if (!left || !right) return 0;
  return left < right ? -1 : left > right ? 1 : 0;
}

export function validateDateRelations(resourceKey: string, values: Row, renewals: Row[]) {
  if (resourceKey !== 'contracts') return '';
  const start = normalizeDateInput(values.startDate);
  const end = normalizeDateInput(values.currentEndDate);
  if (start && end && compareDateText(end, start) < 0) return '结束日期不能早于开始日期';
  for (let index = 0; index < renewals.length; index++) {
    const renewal = renewals[index];
    const rowNumber = index + 1;
    const before = normalizeDateInput(renewal.endDateBefore);
    const after = normalizeDateInput(renewal.endDateAfter);
    const effective = normalizeDateInput(renewal.effectiveDate);
    if (!before && !after && !effective) continue;
    if (before && after && compareDateText(after, before) < 0)
      return `备件第 ${rowNumber} 行到期后日期不能早于到期前日期`;
    if (effective && after && compareDateText(effective, after) > 0)
      return `备件第 ${rowNumber} 行生效日期不能晚于到期后日期`;
  }
  return '';
}

// 机架联动校验（对齐原 itdb 项目）：选择机架后机架位置、大小(U)必填，
// 且位置区间按编号方向展开后不越界、不与同机架其他硬件冲突
export function validateItemRackPlacement(
  resourceKey: string,
  values: Row,
  lookups: Lookups | undefined,
  items: Row[]
) {
  if (resourceKey !== 'items') return '';
  const rackId = Number(values.rackId);
  if (!Number.isFinite(rackId) || rackId <= 0) return '';
  const position = Number(values.rackPosition);
  if (!Number.isFinite(position) || position <= 0) return '请完善必填项：机架位置';
  const units = Number(values.uSize);
  if (!Number.isFinite(units) || units <= 0) return '请完善必填项：大小(U)';
  const rack = (lookups?.racks ?? []).find(row => Number(row.id) === rackId);
  const totalUnits = Number(rack?.usize ?? 0);
  const reverse = Number(rack?.revnums ?? rack?.revNums ?? 0) === 1;
  const spanOf = (start: number, count: number) =>
    Array.from({ length: count }, (_, index) => (reverse ? start + index : start - index));
  const span = spanOf(position, units);
  if (totalUnits > 0 && span.some(unit => unit < 1 || unit > totalUnits)) {
    return '机架位置超出所选机架范围';
  }
  const currentId = Number(values.id ?? 0);
  const used = new Set<number>();
  items.forEach(row => {
    if (Number(row.rackid) !== rackId) return;
    if (currentId > 0 && Number(row.id) === currentId) return;
    const rowPosition = Number(row.rackposition ?? 0);
    const rowUnits = Number(row.usize ?? 0);
    if (
      !Number.isFinite(rowPosition) ||
      rowPosition <= 0 ||
      !Number.isFinite(rowUnits) ||
      rowUnits <= 0
    )
      return;
    spanOf(rowPosition, rowUnits).forEach(unit => used.add(unit));
  });
  const conflicts = [...new Set(span.filter(unit => used.has(unit)))].sort((a, b) => a - b);
  if (conflicts.length > 0) {
    return `机架行 ${conflicts.join('、')} 已被其他硬件占用`;
  }
  return '';
}

// 硬件网络端口：留空合法，填写时必须为 0-65535 的整数
export function validateItemPorts(resourceKey: string, values: Row) {
  if (resourceKey !== 'items') return '';
  const raw = String(values.ports ?? '').trim();
  if (!raw) return '';
  const port = Number(raw);
  if (!Number.isInteger(port) || port < 0 || port > 65535) {
    return '网络端口必须为 0-65535 之间的整数';
  }
  return '';
}

// 软件授权数量：填写时必须为不小于 0 的整数（允许留空表示不限制）
export function validateSoftwareLicenseQty(resourceKey: string, values: Row) {
  if (resourceKey !== 'software') return '';
  const raw = values.licenseQty;
  if (raw === '' || raw === null || raw === undefined) return '';
  const num = Number(raw);
  if (!Number.isFinite(num) || !Number.isInteger(num) || num < 0) {
    return '授权数量必须为正整数';
  }
  return '';
}

type PendingTagChanges = { added: string[]; removed: string[] } | null;

/* 关联类字段在变更说明中统一使用编辑弹窗的页签名称 */
const FIELD_NOTE_NAMES: Record<string, string> = {
  itemLinks: '内部硬件关联',
  invoiceLinks: '单据关联',
  softwareLinks: '软件关联',
  contractLinks: '合同关联',
  fileLinks: '文件关联',
};

export type ResourceChangeExtras = {
  pendingTags?: PendingTagChanges;
  pendingAreas?: boolean;
  pendingEvents?: boolean;
  renewalsChanged?: boolean;
};

function fieldValueChanged(field: ResourceField, before: unknown, after: unknown) {
  if (field.type === 'multiselect') {
    const keyOf = (value: unknown) =>
      Array.isArray(value)
        ? [...value]
            .map(Number)
            .sort((a, b) => a - b)
            .join(',')
        : String(value ?? '');
    return keyOf(before) !== keyOf(after);
  }
  if (Array.isArray(before) || Array.isArray(after)) {
    return JSON.stringify(before ?? []) !== JSON.stringify(after ?? []);
  }
  return String(before ?? '') !== String(after ?? '');
}

/* 各资源编辑窗口的字段显示顺序（与各 DataPane 的渲染顺序保持一致），
   变更项审计按此顺序输出；未定义的资源按 fields 配置顺序 */
const RESOURCE_FIELD_ORDER: Record<string, string[]> = {
  items: [
    'itemTypeId',
    'isPart',
    'rackMountable',
    'manufacturerId',
    'model',
    'uSize',
    'sn',
    'sn2',
    'sn3',
    'comments',
    'label',
    'status',
    'dptId',
    'principal',
    'locationId',
    'locAreaId',
    'rackId',
    'rackPosition',
    'rackPosDepth',
    'function',
    'userId',
    'maintenanceInfo',
    'purchaseDate',
    'warrantyMonths',
    'warrInfo',
    'hd',
    'ram',
    'cpu',
    'cpuNo',
    'coresPerCpu',
    'raid',
    'raidConfig',
    'macs',
    'ipv4',
    'ipv6',
    'remAdmIp',
    'dnsName',
    'panelPort',
    'switchPort',
    'switchId',
    'ports',
    'origin',
    'purchPrice',
    'itemLinks',
    'invoiceLinks',
    'softwareLinks',
    'contractLinks',
    'fileLinks',
    'tags',
  ],
  software: [
    'manufacturerId',
    'title',
    'version',
    'purchaseDate',
    'licenseQty',
    'licenseType',
    'slicenseInfo',
    'info',
    'itemLinks',
    'invoiceLinks',
    'contractLinks',
    'fileLinks',
    'tags',
  ],
  invoices: [
    'vendorId',
    'buyerId',
    'number',
    'date',
    'description',
    'itemLinks',
    'softwareLinks',
    'contractLinks',
    'fileLinks',
  ],
  contracts: [
    'title',
    'number',
    'typeId',
    'subTypeId',
    'contractorId',
    'parentId',
    'totalCost',
    'startDate',
    'currentEndDate',
    'description',
    'comments',
    'renewals',
    'itemLinks',
    'softwareLinks',
    'invoiceLinks',
    'fileLinks',
    'events',
  ],
};

// buildResourceChangeNote 对比编辑前后差异，生成"配置项清单"变更说明尾部；
// 覆盖各资源的全部页签字段、关联页签与 Tags、区域、事件历史、备件等子编辑项；
// 文件字段仅在选择新文件时计入，避免只替换文件内容保存时不写入审计；
// 变更项按编辑窗口从上到下的显示顺序输出
export function buildResourceChangeNote(
  resource: ResourceConfig,
  initial: Row,
  current: Row,
  extras: ResourceChangeExtras = {}
) {
  const changed: Array<{ key: string; label: string }> = resource.fields
    .filter(field => field.type !== 'file' || current[field.key] instanceof File)
    .filter(
      field =>
        field.type === 'file' || fieldValueChanged(field, initial[field.key], current[field.key])
    )
    .filter(field => field.key !== 'renewals')
    .map(field => ({ key: field.key, label: FIELD_NOTE_NAMES[field.key] ?? field.label }));
  if (
    extras.pendingTags &&
    (extras.pendingTags.added.length > 0 || extras.pendingTags.removed.length > 0)
  ) {
    changed.push({ key: 'tags', label: 'Tags' });
  }
  if (extras.pendingAreas) changed.push({ key: 'areas', label: '区域' });
  if (extras.pendingEvents) changed.push({ key: 'events', label: '事件历史' });
  if (extras.renewalsChanged) changed.push({ key: 'renewals', label: '备件' });

  const order = RESOURCE_FIELD_ORDER[resource.key];
  const labels = changed.map(item => item.label);
  if (!order) return labels.join('、');
  const rank = (key: string) => {
    const index = order.indexOf(key);
    return index === -1 ? order.length : index;
  };
  return [...changed]
    .sort((a, b) => rank(a.key) - rank(b.key))
    .map(item => item.label)
    .join('、');
}
