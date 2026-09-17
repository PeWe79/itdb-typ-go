import type { ResourceField } from './resource-config';

type Row = Record<string, unknown>;
type Lookups = Record<string, Row[]>;

export function editorTabs(resourceKey: string, fields: ResourceField[]) {
  if (resourceKey === 'items') {
    return [
      { value: 'main', label: '硬件数据' },
      { value: 'itemLinks', label: '内部硬件关联' },
      { value: 'invoiceLinks', label: '单据关联' },
      { value: 'maintenanceInfo', label: '维护日志' },
      { value: 'softwareLinks', label: '软件关联' },
      { value: 'contractLinks', label: '合同关联' },
      { value: 'fileLinks', label: '文件关联' },
    ];
  }
  if (resourceKey === 'files') {
    const labels: Record<string, string> = {
      itemLinks: '硬件关联',
      softwareLinks: '软件关联',
      contractLinks: '合同关联',
    };
    return [
      { value: 'main', label: '文件数据' },
      ...['itemLinks', 'softwareLinks', 'contractLinks']
        .filter(key => fields.some(field => field.key === key))
        .map(key => ({ value: key, label: labels[key] })),
    ];
  }
  if (resourceKey === 'contracts') {
    return [
      { value: 'main', label: '合同数据' },
      { value: 'events', label: '事件历史' },
      { value: 'itemLinks', label: '硬件关联' },
      { value: 'softwareLinks', label: '软件关联' },
      { value: 'invoiceLinks', label: '单据关联' },
      { value: 'fileLinks', label: '文件关联' },
    ];
  }
  const relationLabels: Record<string, string> = {
    itemLinks: resourceKey === 'software' || resourceKey === 'invoices' ? '硬件关联' : '关联硬件',
    softwareLinks: '软件关联',
    invoiceLinks: '单据关联',
    contractLinks: '合同关联',
    fileLinks: '文件关联',
  };
  return [
    {
      value: 'main',
      label:
        resourceKey === 'software'
          ? '软件数据'
          : resourceKey === 'invoices'
            ? '单据数据'
            : resourceKey === 'agents'
              ? '代理数据'
              : resourceKey === 'locations'
                ? '地点数据'
                : '基本信息',
    },
    ...fields
      .filter(field => field.key.endsWith('Links'))
      .map(field => ({ value: field.key, label: relationLabels[field.key] ?? field.label })),
  ];
}

export function editorTabMatches(resourceKey: string, field: ResourceField, activeTab: string) {
  if (resourceKey === 'items') {
    const tabFields = new Set([
      'itemLinks',
      'invoiceLinks',
      'maintenanceInfo',
      'softwareLinks',
      'contractLinks',
      'fileLinks',
    ]);
    return activeTab === 'main' ? !tabFields.has(field.key) : field.key === activeTab;
  }
  if (resourceKey === 'files') {
    const fileRelationFields = new Set([
      'itemLinks',
      'softwareLinks',
      'contractLinks',
      'invoiceLinks',
      'fileLinks',
    ]);
    return activeTab === 'main' ? !fileRelationFields.has(field.key) : field.key === activeTab;
  }
  if (resourceKey === 'contracts') {
    const relationFields = new Set(['itemLinks', 'softwareLinks', 'invoiceLinks', 'fileLinks']);
    return activeTab === 'main' ? !relationFields.has(field.key) : field.key === activeTab;
  }
  return activeTab === 'main' ? !field.key.endsWith('Links') : field.key === activeTab;
}

export function parseContractRenewals(value: unknown): Row[] {
  if (Array.isArray(value)) return value as Row[];
  const text = String(value ?? '').trim();
  if (!text) return [];
  return text
    .split('|')
    .filter(Boolean)
    .map(part => {
      const [
        endDateBefore = '',
        endDateAfter = '',
        effectiveDate = '',
        notes = '',
        enteredDate = '',
        enteredBy = '',
      ] = part.split('#');
      return {
        endDateBefore: endDateBefore.trim(),
        endDateAfter: endDateAfter.trim(),
        effectiveDate: effectiveDate.trim(),
        notes: notes.trim(),
        enteredDate: enteredDate.trim(),
        enteredBy: enteredBy.trim(),
      };
    });
}

