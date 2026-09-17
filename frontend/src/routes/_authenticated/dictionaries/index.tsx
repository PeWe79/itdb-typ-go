import { createFileRoute, useNavigate } from '@tanstack/react-router';
import { useEffect } from 'react';
import { getStoredUser, userHasPermission } from '@/lib/auth';
import { PERM } from '@/lib/permissions';

const DICTIONARY_TABS = [
  { to: '/dictionaries/itemtypes', permission: PERM.dictionariesItemtypesRead },
  { to: '/dictionaries/contracttypes', permission: PERM.dictionariesContracttypesRead },
  { to: '/dictionaries/statustypes', permission: PERM.dictionariesStatustypesRead },
  { to: '/dictionaries/filetypes', permission: PERM.dictionariesFiletypesRead },
  { to: '/dictionaries/dpttypes', permission: PERM.dictionariesDpttypesRead },
  { to: '/dictionaries/tags', permission: PERM.dictionariesTagsRead },
] as const;

function DictionaryIndexRedirect() {
  const navigate = useNavigate();
  useEffect(() => {
    const user = getStoredUser();
    const target = DICTIONARY_TABS.find(tab => userHasPermission(user, tab.permission));
    void navigate({ to: target?.to ?? '/dictionaries/itemtypes', replace: true });
  }, [navigate]);
  return null;
}

export const Route = createFileRoute('/_authenticated/dictionaries/')({
  component: DictionaryIndexRedirect,
});
