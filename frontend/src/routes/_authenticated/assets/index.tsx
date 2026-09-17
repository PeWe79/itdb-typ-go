import { createFileRoute, useNavigate } from '@tanstack/react-router';
import { useEffect } from 'react';
import { getStoredUser, userHasPermission } from '@/lib/auth';
import { PERM } from '@/lib/permissions';

const ASSET_TABS = [
  { to: '/assets/hardware', permission: PERM.assetsItemsRead },
  { to: '/assets/software', permission: PERM.assetsSoftwareRead },
  { to: '/assets/invoices', permission: PERM.assetsInvoicesRead },
  { to: '/assets/agents', permission: PERM.assetsAgentsRead },
  { to: '/assets/files', permission: PERM.assetsFilesRead },
  { to: '/assets/contracts', permission: PERM.assetsContractsRead },
  { to: '/assets/locations', permission: PERM.assetsLocationsRead },
  { to: '/assets/racks', permission: PERM.assetsRacksRead },
] as const;

function AssetIndexRedirect() {
  const navigate = useNavigate();
  useEffect(() => {
    const user = getStoredUser();
    const target = ASSET_TABS.find(tab => userHasPermission(user, tab.permission));
    void navigate({ to: target?.to ?? '/assets/hardware', replace: true });
  }, [navigate]);
  return null;
}

export const Route = createFileRoute('/_authenticated/assets/')({
  component: AssetIndexRedirect,
});
