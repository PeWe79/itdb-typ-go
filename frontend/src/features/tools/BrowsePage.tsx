import { useCallback, useEffect, useMemo, useState } from 'react';
import { toast } from 'sonner';
import { AppTooltip } from '@/components/app-tooltip';
import { PermissionGate } from '@/components/permission-gate';
import { api, getStoredUser, userHasPermission } from '@/lib/auth';
import { PERM } from '@/lib/permissions';
import { canOpenRecordEditor } from '@/features/assets/record-links';

type BrowseNode = {
  id: string;
  label: string;
  leaf: boolean;
  resource?: string;
  entityId?: number;
};

type TreeRow = {
  node: BrowseNode;
  prefix: string;
  expanded: boolean;
  loading: boolean;
};

const ROOT_ID = '0';

// 叶子资源到资产列表页的映射与文案，resource 为空（如用户）时点击不跳转
const resourcePathMap: Record<string, { path: string; noun: string }> = {
  items: { path: '/assets/hardware', noun: '硬件' },
  software: { path: '/assets/software', noun: '软件' },
  invoices: { path: '/assets/invoices', noun: '单据' },
  contracts: { path: '/assets/contracts', noun: '合同' },
};

export function BrowsePage() {
  const [nodesById, setNodesById] = useState<Record<string, BrowseNode>>({});
  const [childrenByParent, setChildrenByParent] = useState<Record<string, string[]>>({});
  const [expandedById, setExpandedById] = useState<Record<string, boolean>>({});
  const [loadingByParent, setLoadingByParent] = useState<Record<string, boolean>>({});
  const [selectedId, setSelectedId] = useState('');
  const [rootLoading, setRootLoading] = useState(false);

  const loadChildren = useCallback(async (parentId: string) => {
    setLoadingByParent(current => ({ ...current, [parentId]: true }));
    try {
      const nodes = await api<BrowseNode[]>(`/api/browse/tree?id=${encodeURIComponent(parentId)}`);
      setNodesById(current => {
        const next = { ...current };
        for (const node of nodes) next[node.id] = node;
        return next;
      });
      setChildrenByParent(current => ({
        ...current,
        [parentId]: nodes.map(node => node.id),
      }));
    } catch (error) {
      toast.error(error instanceof Error ? error.message : '浏览树加载失败');
    } finally {
      setLoadingByParent(current => ({ ...current, [parentId]: false }));
    }
  }, []);

  const resetTree = useCallback(async () => {
    setNodesById({});
    setChildrenByParent({});
    setExpandedById({});
    setSelectedId('');
    setRootLoading(true);
    await loadChildren(ROOT_ID);
    setRootLoading(false);
  }, [loadChildren]);

  useEffect(() => {
    if (!userHasPermission(getStoredUser(), PERM.browseRead)) return;
    void resetTree();
  }, [resetTree]);

  const toggleNode = useCallback(
    async (node: BrowseNode) => {
      const nextExpanded = !expandedById[node.id];
      setExpandedById(current => ({ ...current, [node.id]: nextExpanded }));
      if (nextExpanded && !Object.hasOwn(childrenByParent, node.id)) {
        await loadChildren(node.id);
      }
    },
    [childrenByParent, expandedById, loadChildren]
  );

  const choose = useCallback(
    async (node: BrowseNode) => {
      setSelectedId(node.id);
      if (node.leaf) {
        const target = node.resource ? resourcePathMap[node.resource] : undefined;
        const entityId = Number(node.entityId ?? 0);
        if (target && entityId > 0 && canOpenRecordEditor(target.path)) {
          window.open(`${target.path}?edit=${entityId}`, '_blank', 'noopener');
        }
        return;
      }
      await toggleNode(node);
    },
    [toggleNode]
  );

  const treeRows = useMemo<TreeRow[]>(() => {
    const rows: TreeRow[] = [];
    const walk = (parentId: string, ancestorLastFlags: boolean[]) => {
      const childIds = childrenByParent[parentId] ?? [];
      childIds.forEach((id, index) => {
        const node = nodesById[id];
        if (!node) return;
        const isLast = index === childIds.length - 1;
        let prefix = '';
        for (const flag of ancestorLastFlags) {
          prefix += flag ? '   ' : '│  ';
        }
        prefix += isLast ? '└─ ' : '├─ ';
        rows.push({
          node,
          prefix,
          expanded: Boolean(expandedById[id]),
          loading: Boolean(loadingByParent[id]),
        });
        if (!node.leaf && expandedById[id]) {
          walk(id, [...ancestorLastFlags, isLast]);
        }
      });
    };
    walk(ROOT_ID, []);
    return rows;
  }, [childrenByParent, expandedById, loadingByParent, nodesById]);

  const expandedChildCount = (node: BrowseNode) => (childrenByParent[node.id] ?? []).length;
  const shouldShowCount = (node: BrowseNode) => {
    if (node.leaf || !expandedById[node.id]) return false;
    if (!Object.hasOwn(childrenByParent, node.id)) return false;
    const childIds = childrenByParent[node.id] ?? [];
    if (childIds.length === 0) return true;
    return childIds.every(childId => nodesById[childId]?.leaf);
  };

  return (
    <PermissionGate anyOf={[PERM.browseRead]}>
      <div className="flex min-h-0 flex-1 flex-col gap-3">
        <div className="itdb-surface-3d flex min-h-0 flex-1 flex-col rounded-xl">
          <div className="itdb-hidden-scrollbar min-h-0 flex-1 overflow-y-auto p-3">
            {rootLoading ? (
              <div className="grid min-h-40 place-items-center text-sm text-[var(--itdb-text-muted)]">
                加载中
              </div>
            ) : (
              <ul className="grid gap-0.5">
                {treeRows.map(row => (
                  <li key={row.node.id}>
                    <button
                      type="button"
                      className={`flex min-h-8 w-full items-center gap-1 rounded-lg border border-transparent px-1.5 py-1 text-left text-sm text-[var(--itdb-text)] transition-colors hover:bg-[rgba(59,130,246,0.08)] ${
                        selectedId === row.node.id
                          ? 'border-[rgba(59,130,246,0.45)] bg-[rgba(59,130,246,0.12)]'
                          : ''
                      }`}
                      onClick={() => void choose(row.node)}
                    >
                      <span className="itdb-hidden-scrollbar shrink-0 whitespace-pre font-mono text-xs text-[var(--itdb-text-muted)]">
                        {row.prefix}
                      </span>
                      <span className="w-4 shrink-0 text-center text-xs text-[var(--itdb-accent-text)]">
                        {row.node.leaf ? '↗' : row.expanded ? '▾' : '▸'}
                      </span>
                      {row.node.leaf ? (
                        <AppTooltip
                          className="min-w-0 shrink"
                          placement="top"
                          label={
                            row.node.resource &&
                            resourcePathMap[row.node.resource] &&
                            canOpenRecordEditor(resourcePathMap[row.node.resource].path)
                              ? `在新窗口编辑${resourcePathMap[row.node.resource].noun}编号 ${row.node.entityId}`
                              : undefined
                          }
                        >
                          <span className="block truncate font-medium text-[var(--itdb-accent-text)] underline decoration-[var(--itdb-accent-text)]/40 underline-offset-2">
                            {row.node.label}
                          </span>
                        </AppTooltip>
                      ) : (
                        <span className="min-w-0 shrink truncate">{row.node.label}</span>
                      )}
                      <span className="min-w-8 flex-1" />
                      {shouldShowCount(row.node) ? (
                        <span
                          className={`ml-2 shrink-0 rounded-full border px-2.5 py-0.5 text-xs font-medium ${
                            expandedChildCount(row.node) === 0
                              ? 'border-[var(--itdb-border)] text-[var(--itdb-text-muted)]'
                              : 'border-[rgba(59,130,246,0.35)] bg-[rgba(59,130,246,0.1)] text-[var(--itdb-accent-text)]'
                          }`}
                        >
                          共 {expandedChildCount(row.node)} 条
                        </span>
                      ) : null}
                      {row.loading ? (
                        <span className="ml-2 shrink-0 text-xs text-[var(--itdb-text-muted)]">
                          加载中...
                        </span>
                      ) : null}
                    </button>
                  </li>
                ))}
                {!rootLoading && treeRows.length === 0 ? (
                  <li className="px-2 py-6 text-center text-sm text-[var(--itdb-text-muted)]">
                    空
                  </li>
                ) : null}
              </ul>
            )}
          </div>
        </div>
        <p className="shrink-0 text-xs text-[var(--itdb-text-muted)]">
          点击分组展开下级，点击带下划线的条目在新窗口打开对应资产；重置将收起全部节点。
        </p>
      </div>
    </PermissionGate>
  );
}
