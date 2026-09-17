export const PERM = {
  assetsItemsRead: 'assets.items.read',
  assetsItemsManage: 'assets.items.manage',
  assetsSoftwareRead: 'assets.software.read',
  assetsSoftwareManage: 'assets.software.manage',
  assetsInvoicesRead: 'assets.invoices.read',
  assetsInvoicesManage: 'assets.invoices.manage',
  assetsAgentsRead: 'assets.agents.read',
  assetsAgentsManage: 'assets.agents.manage',
  assetsFilesRead: 'assets.files.read',
  assetsFilesManage: 'assets.files.manage',
  assetsContractsRead: 'assets.contracts.read',
  assetsContractsManage: 'assets.contracts.manage',
  assetsLocationsRead: 'assets.locations.read',
  assetsLocationsManage: 'assets.locations.manage',
  assetsRacksRead: 'assets.racks.read',
  assetsRacksManage: 'assets.racks.manage',
  dictionariesItemtypesRead: 'dictionaries.itemtypes.read',
  dictionariesItemtypesManage: 'dictionaries.itemtypes.manage',
  dictionariesContracttypesRead: 'dictionaries.contracttypes.read',
  dictionariesContracttypesManage: 'dictionaries.contracttypes.manage',
  dictionariesStatustypesRead: 'dictionaries.statustypes.read',
  dictionariesStatustypesManage: 'dictionaries.statustypes.manage',
  dictionariesFiletypesRead: 'dictionaries.filetypes.read',
  dictionariesFiletypesManage: 'dictionaries.filetypes.manage',
  dictionariesDpttypesRead: 'dictionaries.dpttypes.read',
  dictionariesDpttypesManage: 'dictionaries.dpttypes.manage',
  dictionariesTagsRead: 'dictionaries.tags.read',
  dictionariesTagsManage: 'dictionaries.tags.manage',
  labelsPreview: 'labels.preview',
  labelsPrint: 'labels.print',
  labelsManage: 'labels.manage',
  reportsRead: 'reports.read',
  reportsManage: 'reports.manage',
  browseRead: 'browse.read',
  auditRead: 'audit.read',
  auditManage: 'audit.manage',
  settingsBaseRead: 'settings.base.read',
  settingsBaseManage: 'settings.base.manage',
  settingsUsersRead: 'settings.users.read',
  settingsUsersManage: 'settings.users.manage',
  settingsAuthRead: 'settings.auth.read',
  settingsAuthManage: 'settings.auth.manage',
  settingsNotificationsRead: 'settings.notifications.read',
  settingsNotificationsManage: 'settings.notifications.manage',
} as const;

const ASSET_RESOURCE_KEYS = [
  'items',
  'software',
  'invoices',
  'agents',
  'files',
  'contracts',
  'locations',
  'racks',
] as const;

const DICTIONARY_KEYS = [
  'itemtypes',
  'contracttypes',
  'statustypes',
  'filetypes',
  'dpttypes',
  'tags',
] as const;

export const ASSETS_READ_PERMISSIONS = ASSET_RESOURCE_KEYS.map(key => `assets.${key}.read`);
export const DICTIONARIES_READ_PERMISSIONS = DICTIONARY_KEYS.map(key => `dictionaries.${key}.read`);
export const SETTINGS_READ_PERMISSIONS = [
  PERM.settingsBaseRead,
  PERM.settingsUsersRead,
  PERM.settingsAuthRead,
  PERM.settingsNotificationsRead,
];
export const ANY_READ_PERMISSIONS = [
  ...ASSETS_READ_PERMISSIONS,
  ...DICTIONARIES_READ_PERMISSIONS,
  PERM.labelsPreview,
  PERM.reportsRead,
  PERM.browseRead,
  PERM.auditRead,
  ...SETTINGS_READ_PERMISSIONS,
];

export const assetPermission = (resource: string, scope: 'read' | 'manage') =>
  `assets.${resource}.${scope}`;

export const dictionaryPermission = (name: string, scope: 'read' | 'manage') =>
  `dictionaries.${name}.${scope}`;
