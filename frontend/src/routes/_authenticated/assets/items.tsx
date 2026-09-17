import { createFileRoute, redirect } from '@tanstack/react-router';

export const Route = createFileRoute('/_authenticated/assets/items')({
  beforeLoad: () => {
    throw redirect({ to: '/assets/hardware', replace: true });
  },
});
