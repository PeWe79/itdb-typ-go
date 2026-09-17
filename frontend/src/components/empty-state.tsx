import { Inbox } from 'lucide-react';

export function EmptyState({ title, description }: { title: string; description?: string }) {
  return (
    <div className="flex min-h-44 flex-col items-center justify-center px-6 py-10 text-center">
      <span className="grid h-11 w-11 place-items-center rounded-xl border border-[var(--itdb-border)] bg-[var(--itdb-control-bg)] text-[var(--itdb-text-muted)]">
        <Inbox size={20} aria-hidden="true" />
      </span>
      <p className="mt-3 text-sm font-medium text-[var(--itdb-text)]">{title}</p>
      {description ? (
        <p className="mt-1 max-w-md text-xs leading-5 text-[var(--itdb-text-muted)]">
          {description}
        </p>
      ) : null}
    </div>
  );
}
