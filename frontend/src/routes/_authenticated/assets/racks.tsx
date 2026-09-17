import { createFileRoute } from '@tanstack/react-router';
import { ResourceListPage } from '@/features/assets/ResourceListPage';

export const Route = createFileRoute('/_authenticated/assets/racks')({
  component: () => <ResourceListPage resourceKey="racks" />,
});
