import { createFileRoute, redirect } from '@tanstack/react-router';

export const Route = createFileRoute('/_authenticated/assets/users')({
  beforeLoad: () => {
    throw redirect({ to: '/settings' });
  },
});
