import type { ReactNode } from 'react';
import { cn } from '@/lib/utils';

export function DataTableShell({
  toolbar,
  toolbarClassName,
  footer,
  children,
  className,
  contentClassName,
}: {
  toolbar?: ReactNode;
  toolbarClassName?: string;
  footer?: ReactNode;
  children: ReactNode;
  className?: string;
  contentClassName?: string;
}) {
  return (
    <section className={cn('itdb-surface-3d overflow-hidden rounded-xl border', className)}>
      {toolbar ? (
        <div
          className={cn(
            'flex flex-wrap items-center gap-3 border-b border-[var(--itdb-border)] p-3',
            toolbarClassName
          )}
        >
          {toolbar}
        </div>
      ) : null}
      <div className={cn('overflow-x-auto', contentClassName)}>{children}</div>
      {footer}
    </section>
  );
}
