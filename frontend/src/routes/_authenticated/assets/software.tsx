import { createFileRoute } from '@tanstack/react-router';
import { ResourceListPage } from '@/features/assets/ResourceListPage';

export const Route = createFileRoute('/_authenticated/assets/software')({
  component: () => <ResourceListPage resourceKey="software" />,
});