export function serializeContractRenewals(rows: Row[]) {
  return rows
    .map(row =>
      [
        row.endDateBefore,
        row.endDateAfter,
        row.effectiveDate,
        row.notes,
        row.enteredDate,
        row.enteredBy,
      ]
        .map(value =>
          String(value ?? '')
            .replace(/[|#]/g, ' ')
            .trim()
        )
        .join('#')
    )
    .join('|');
}

export function editorSection(resourceKey: string, fieldKey: string, index: number) {
  if (index === 0) return '基础信息';
  if (resourceKey !== 'items') return fieldKey.endsWith('Links') ? '关联信息' : undefined;
  const labels: Record<string, string> = {
    status: '使用信息',
    purchaseDate: '账目与维保',
    hd: '硬件配置',
    macs: '网络配置',
    itemLinks: '关联信息',
  };
  return labels[fieldKey];
}

export function hardwareMainGroups(fields: ResourceField[], section: string) {
  const groups = [
    {
      value: 'basic',
      title: '基础信息配置',
      keys: [
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
      ],
    },
    {
      value: 'usage',
      title: '使用信息配置',
      keys: [
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
      ],
    },
    {
      value: 'warranty',
      title: '维保信息配置',
      keys: ['purchaseDate', 'warrantyMonths', 'warrInfo'],
    },
    {
      value: 'hardware',
      title: '硬件信息配置',
      keys: ['hd', 'ram', 'cpu', 'cpuNo', 'coresPerCpu', 'raid', 'raidConfig'],
    },
    {
      value: 'network',
      title: '网络信息配置',
      keys: [
        'macs',
        'ipv4',
        'ipv6',
        'remAdmIp',
        'dnsName',
        'panelPort',
        'switchPort',
        'switchId',
        'ports',
      ],
    },
    {
      value: 'accounting',
      title: '账目信息配置',
      keys: ['origin', 'purchPrice'],
    },
  ];
  return groups
    .filter(group => group.value === section)
    .map(group => ({
      title: group.title,
      fields: group.keys
        .map(key => fields.find(field => field.key === key))
        .filter((field): field is ResourceField => Boolean(field)),
    }));
}

export function softwareMainGroup(fields: ResourceField[], section: string) {
  if (section !== 'attribute') return [];
  const keys = [
    'manufacturerId',
    'title',
    'version',
    'purchaseDate',
    'licenseQty',
    'licenseType',
    'slicenseInfo',
    'info',
  ];
  return [
    {
      title: '软件属性配置',
      fields: keys
        .map(key => fields.find(field => field.key === key))
        .filter((field): field is ResourceField => Boolean(field)),
    },
  ];
}

export function invoiceMainGroup(fields: ResourceField[], section: string) {
  if (section !== 'attribute') return [];
  const keys = ['vendorId', 'buyerId', 'number', 'date', 'description'];
  return [
    {
      title: '单据属性配置',
      fields: keys
        .map(key => fields.find(field => field.key === key))
        .filter((field): field is ResourceField => Boolean(field)),
    },
  ];
}

export function contractMainGroup(fields: ResourceField[], section: string) {
  if (section !== 'attribute') return [];
  const keys = [
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
  ];
  return [
    {
      title: '合同属性配置',
      fields: keys
        .map(key => fields.find(field => field.key === key))
        .filter((field): field is ResourceField => Boolean(field)),
    },
  ];
}

export function agentMainGroup(fields: ResourceField[], section: string) {
  if (section !== 'attribute') return [];
  const keys = ['title', 'types', 'contactInfo'];
  return [
    {
      title: '代理属性配置',
      fields: keys
        .map(key => fields.find(field => field.key === key))
        .filter((field): field is ResourceField => Boolean(field)),
    },
  ];
}

export type AgentTypeBadge = {
  key: number;
  label: string;
  tone: 'vendor' | 'software' | 'hardware' | 'buyer' | 'contractor';
};

export function parseAgentTypeBadges(value: unknown): AgentTypeBadge[] {
  const mask = Number(value);
  if (!Number.isFinite(mask) || mask <= 0) return [];
  const badges: AgentTypeBadge[] = [];
  if ((mask & 4) === 4) badges.push({ key: 4, label: '供应商', tone: 'vendor' });
  if ((mask & 2) === 2) badges.push({ key: 2, label: '软件厂商', tone: 'software' });
  if ((mask & 8) === 8) badges.push({ key: 8, label: '硬件厂商', tone: 'hardware' });
  if ((mask & 1) === 1) badges.push({ key: 1, label: '采购方', tone: 'buyer' });
  if ((mask & 16) === 16) badges.push({ key: 16, label: '承包方', tone: 'contractor' });
  return badges;
}

// 代理“合同”列（contacts）的字段标签，与编辑器“合同”页签表头一致
const agentContactFieldLabels: Record<string, string> = {
  name: '姓名',
  phones: '电话',
  email: '邮箱',
  role: '角色',
  comments: '备注',
};

export function getAgentContactEntries(raw: unknown) {
  const text = String(raw ?? '').trim();
  if (!text || text === '####') return [];
  return text
    .split('|')
    .map(row => row.trim())
    .filter(Boolean)
    .map(row => {
      const [name = '', phones = '', email = '', role = '', comments = ''] = row.split('#');
      return [
        { key: 'name', value: name.trim() },
        { key: 'phones', value: phones.trim() },
        { key: 'email', value: email.trim() },
        { key: 'role', value: role.trim() },
        { key: 'comments', value: comments.trim() },
      ]
        .filter(field => field.value)
        .map(field => ({ ...field, label: agentContactFieldLabels[field.key] ?? field.key }));
    })
    .filter(fields => fields.length > 0);
}

/* 硬件类型的“支持软件”关闭时，硬件编辑器的关联软件页签给出锁定说明 */
export function itemTypeBlocksSoftware(lookups: Lookups | undefined, typeId: number): string {
  if (!typeId) return '';
  const type = (lookups?.itemtypes ?? []).find(item => Number(item.id) === typeId);
  if (!type || Number(type.hassoftware ?? 1) !== 0) return '';
  const name = String(type.typedesc ?? '').trim();
  return `该硬件类型${name ? `「${name}」` : ''}不支持软件，无法关联软件。`;
}

export function lookupOptions(lookups: Lookups | undefined, key?: string) {
  const rows = key ? (lookups?.[key] ?? []) : [];
  return rows.map(row => {
    const id = Number(row.id);
    const label = (() => {
      switch (key) {
        case 'locations': {
          const name = String(row.name ?? '').trim();
          const floor = String(row.floor ?? '').trim();
          return formatLookupLabel(floor ? `${name}, 楼层:${floor}` : name);
        }
        case 'locareas':
          return formatLookupLabel(row.areaname);
        case 'racks':
          return formatLookupLabel(row.label, row.usize ? `${row.usize}U` : undefined, row.model);
        case 'agents':
          return formatLookupLabel(row.title);
        case 'users':
          return formatLookupLabel(row.username);
        case 'items_ref':
          return formatLookupLabel(row.label, row.model);
        case 'software_ref':
          return formatLookupLabel(row.stitle, row.sversion);
        case 'contracts_ref':
          return formatLookupLabel(row.number, row.title);
        case 'files_ref':
          return formatLookupLabel(row.title, row.fname);
        case 'invoices_ref':
          return formatLookupLabel(row.number);
        default:
          return formatLookupLabel(
            row.typedesc ?? row.name ?? row.statusdesc ?? row.dptname ?? row.title ?? row.label
          );
      }
    })();
    return { value: id, label: label || String(id) };
  });
}

export function formatLookupLabel(...values: unknown[]) {
  return values
    .map(value => String(value ?? '').trim())
    .filter(Boolean)
    .join(' · ');
}

/* 代理类型位掩码：不同资源的代理下拉按各自所需的类型过滤（与原项目一致） */
const agentTypeFilters: Array<{ resource: string; key: string; mask: number }> = [
  { resource: 'items', key: 'manufacturerId', mask: 8 },
  { resource: 'software', key: 'manufacturerId', mask: 2 },
  { resource: 'invoices', key: 'vendorId', mask: 4 },
  { resource: 'invoices', key: 'buyerId', mask: 1 },
  { resource: 'contracts', key: 'contractorId', mask: 16 },
];

export function resolveFieldOptions(
  field: ResourceField,
  lookups: Lookups | undefined,
  values: Row,
  resourceKey?: string
) {
  if (resourceKey) {
    const entry = agentTypeFilters.find(
      item => item.resource === resourceKey && item.key === field.key
    );
    if (entry) {
      const rows = lookups?.agents ?? [];
      const filtered = rows.filter(item => (Number(item.type ?? 0) & entry.mask) === entry.mask);
      const selectedId = Number(values[field.key] ?? 0);
      const selectedRow =
        selectedId > 0 ? rows.find(item => Number(item.id ?? 0) === selectedId) : undefined;
      const options =
        selectedRow && !filtered.some(item => Number(item.id ?? 0) === selectedId)
          ? [selectedRow, ...filtered]
          : filtered;
      return lookupOptions({ agents: options }, 'agents');
    }
  }
  if (field.key === 'subTypeId') {
    const typeId = Number(values.typeId ?? 0);
    if (!typeId) return [];
    return lookupOptions(
      {
        contractsubtypes: (lookups?.contractsubtypes ?? []).filter(
          subtype => Number(subtype.contypeid) === typeId
        ),
      },
      'contractsubtypes'
    );
  }
  if (resourceKey === 'contracts' && field.key === 'parentId') {
    const currentId = Number(values.id ?? 0);
    return (lookups?.contracts_ref ?? [])
      .filter(item => {
        const id = Number(item.id ?? 0);
        return id > 0 && id !== currentId;
      })
      .map(row => {
        const id = Number(row.id);
        const prefix = String(row.id ?? '').trim();
        const title = String(row.title ?? '').trim();
        return {
          value: id,
          label: [prefix ? `${prefix}:` : '', title].filter(Boolean).join(' ') || String(id),
        };
      });
  }
  if (field.key === 'locAreaId') {
    const locationId = Number(values.locationId ?? 0);
    if (!locationId) return [];
    return lookupOptions(
      {
        locareas: (lookups?.locareas ?? []).filter(area => Number(area.locationid) === locationId),
      },
      'locareas'
    );
  }
  if (field.key === 'rackId') {
    const locationId = Number(values.locationId ?? 0);
    const areaId = Number(values.locAreaId ?? 0);
    if (!locationId) return [];
    return lookupOptions(
      {
        racks: (lookups?.racks ?? []).filter(
          rack =>
            Number(rack.locationid) === locationId && (!areaId || Number(rack.locareaid) === areaId)
        ),
      },
      'racks'
    );
  }
  if (field.key === 'rackPosition') {
    const rack = (lookups?.racks ?? []).find(item => Number(item.id) === Number(values.rackId));
    const size = Math.max(0, Number(rack?.usize ?? 0));
    return Array.from({ length: size }, (_, index) => ({
      value: index + 1,
      label: `U${index + 1}`,
    }));
  }
  if (field.key === 'switchId') {
    const currentId = Number(values.id ?? 0);
    return (lookups?.items_ref ?? [])
      .filter(item => String(item.itemtype ?? '').trim() === '交换机')
      .filter(item => currentId <= 0 || Number(item.id) !== currentId)
      .map(item => {
        const text = [
          String(item.manufacturer ?? '').trim(),
          String(item.label ?? '').trim(),
          String(item.model ?? '').trim(),
        ]
          .filter(Boolean)
          .join(' ');
        return { value: Number(item.id), label: text ? `${item.id}: ${text}` : `${item.id}` };
      });
  }
  if (resourceKey === 'items' && field.key === 'origin') {
    const options = (lookups?.agents ?? [])
      .filter(agent => (Number(agent.type ?? 0) & 4) === 4)
      .map(agent => {
        const title = String(agent.title ?? '').trim();
        return { value: title, label: formatLookupLabel(title) };
      });
    const current = String(values.origin ?? '').trim();
    if (current && !options.some(option => option.value === current)) {
      return [{ value: current, label: formatLookupLabel(current) }, ...options];
    }
    return options;
  }
  return field.options ?? lookupOptions(lookups, field.optionsKey);
}

export function parseAgentContacts(value: unknown): Row[] {
  if (Array.isArray(value)) return value as Row[];
  return String(value ?? '')
    .split('|')
    .filter(Boolean)
    .map(part => {
      const [name = '', phones = '', email = '', role = '', comments = ''] = part.split('#');
      return { name, phones, email, role, comments };
    });
}

export function parseAgentUrls(value: unknown): Row[] {
  if (Array.isArray(value)) return value as Row[];
  return String(value ?? '')
    .split('|')
    .filter(Boolean)
    .map(part => {
      const [description = '', url = ''] = part.split('#');
      return { description, url };
    });
}

// 取代理 URLs 中描述包含“服务”或 service（不区分大小写）的链接，供硬件编辑页展示厂商服务入口
export function getAgentServiceUrl(agent: Row | undefined) {
  const entry = parseAgentUrls(agent?.urls).find(row => {
    const description = String(row.description ?? '')
      .trim()
      .toLowerCase();
    return (
      String(row.url ?? '').trim() &&
      (description.includes('服务') || description.includes('service'))
    );
  });
  return String(entry?.url ?? '').trim();
}

export function getSortValue(row: Row, key: string): unknown {
  if (key === 'warrantyRemain') {
    const purchase = Number(row.purchasedate ?? row.purchaseDate ?? 0);
    const warrantyMonths = Number(row.warrantymonths ?? row.warrantyMonths ?? 0);
    if (!purchase || !warrantyMonths) return Number.NEGATIVE_INFINITY;
    const end = new Date(purchase * 1000);
    end.setMonth(end.getMonth() + warrantyMonths);
    return Math.floor((end.getTime() - Date.now()) / 86_400_000);
  }
  if (key === 'qty') {
    const used = Number(row.usedqty ?? 0);
    const total = Number(row.licqty ?? 0);
    return `${used}/${total}`;
  }
  return row[key];
}

export function compareTableValues(left: unknown, right: unknown) {
  const leftNumber = toComparableNumber(left);
  const rightNumber = toComparableNumber(right);
  if (leftNumber !== null && rightNumber !== null) return leftNumber - rightNumber;
  return String(left ?? '').localeCompare(String(right ?? ''), 'zh-CN', {
    numeric: true,
    sensitivity: 'base',
  });
}

export function toComparableNumber(value: unknown) {
  if (typeof value === 'number' && Number.isFinite(value)) return value;
  if (typeof value !== 'string' || !value.trim() || !/^[-+]?\d+(\.\d+)?$/.test(value.trim()))
    return null;
  const numberValue = Number(value);
  return Number.isFinite(numberValue) ? numberValue : null;
}

export function getSoftwareQuantity(row: Row) {
  const used = Number(row.usedqty ?? 0);
  const rawTotal = row.licqty;
  const total = Number(rawTotal ?? 0);
  if (total > 0)
    return {
      text: `${used}/${total}`,
      tone: used > total ? 'is-over' : used < total ? 'is-available' : 'is-full',
    };
  if (rawTotal !== null && rawTotal !== undefined && String(rawTotal).trim() !== '')
    return { text: `0/${total}`, tone: 'is-zero' };
  if (used > 0) return { text: `${used}/-`, tone: 'is-over' };
  return { text: '-', tone: 'is-full' };
}

export function parseSoftwareInstalledEntries(rawValue: unknown) {
  const raw = String(rawValue ?? '').trim();
  if (!raw || raw === '-') return [] as { index: number; id: number | null; text: string }[];
  const parts = raw.includes('|')
    ? raw
        .split('|')
        .map(part => part.trim())
        .filter(Boolean)
    : [...raw.matchAll(/\(\d+\)\s*/g)].length > 1
      ? [...raw.matchAll(/\(\d+\)\s*/g)].map((match, index, matches) =>
          raw
            .slice(match.index, matches[index + 1]?.index ?? raw.length)
            .replace(/[|,]\s*$/, '')
            .trim()
        )
      : [raw];
  return parts.map((part, index) => {
    const match = part.match(/^\((\d+)\)\s*(.*)$/);
    return {
      index: index + 1,
      id: match ? Number(match[1]) : null,
      text: (match?.[2] ?? part).trim(),
    };
  });
}

export function parseInvoiceFileEntries(rawValue: unknown) {
  const raw = String(rawValue ?? '').trim();
  if (!raw || raw === '-')
    return [] as { index: number; id: number | null; title: string; text: string }[];
  return raw
    .split('|')
    .map(part => part.trim())
    .filter(Boolean)
    .map((part, index) => {
      const match = part.match(/^\((\d+)\)\s*(.*)$/);
      if (!match) {
        const plainText = part.trim();
        return { index: index + 1, id: null, title: '', text: plainText };
      }
      const id = Number(match[1]);
      const rawText = match[2].trim();
      const splitParts = rawText.includes(' / ') ? rawText.split(' / ', 2) : [];
      const title = String(splitParts[0] ?? '').trim();
      const fileName = String(splitParts[1] ?? (splitParts.length > 0 ? '' : rawText)).trim();
      return {
        index: index + 1,
        id: Number.isFinite(id) && id > 0 ? id : null,
        title,
        text: fileName || rawText || `编号=${id}`,
      };
    });
}

export function getInstalledItemBadgeStyle(id: number, itemColors: Map<number, string>) {
  const color = itemColors.get(id) ?? '#2563eb';
  return { backgroundColor: hexToRgba(color, 0.13), borderColor: hexToRgba(color, 0.28), color };
}

export function getItemIDBadgeStyle(row: Row, statusColors: Map<string, string>) {
  const direct = String(row.statuscolor ?? row.statusColor ?? '').trim();
  const color = isHexColor(direct) ? direct : statusColors.get(String(row.status ?? '').trim());
  if (!isHexColor(color)) return undefined;
  return {
    backgroundColor: hexToRgba(color, 0.13),
    borderColor: hexToRgba(color, 0.3),
    color,
  };
}

export function isHexColor(value: string | undefined): value is string {
  return Boolean(value && /^#([\da-f]{3}|[\da-f]{6})$/i.test(value));
}

export function hexToRgba(color: string, alpha: number) {
  const raw = color.slice(1);
  const hex =
    raw.length === 3
      ? raw
          .split('')
          .map(part => `${part}${part}`)
          .join('')
      : raw;
  const [red, green, blue] = [0, 2, 4].map(index =>
    Number.parseInt(hex.slice(index, index + 2), 16)
  );
  return `rgba(${red}, ${green}, ${blue}, ${alpha})`;
}

export function getWarrantyRemain(row: Row) {
  const purchase = Number(row.purchasedate ?? row.purchaseDate ?? 0);
  const warrantyMonths = Number(row.warrantymonths ?? row.warrantyMonths ?? 0);
  if (!purchase || !warrantyMonths) return { text: '-', expired: false, endDate: '' };
  const start = new Date(purchase * 1000);
  const end = new Date(start.getTime());
  end.setMonth(end.getMonth() + warrantyMonths);
  const now = new Date();
  if (end.getTime() < now.getTime())
    return { text: '已过保', expired: true, endDate: formatDateOnly(end) };
  const diff = diffYMD(now, end);
  return {
    text:
      diff.years > 0
        ? `${diff.years} 年 ${diff.months} 月, ${diff.days} 天`
        : `${diff.months} 月, ${diff.days} 天`,
    expired: false,
    endDate: formatDateOnly(end),
  };
}

export function diffYMD(from: Date, to: Date) {
  let years = to.getFullYear() - from.getFullYear();
  let cursor = new Date(from.getTime());
  cursor.setFullYear(cursor.getFullYear() + years);
  if (cursor > to) {
    years -= 1;
    cursor = new Date(from.getTime());
    cursor.setFullYear(cursor.getFullYear() + years);
  }
  let months = (to.getFullYear() - cursor.getFullYear()) * 12 + to.getMonth() - cursor.getMonth();
  cursor = new Date(cursor.getTime());
  cursor.setMonth(cursor.getMonth() + months);
  if (cursor > to) {
    months -= 1;
    cursor = new Date(from.getTime());
    cursor.setFullYear(cursor.getFullYear() + years);
    cursor.setMonth(cursor.getMonth() + months);
  }
  const days = Math.max(0, Math.floor((to.getTime() - cursor.getTime()) / 86_400_000));
  return { years: Math.max(0, years), months: Math.max(0, months), days };
}

export function formatValue(value: unknown, fieldName?: string) {
  if (value === null || value === undefined || value === '') return '-';
  if (fieldName && /(?:date|purchdate|startdate|enddate)$/i.test(fieldName)) {
    const numericValue = typeof value === 'number' ? value : Number(value);
    if (Number.isFinite(numericValue) && numericValue > 1_000_000_000) {
      return formatDateOnly(new Date(numericValue * 1000));
    }
    if (Number.isFinite(numericValue) && numericValue === 0) return '-';
  }
  if (Array.isArray(value)) return value.join(', ');
  if (typeof value === 'object') return JSON.stringify(value);
  return String(value);
}

export function overviewDateText(value: unknown) {
  return Number(value) > 0 ? formatValue(value, 'date') : '-';
}

export function overviewEntryText(lookup: string, row: Row | undefined, id: number) {
  const text = (v: unknown) => String(v ?? '').trim();
  if (!row) {
    if (lookup === 'items_ref') return `- - [-, ID:${id}]`;
    if (lookup === 'invoices_ref') return `(-) - - [ID:${id}]`;
    if (lookup === 'contracts_ref') return `(- -) - -- [ID:${id}]`;
    return `- - [ID:${id}]`;
  }
  if (lookup === 'items_ref')
    return `${text(row.manufacturer) || '-'} ${text(row.model) || '-'} [${text(row.itemtype) || '-'}, ID:${id}]`;
  if (lookup === 'software_ref') {
    const name = [text(row.manufacturer) || '-', text(row.stitle) || '-', text(row.sversion)]
      .filter(Boolean)
      .join(' ');
    return `${name} [ID:${id}]`;
  }
  if (lookup === 'invoices_ref')
    return `(${text(row.number) || '-'}) - ${overviewDateText(row.date)} [ID:${id}]`;
  const label = [text(row.title) || '-', text(row.number) || '-'].filter(Boolean).join(' ');
  return `(${label}) - ${overviewDateText(row.startdate)}-${overviewDateText(row.currentenddate)} [ID:${id}]`;
}

export function formatDateOnly(date: Date) {
  const parts = new Intl.DateTimeFormat('zh-CN', {
    year: 'numeric',
    month: '2-digit',
    day: '2-digit',
  }).formatToParts(date);
  const map = new Map(parts.map(part => [part.type, part.value]));
  return `${map.get('year')}-${map.get('month')}-${map.get('day')}`;
}

export function normalizeDateInput(value: unknown) {
  const text = String(value ?? '').trim();
  if (!text) return '';
  const timestamp = Number(text);
  if (Number.isFinite(timestamp) && timestamp > 1_000_000_000)
    return formatDateOnly(new Date(timestamp * 1000));
  const match = /^(\d{4}-\d{2}-\d{2})/.exec(text);
  return match ? match[1] : text;
}

/* 表格单元格的纯文本形态：与 TableCellValue 渲染内容一致，多行内容以换行分隔，供导出使用 */
export function getResourceCellText(row: Row, columnKey: string, resourceKey?: string): string {
  if (columnKey === 'warrantyRemain') return getWarrantyRemain(row).text;
  if (columnKey === 'qty') return getSoftwareQuantity(row).text;
  if (columnKey === 'maintend') {
    const value = Number(row.maintend ?? 0);
    return Number.isFinite(value) && value > 0 ? formatValue(value, 'date') : '-';
  }
  if (columnKey === 'software') {
    const entries = String(row[columnKey] ?? '')
      .split(',')
      .map(entry => entry.trim())
      .filter(Boolean)
      .map(entry => {
        const separator = entry.indexOf('#');
        return separator <= 0 ? entry : entry.slice(separator + 1).trim();
      });
    return entries.length > 0 ? entries.join('\n') : '-';
  }
  if (columnKey === 'tags') {
    const names = String(row.tags ?? '')
      .split(',')
      .map(name => name.trim())
      .filter(Boolean);
    return names.length > 0 ? names.join('\n') : '-';
  }
  if (columnKey === 'vendor') {
    const names = String(row.vendor ?? '')
      .split(',')
      .map(name => name.trim())
      .filter(Boolean);
    return names.length > 0 ? names.join('\n') : '-';
  }
  if (columnKey === 'invoice') {
    const numbers = String(row.invoice ?? '')
      .split(',')
      .map(part => part.trim())
      .filter(Boolean)
      .map(part => {
        const sep = part.indexOf(':');
        return sep === -1 ? part : part.slice(sep + 1) || part.slice(0, sep);
      });
    return numbers.length > 0 ? numbers.join('\n') : '-';
  }
  if (columnKey === 'usize' || columnKey === 'depth') {
    const value = formatValue(row[columnKey], columnKey);
    if (!value || value === '-') return '-';
    return columnKey === 'usize' ? `${value}U` : `${value}mm`;
  }
  if (columnKey === 'occupation') {
    const usedRaw = Number(row.occupation ?? 0);
    const totalRaw = Number(row.usize ?? 0);
    const used = Number.isFinite(usedRaw) && usedRaw > 0 ? usedRaw : 0;
    const total = Number.isFinite(totalRaw) && totalRaw > 0 ? totalRaw : 0;
    if (!total) return '-';
    return `${used}U 在用 / 总计 ${total}U`;
  }
  if (columnKey === 'installedon') {
    const entries = parseSoftwareInstalledEntries(row[columnKey]);
    if (entries.length === 0) return '-';
    return entries.map(entry => (entry.id ? `${entry.id} ${entry.text}` : entry.text)).join('\n');
  }
  if (columnKey === 'type' && resourceKey === 'agents') {
    const badges = parseAgentTypeBadges(row[columnKey]);
    if (badges.length === 0) return '-';
    return badges.map(badge => badge.label).join('\n');
  }
  if (columnKey === 'contacts') {
    const contacts = getAgentContactEntries(row[columnKey]);
    if (contacts.length === 0) return '-';
    return contacts.map(fields => fields.map(field => field.value).join('/')).join('\n');
  }
  if (columnKey === 'fname') return String(row.fname ?? '').trim() || '-';
  if (columnKey === 'links') {
    const count = Number(row.links ?? 0);
    return Number.isFinite(count) ? String(count) : '-';
  }
  if (columnKey === 'floorplanfn') return String(row.floorplanfn ?? '').trim() || '-';
  if (columnKey === 'areaname') {
    const areas = String(row[columnKey] ?? '')
      .split(',')
      .map(value => value.trim())
      .filter(Boolean);
    return areas.length > 0 ? areas.join('\n') : '-';
  }
  if (columnKey === 'files') {
    const entries = parseInvoiceFileEntries(row[columnKey]);
    if (entries.length === 0) return '-';
    return entries.map(entry => `${entry.id ?? entry.index} ${entry.text}`).join('\n');
  }
  if (columnKey === 'population') return String(Number(row.population ?? 0));
  return formatValue(row[columnKey], columnKey);
}

export function getRowSearchText(row: Row, columnKeys: string[]) {
  const parts = columnKeys.map(key => {
    if (key === 'warrantyRemain') return getWarrantyRemain(row).text;
    if (key === 'type' && row.typetext !== undefined) return String(row.typetext);
    if (key === 'maintend') return formatValue(row[key], 'date');
    if (key === 'qty') return getSoftwareQuantity(row).text;
    if (key === 'usize' || key === 'depth') {
      const value = formatValue(row[key], key);
      if (!value || value === '-') return '';
      return key === 'usize' ? `${value}U` : `${value}mm`;
    }
    if (key === 'occupation') {
      const usedRaw = Number(row.occupation ?? 0);
      const totalRaw = Number(row.usize ?? 0);
      const used = Number.isFinite(usedRaw) && usedRaw > 0 ? usedRaw : 0;
      const total = Number.isFinite(totalRaw) && totalRaw > 0 ? totalRaw : 0;
      if (!total) return '';
      return `${used}U 在用 / 总计 ${total}U`;
    }
    return formatValue(row[key], key);
  });
  return parts.join(' ').toLowerCase();
}
