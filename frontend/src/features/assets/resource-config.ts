export type FieldType =
  | 'text'
  | 'number'
  | 'textarea'
  | 'date'
  | 'select'
  | 'multiselect'
  | 'password'
  | 'file'
  | 'contacts'
  | 'urls';

export type ResourceField = {
  key: string;
  label: string;
  type: FieldType;
  required?: boolean;
  options?: { label: string; value: string | number; tooltip?: string }[];
  optionsKey?: string;
  readKey?: string;
  tooltip?: string;
  tooltipWrap?: boolean;
  defaultValue?: unknown;
  accept?: string;
  min?: number;
  noSpinner?: boolean;
  textValue?: boolean;
};

export type ResourceConfig = {
  key: string;
  title: string;
  endpoint: string;
  columns: { key: string; label: string; tooltip?: string }[];
  fields: ResourceField[];
  multipart?: boolean;
  readonly?: boolean;
  actionHeader?: string;
};

export const resources: ResourceConfig[] = [
  {
    key: 'items',
    title: '硬件',
    endpoint: '/items',
    columns: [
      { key: 'id', label: '编号' },
      { key: 'label', label: '标签' },
      { key: 'itemType', label: '硬件类型' },
      { key: 'manufacturer', label: '厂商' },
      { key: 'model', label: '型号' },
      { key: 'sn', label: '设备序列号' },
      { key: 'purchasedate', label: '采购日期' },
      { key: 'warrantyRemain', label: '维保剩余天数' },
      { key: 'ipv4', label: 'IPv4' },
      { key: 'dpt', label: '所属部门' },
      { key: 'principal', label: '负责人' },
      { key: 'status', label: '状态' },
      { key: 'location', label: '地点' },
      { key: 'rack', label: '机架' },
      { key: 'function', label: '用途' },
      { key: 'software', label: '软件' },
      { key: 'remadmip', label: '远程管理IP' },
      { key: 'tags', label: '标记' },
    ],
    fields: [
      { key: 'label', label: '标签', type: 'text', tooltip: '在可打印表格上也显示此文本' },
      {
        key: 'itemTypeId',
        readKey: 'itemtypeid',
        tooltip: '根据硬件类型分组',
        label: '硬件类型',
        type: 'select',
        required: true,
        optionsKey: 'itemtypes',
      },
      {
        key: 'isPart',
        readKey: 'ispart',
        label: '从属部件',
        type: 'select',
        required: true,
        options: [
          { label: '否', value: 0 },
          { label: '是', value: 1 },
        ],
      },
      {
        key: 'rackMountable',
        readKey: 'rackmountable',
        label: '机架式',
        type: 'select',
        required: true,
        options: [
          { label: '否', value: 0 },
          { label: '是', value: 1 },
        ],
      },
      {
        key: 'manufacturerId',
        readKey: 'manufacturerid',
        tooltip: '根据代理菜单中定义的硬件厂商分组',
        label: '厂商',
        type: 'select',
        required: true,
        optionsKey: 'agents',
      },
      { key: 'model', label: '型号', type: 'text', required: true },
      {
        key: 'uSize',
        readKey: 'usize',
        label: '大小(U)',
        type: 'select',
        options: Array.from({ length: 44 }, (_, index) => ({
          label: String(index + 1),
          value: index + 1,
        })),
      },
      { key: 'sn', label: '设备序列号', type: 'text' },
      { key: 'sn2', label: '序列号2', type: 'text' },
      { key: 'sn3', label: 'Service Tag', type: 'text' },
      { key: 'comments', label: '注释', type: 'textarea' },
      { key: 'principal', label: '负责人', type: 'text', required: true },
      {
        key: 'status',
        readKey: 'status',
        label: '状态',
        type: 'select',
        required: true,
        optionsKey: 'statustypes',
      },
      { key: 'dptId', readKey: 'dptid', label: '所属部门', type: 'select', optionsKey: 'dpttypes' },
      {
        key: 'locationId',
        readKey: 'locationid',
        label: '地点',
        type: 'select',
        optionsKey: 'locations',
      },
      {
        key: 'locAreaId',
        readKey: 'locareaid',
        label: '区域/房间',
        type: 'select',
        optionsKey: 'locareas',
      },
      { key: 'rackId', readKey: 'rackid', label: '机架', type: 'select', optionsKey: 'racks' },
      {
        key: 'rackPosition',
        readKey: 'rackposition',
        label: '机架位置',
        type: 'select',
        tooltip: '机架行',
      },
      {
        key: 'rackPosDepth',
        readKey: 'rackposdepth',
        label: '机架深度位',
        tooltip: '占用机架深度。(F)前, (M)中, (B)后',
        type: 'select',
        defaultValue: 4,
        options: [
          { value: 4, label: 'F-- 前侧' },
          { value: 6, label: 'FM- 前中' },
          { value: 3, label: '-MB 中后' },
          { value: 2, label: '-M- 中部' },
          { value: 1, label: '--B 后侧' },
          { value: 7, label: 'FMB 全深' },
        ],
      },
      { key: 'function', label: '用途', type: 'text' },
      { key: 'userId', readKey: 'userid', label: '使用人', type: 'select', optionsKey: 'users' },
      { key: 'purchaseDate', readKey: 'purchasedate', label: '采购日期', type: 'date' },
      { key: 'warrantyMonths', readKey: 'warrantymonths', label: '维保月数', type: 'number' },
      { key: 'warrInfo', readKey: 'warrinfo', label: '维保信息', type: 'textarea' },
      { key: 'maintenanceInfo', readKey: 'maintenanceinfo', label: '维护记录', type: 'textarea' },
      { key: 'hd', label: '硬盘', type: 'text' },
      { key: 'ram', label: '内存', type: 'text' },
      { key: 'cpu', label: 'CPU型号', type: 'text' },
      {
        key: 'cpuNo',
        readKey: 'cpuno',
        label: 'CPU数量',
        type: 'text',
        tooltip: '按CPU/按核心类型授权时，用于统计已用授权数量',
      },
      {
        key: 'coresPerCpu',
        readKey: 'corespercpu',
        label: '每CPU核心数',
        type: 'text',
        tooltip: '按核心类型授权时，与 CPU 数量相乘统计已用授权数量',
      },
      { key: 'raid', label: 'Raid卡型号', type: 'text' },
      { key: 'raidConfig', readKey: 'raidconfig', label: 'Raid配置', type: 'textarea' },
      { key: 'macs', label: 'MACs', type: 'text' },
      { key: 'ipv4', label: 'IPv4', type: 'text' },
      { key: 'ipv6', label: 'IPv6', type: 'text' },
      { key: 'remAdmIp', readKey: 'remadmip', label: '远程管理IP', type: 'text' },
      { key: 'dnsName', readKey: 'dnsname', label: '管理跳线', type: 'text' },
      { key: 'panelPort', readKey: 'panelport', label: 'Bond名称', type: 'text' },
      {
        key: 'switchId',
        readKey: 'switchid',
        label: '交换机',
        type: 'select',
        optionsKey: 'items_ref',
      },
      { key: 'switchPort', readKey: 'switchport', label: '业务跳线', type: 'text' },
      { key: 'ports', label: '网络端口', type: 'text' },
      { key: 'purchPrice', readKey: 'purchprice', label: '采购价格(￥)', type: 'text' },
      {
        key: 'origin',
        label: '供应商',
        type: 'select',
        textValue: true,
        tooltip: '诸如捐赠者、供应商之类信息最好在相关单据中录入',
      },
      { key: 'itemLinks', label: '内部硬件关联', type: 'multiselect', optionsKey: 'items_ref' },
      { key: 'invoiceLinks', label: '关联单据', type: 'multiselect', optionsKey: 'invoices_ref' },
      { key: 'softwareLinks', label: '关联软件', type: 'multiselect', optionsKey: 'software_ref' },
      { key: 'contractLinks', label: '关联合同', type: 'multiselect', optionsKey: 'contracts_ref' },
      { key: 'fileLinks', label: '管理文件', type: 'multiselect', optionsKey: 'files_ref' },
    ],
  },
  {
    key: 'software',
    title: '软件',
    endpoint: '/software',
    columns: [
      { key: 'id', label: '编号' },
      { key: 'manufacturer', label: '厂商' },
      { key: 'title', label: '标题' },
      { key: 'version', label: '版本' },
      { key: 'purchdate', label: '采购日期' },
      { key: 'maintend', label: '维护结束', tooltip: '维护截止日期' },
      { key: 'slicenseinfo', label: '许可信息' },
      { key: 'sinfo', label: '其它信息' },
      { key: 'tags', label: '标记' },
      { key: 'qty', label: '数量' },
      { key: 'vendor', label: '供应商', tooltip: '从相关单据获取' },
      { key: 'invoice', label: '单据' },
      { key: 'installedon', label: '安装于' },
    ],
    fields: [
      { key: 'title', readKey: 'stitle', label: '标题', type: 'text', required: true },
      { key: 'version', readKey: 'sversion', label: '版本', type: 'text', required: true },
      {
        key: 'manufacturerId',
        readKey: 'manufacturerid',
        label: '厂商',
        type: 'select',
        required: true,
        optionsKey: 'agents',
        tooltip: '可在“代理”菜单中新增更多厂商',
      },
      {
        key: 'purchaseDate',
        readKey: 'purchdate',
        label: '采购日期',
        type: 'date',
        required: true,
      },
      {
        key: 'licenseQty',
        readKey: 'licqty',
        label: '授权数量',
        type: 'number',
        min: 0,
        noSpinner: true,
      },
      {
        key: 'licenseType',
        readKey: 'lictype',
        label: '授权类型',
        type: 'select',
        defaultValue: 0,
        options: [
          { label: '按设备', value: 0, tooltip: '每台硬件占用 1 个授权' },
          { label: '按CPU', value: 1, tooltip: '按各硬件的 CPU 数量占用授权' },
          { label: '按核心', value: 2, tooltip: '按各硬件的 CPU 数 × 核心数占用授权' },
        ],
      },
      { key: 'slicenseInfo', readKey: 'slicenseinfo', label: '许可信息', type: 'textarea' },
      { key: 'info', readKey: 'sinfo', label: '其它信息', type: 'textarea' },
      { key: 'itemLinks', label: '关联硬件', type: 'multiselect', optionsKey: 'items_ref' },
      { key: 'invoiceLinks', label: '关联单据', type: 'multiselect', optionsKey: 'invoices_ref' },
      { key: 'contractLinks', label: '关联合同', type: 'multiselect', optionsKey: 'contracts_ref' },
      { key: 'fileLinks', label: '管理文件', type: 'multiselect', optionsKey: 'files_ref' },
    ],
  },
  {
    key: 'invoices',
    title: '单据',
    endpoint: '/invoices',
    actionHeader: '操作',
    columns: [
      { key: 'id', label: '编号' },
      { key: 'vendor', label: '供应商' },
      { key: 'buyer', label: '采购方' },
      { key: 'date', label: '日期' },
      { key: 'number', label: '订单编号' },
      { key: 'description', label: '描述' },
      { key: 'files', label: '关联文件' },
    ],
    fields: [
      { key: 'number', label: '订单编号', type: 'text', required: true },
      {
        key: 'vendorId',
        readKey: 'vendorid',
        label: '供应商',
        type: 'select',
        required: true,
        optionsKey: 'agents',
      },
      {
        key: 'buyerId',
        readKey: 'buyerid',
        label: '采购方',
        type: 'select',
        required: true,
        optionsKey: 'agents',
      },
      { key: 'date', readKey: 'date', label: '日期', type: 'date', required: true },
      { key: 'description', label: '描述', type: 'textarea' },
      { key: 'itemLinks', label: '关联硬件', type: 'multiselect', optionsKey: 'items_ref' },
      { key: 'softwareLinks', label: '关联软件', type: 'multiselect', optionsKey: 'software_ref' },
      { key: 'contractLinks', label: '关联合同', type: 'multiselect', optionsKey: 'contracts_ref' },
      { key: 'fileLinks', label: '管理文件', type: 'multiselect', optionsKey: 'files_ref' },
    ],
  },
  {
    key: 'agents',
    title: '代理',
    endpoint: '/agents',
    columns: [
      { key: 'id', label: '编号' },
      { key: 'type', label: '类型' },
      { key: 'title', label: '名称' },
      { key: 'contactinfo', label: '合同信息' },
      { key: 'contacts', label: '合同' },
    ],
    fields: [
      { key: 'title', label: '名称', type: 'text', required: true },
      {
        key: 'types',
        readKey: 'type',
        label: '类型',
        type: 'multiselect',
        required: true,
        tooltip:
          '供应商/采购方用于单据与合同，软件厂商用于软件，硬件厂商用于硬件，承包方用于合同。',
        tooltipWrap: true,
        options: [
          { label: '供应商', value: 4 },
          { label: '软件厂商', value: 2 },
          { label: '硬件厂商', value: 8 },
          { label: '采购方', value: 1 },
          { label: '承包方', value: 16 },
        ],
      },
      {
        key: 'contactInfo',
        readKey: 'contactinfo',
        label: '合同信息',
        type: 'textarea',
        tooltip: '地址, 电话号码, 其余信息',
      },
      { key: 'contacts', label: '联系人', type: 'contacts' },
      { key: 'urls', label: 'URLs', type: 'urls' },
    ],
  },
  {
    key: 'files',
    title: '文件',
    endpoint: '/files',
    multipart: true,
    actionHeader: '操作',
    columns: [
      { key: 'id', label: '编号' },
      { key: 'typedesc', label: '类型' },
      { key: 'title', label: '标题' },
      { key: 'fname', label: '文件' },
      { key: 'links', label: '关联数' },
    ],
    fields: [
      { key: 'title', label: '标题', type: 'text', required: true },
      {
        key: 'typeId',
        readKey: 'type',
        label: '类型',
        type: 'select',
        required: true,
        optionsKey: 'filetypes',
      },
      { key: 'file', label: '上传文件', type: 'file', required: true },
      { key: 'date', readKey: 'date', label: '签署日期', type: 'date' },
      { key: 'itemLinks', label: '关联硬件', type: 'multiselect', optionsKey: 'items_ref' },
      { key: 'softwareLinks', label: '关联软件', type: 'multiselect', optionsKey: 'software_ref' },
      { key: 'invoiceLinks', label: '关联单据', type: 'multiselect', optionsKey: 'invoices_ref' },
      { key: 'fileLinks', label: '管理文件', type: 'multiselect', optionsKey: 'files_ref' },
      { key: 'contractLinks', label: '关联合同', type: 'multiselect', optionsKey: 'contracts_ref' },
    ],
  },
  {
    key: 'contracts',
    title: '合同',
    endpoint: '/contracts',
    columns: [
      { key: 'id', label: '编号' },
      { key: 'parentid', label: '上级编号' },
      { key: 'type', label: '类型' },
      { key: 'number', label: '数量' },
      { key: 'title', label: '标题' },
      { key: 'startdate', label: '开始日期' },
      { key: 'currentenddate', label: '结束日期' },
    ],
    fields: [
      { key: 'title', label: '标题', type: 'text', required: true },
      { key: 'number', label: '数量', type: 'text', required: true },
      {
        key: 'typeId',
        readKey: 'type',
        label: '合同类型',
        type: 'select',
        required: true,
        optionsKey: 'contracttypes',
      },
      {
        key: 'subTypeId',
        readKey: 'subtype',
        label: '合同子类型',
        type: 'select',
        optionsKey: 'contractsubtypes',
      },
      {
        key: 'parentId',
        readKey: 'parentid',
        label: '上级合同',
        type: 'select',
        optionsKey: 'contracts_ref',
      },
      {
        key: 'contractorId',
        readKey: 'contractorid',
        label: '承包方',
        type: 'select',
        required: true,
        optionsKey: 'agents',
        tooltip: '承包方代理类型',
      },
      { key: 'startDate', readKey: 'startdate', label: '开始日期', type: 'date', required: true },
      {
        key: 'currentEndDate',
        readKey: 'currentenddate',
        label: '结束日期',
        type: 'date',
        required: true,
      },
      { key: 'renewals', label: '续签信息', type: 'text' },
      { key: 'totalCost', readKey: 'totalcost', label: '总成本', type: 'text' },
      { key: 'description', label: '合同描述', type: 'textarea' },
      { key: 'comments', label: '注释', type: 'textarea' },
      { key: 'itemLinks', label: '关联硬件', type: 'multiselect', optionsKey: 'items_ref' },
      { key: 'softwareLinks', label: '关联软件', type: 'multiselect', optionsKey: 'software_ref' },
      { key: 'invoiceLinks', label: '关联单据', type: 'multiselect', optionsKey: 'invoices_ref' },
      { key: 'fileLinks', label: '管理文件', type: 'multiselect', optionsKey: 'files_ref' },
    ],
  },
  {
    key: 'locations',
    title: '地点',
    endpoint: '/locations',
    multipart: true,
    columns: [
      { key: 'id', label: '编号' },
      { key: 'name', label: '地点名称/建筑名称' },
      { key: 'floor', label: '楼层' },
      { key: 'areaname', label: '区域/办公室' },
      { key: 'floorplanfn', label: '建筑平面图' },
    ],
    fields: [
      { key: 'name', label: '建筑名称', type: 'text', required: true },
      { key: 'floor', label: '楼层', type: 'text', required: true },
      {
        key: 'file',
        label: '建筑平面图',
        type: 'file',
        tooltip: '如果你选择了新的文件，它将替换当前文件，同时保留它的关联关系。',
        accept: '.jpg,.jpeg,.png,.gif,.bmp,.webp,.svg,.avif',
      },
    ],
  },
  {
    key: 'racks',
    title: '机架',
    endpoint: '/racks',
    columns: [
      { key: 'id', label: '编号' },
      { key: 'occupation', label: '在用' },
      { key: 'population', label: '硬件', tooltip: '多少硬件被分配到这个机架' },
      { key: 'usize', label: '高度' },
      { key: 'depth', label: '深度' },
      { key: 'location', label: '地点' },
      { key: 'area', label: '区域/房间' },
      { key: 'label', label: '标签' },
    ],
    fields: [
      { key: 'uSize', readKey: 'usize', label: '高度(U)', type: 'number', required: true },
      {
        key: 'revNums',
        readKey: 'revnums',
        label: '编号方向',
        type: 'select',
        defaultValue: 0,
        options: [
          { value: 0, label: '1=Bottom' },
          { value: 1, label: '1=Top' },
        ],
      },
      { key: 'label', label: '标签', type: 'text', required: true },
      { key: 'depth', label: '深度(mm)', type: 'number', required: true },
      { key: 'model', label: '型号', type: 'text' },
      {
        key: 'locationId',
        readKey: 'locationid',
        label: '地点',
        type: 'select',
        required: true,
        optionsKey: 'locations',
      },
      {
        key: 'locAreaId',
        readKey: 'locareaid',
        label: '区域',
        type: 'select',
        optionsKey: 'locareas',
      },
      { key: 'comments', label: '注释', type: 'textarea' },
    ],
  },
];

export const resourceMap = Object.fromEntries(resources.map(r => [r.key, r])) as Record<
  string,
  ResourceConfig
>;
