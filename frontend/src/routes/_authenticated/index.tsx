import { createFileRoute } from '@tanstack/react-router';
import { ITDBDashboardPage } from '@/features/dashboard/DashboardPage';

export const Route = createFileRoute('/_authenticated/')({ component: ITDBDashboardPage });
