import type { ReactNode } from 'react';
import { getStoredUser, userHasAnyPermission } from '@/lib/auth';

type PermissionGateProps = {
  anyOf: string[];
  children: ReactNode;
};

// PermissionGate 页面级权限闸门，无权限时渲染整块卡片占位提示而不渲染受控内容
export function PermissionGate({ anyOf, children }: PermissionGateProps) {
  if (!userHasAnyPermission(getStoredUser(), anyOf)) {
    return (
      <div
        className="grid h-full min-h-[50vh] place-items-center rounded-xl border border-[var(--itdb-border)] bg-[var(--itdb-card)] p-6 text-sm text-[var(--itdb-text-muted)]"
        style={{ boxShadow: 'var(--shadow-card)' }}
      >
        当前账号无权访问该功能
      </div>
    );
  }
  return <>{children}</>;
}
