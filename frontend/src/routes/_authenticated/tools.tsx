import { createFileRoute } from '@tanstack/react-router';
import { ITDBToolsPage } from '@/features/tools/ToolsPage';

export const Route = createFileRoute('/_authenticated/tools')({ component: ITDBToolsPage });
