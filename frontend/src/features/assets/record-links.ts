import { getStoredUser, userHasPermission } from '@/lib/auth';
import { PERM } from '@/lib/permissions';

const recordResourcePermissions: Record<string, { read: string; manage: string }> = {
  '/assets/hardware': { read: PERM.assetsItemsRead, manage: PERM.assetsItemsManage },
  '/assets/software': { read: PERM.assetsSoftwareRead, manage: PERM.assetsSoftwareManage },
  '/assets/invoices': { read: PERM.assetsInvoicesRead, manage: PERM.assetsInvoicesManage },
  '/assets/agents': { read: PERM.assetsAgentsRead, manage: PERM.assetsAgentsManage },
  '/assets/files': { read: PERM.assetsFilesRead, manage: PERM.assetsFilesManage },
  '/assets/contracts': { read: PERM.assetsContractsRead, manage: PERM.assetsContractsManage },
  '/assets/locations': { read: PERM.assetsLocationsRead, manage: PERM.assetsLocationsManage },
  '/assets/racks': { read: PERM.assetsRacksRead, manage: PERM.assetsRacksManage },
  '/dictionaries/tags': { read: PERM.dictionariesTagsRead, manage: PERM.dictionariesTagsManage },
};

// canOpenRecordEditor 判断当前用户是否可跳转打开目标记录的编辑弹窗
export function canOpenRecordEditor(path: string) {
  return userHasPermission(getStoredUser(), recordResourcePermissions[path]?.manage ?? '');
}

// canViewResource 判断当前用户是否可查看目标资源的列表数据
export function canViewResource(path: string) {
  return userHasPermission(getStoredUser(), recordResourcePermissions[path]?.read ?? '');
}
