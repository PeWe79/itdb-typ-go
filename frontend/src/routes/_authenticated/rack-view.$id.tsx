import { useQuery } from '@tanstack/react-query';
import { createFileRoute } from '@tanstack/react-router';
import { X } from 'lucide-react';
import { Button } from '@/components/ui/button';
import { PermissionGate } from '@/components/permission-gate';
import { RackElevation } from '@/features/assets/components/RackPanes';
import { api } from '@/lib/auth';
import { PERM } from '@/lib/permissions';

type Row = Record<string, unknown>;
type Lookups = Record<string, Row[]>;

export const Route = createFileRoute('/_authenticated/rack-view/$id')({
  component: RackViewPage,
});

function RackViewPage() {
  const { id } = Route.useParams();
  const highlight = new URLSearchParams(window.location.search).get('highlight') ?? '';
  const rackId = Number(id);
  const highlightId = Number(highlight ?? 0);

  const lookups = useQuery({
    queryKey: ['itdb', 'bootstrap'],
    queryFn: () => api<Lookups>('/api/bootstrap'),
  });
  const items = useQuery({
    queryKey: ['itdb', 'items', 'all'],
    queryFn: () => api<Row[]>('/api/items?limit=-1&offset=0'),
  });

  const racks = Array.isArray(lookups.data?.racks) ? lookups.data.racks : [];
  const rack = racks.find(row => Number(row.id) === rackId);
  const locations = Array.isArray(lookups.data?.locations) ? lookups.data.locations : [];
  const areas = Array.isArray(lookups.data?.locareas) ? lookups.data.locareas : [];
  const location = locations.find(row => Number(row.id) === Number(rack?.locationid ?? 0));
  const area = areas.find(row => Number(row.id) === Number(rack?.locareaid ?? 0));

  const label = String(rack?.label ?? '').trim() || '-';
  const totalUnits = Number(rack?.usize ?? 0);
  const model = String(rack?.model ?? '').trim() || '-';
  const locationText = location
    ? [
        String(location.name ?? '').trim(),
        Number(location.floor) > 0 ? `楼层:${location.floor}` : '',
      ]
        .filter(Boolean)
        .join(', ')
    : '-';
  const areaText = area ? String(area.areaname ?? '').trim() || '-' : '-';
  const mountedItems = (Array.isArray(items.data) ? items.data : []).filter(
    item => Number(item.rackid) === rackId
  );

  return (
    <PermissionGate anyOf={[PERM.assetsItemsRead]}>
      <div className="itdb-hidden-scrollbar flex h-full min-h-0 flex-col overflow-y-auto">
        <header className="mb-5 flex shrink-0 flex-wrap items-start justify-between gap-4">
          <div className="min-w-0">
            <h1 className="text-lg font-semibold text-[var(--itdb-text)]">
              {label},{totalUnits}U 晟图
            </h1>
            <p className="mt-1 text-sm text-[var(--itdb-text-muted)]">
              编号: {rackId} / 型号: {model} / 地点: {locationText} / 区域: {areaText}
            </p>
          </div>
          <Button
            type="button"
            variant="outline"
            className="itdb-action-button shrink-0"
            style={{
              borderColor: 'rgba(59,130,246,0.38)',
              background: 'rgba(59,130,246,0.1)',
              color: 'var(--itdb-accent-text)',
            }}
            onClick={() => window.close()}
          >
            <X size={16} />
            关闭
          </Button>
        </header>
        <section className="itdb-resource-form-section shrink-0 rounded-xl border border-[var(--itdb-border)] bg-[var(--itdb-card)] p-5">
          {totalUnits > 0 ? (
            <RackElevation
              units={totalUnits}
              reverse={Number(rack?.revnums ?? rack?.revNums ?? 0) === 1}
              items={mountedItems}
              highlightId={highlightId > 0 ? highlightId : undefined}
            />
          ) : (
            <div className="itdb-rack-empty h-40">机架数据不存在或高度未配置</div>
          )}
        </section>
      </div>
    </PermissionGate>
  );
}
