import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query';
import { Fragment, useEffect, useRef, useState } from 'react';
import { toast } from 'sonner';
import { showErrorToast } from '@/lib/toast-errors';
import { AppTooltip } from '@/components/app-tooltip';
import { Button } from '@/components/ui/button';
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from '@/components/ui/dialog';
import { Tabs, TabsList, TabsTrigger } from '@/components/ui/tabs';
import { api } from '@/lib/auth';
import { type ResourceConfig, type ResourceField } from '../resource-config';
import { HardwareRelationPane, isInvoiceFileRow, resolveInvoiceFileTypeId } from './HardwarePanes';
import { MaintenanceLogPane } from './MaintenanceLogPane';
import { HardwareDataPane } from './HardwareDataPanes';
import { SoftwareDataPane } from './SoftwareDataPanes';
import { InvoiceDataPane } from './InvoiceDataPanes';
import { ContractDataPane } from './ContractDataPanes';
import { ContractEventsPane, type ContractEventOp } from './ContractPanes';
import { type PendingTagChanges } from './HardwareDataPanes';
import { AgentDataPane } from './AgentPanes';
import { LocationDataPane, type LocationAreaOp } from './LocationPane';
import { RackDataPane } from './RackPanes';
import { FileDataPane } from './FilePane';
import { FieldEditor } from './FieldEditor';
import {
  editorTabs,
  editorTabMatches,
  parseContractRenewals,
  serializeContractRenewals,
  editorSection,
  resolveFieldOptions,
  parseAgentContacts,
  parseAgentUrls,
  normalizeDateInput,
  itemTypeBlocksSoftware,
} from '../resource-helpers';
import {
  buildResourceChangeNote,
  buildJsonPayload,
  validateDateRelations,
  validateInvoiceFileExtension,
  validateItemPorts,
  validateItemRackPlacement,
  validateRequiredFields,
  validateSoftwareLicenseQty,
} from './resource-editor-rules';

type Row = Record<string, unknown>;
type Lookups = Record<string, Row[]>;

export function ResourceEditor({
  resource,
  row,
  prefillId,
  onClose,
  onSaved,
}: {
  resource: ResourceConfig;
  row?: Row;
  prefillId?: number;
  onClose: () => void;
  onSaved: () => void;
}) {
  const formId = `resource-editor-${resource.key}-${row?.id ?? 'new'}`;
  const queryClient = useQueryClient();
  const [values, setValues] = useState<Row>({});
  const initialValuesRef = useRef<Row>({});
  const [activeTab, setActiveTab] = useState('main');
  const [activeHardwareSection, setActiveHardwareSection] = useState('basic');
  const [activeSoftwareSection, setActiveSoftwareSection] = useState('attribute');
  const [activeInvoiceSection, setActiveInvoiceSection] = useState('attribute');
  const [activeContractSection, setActiveContractSection] = useState('attribute');
  const [activeAgentSection, setActiveAgentSection] = useState('attribute');
  const [renewals, setRenewals] = useState<Row[]>([]);
  const initialRenewalsRef = useRef<Row[]>([]);
  const [locationPendingAreas, setLocationPendingAreas] = useState<LocationAreaOp[]>([]);
  const locationAreaNamesRef = useRef<string[]>([]);
  const [pendingContractEvents, setPendingContractEvents] = useState<ContractEventOp[]>([]);
  const [pendingTags, setPendingTags] = useState<PendingTagChanges | null>(null);
  const [pendingCleanupFileLinks, setPendingCleanupFileLinks] = useState<number[]>([]);
  const unlinkFileWithCleanup = (fileId: number) =>
    setPendingCleanupFileLinks(current =>
      current.includes(fileId) ? current : [...current, fileId].sort((left, right) => left - right)
    );
  const lookups = useQuery({
    queryKey: ['itdb', 'bootstrap'],
    queryFn: () => api<Lookups>('/api/bootstrap'),
  });
  const details = useQuery({
    queryKey: ['itdb', resource.key, row?.id],
    enabled: Boolean(row?.id),
    queryFn: () => api<Row>(`/api${resource.endpoint}/${row?.id}`),
  });
  const prefillSource = useQuery({
    queryKey: ['itdb', resource.key, 'copy', prefillId],
    queryFn: () => api<Row>(`/api${resource.endpoint}/${prefillId}`),
    enabled: Boolean(prefillId) && !row,
  });
  const rackPlacementItems = useQuery({
    queryKey: ['itdb', 'items', 'all'],
    queryFn: () => api<Row[]>('/api/items?limit=-1&offset=0'),
    enabled: resource.key === 'items',
  });
  useEffect(() => {
    let source = details.data ?? row ?? {};
    if (!row?.id && prefillId && prefillSource.data) {
      const excluded = new Set(['id', 'renewals', 'tags']);
      resource.fields.forEach(field => {
        if (field.type === 'file' || field.key.endsWith('Links')) excluded.add(field.key);
      });
      if (['invoices', 'contracts', 'files', 'agents'].includes(resource.key)) {
        excluded.add('title');
      }
      if (resource.key === 'software') {
        excluded.add('version');
        excluded.add('sversion');
      }
      if (resource.key === 'agents') {
        excluded.add('contacts');
        excluded.add('urls');
      }
      source = {};
      Object.entries(prefillSource.data).forEach(([key, value]) => {
        if (!excluded.has(key)) source[key] = value;
      });
    }
    const next: Row = {};
    resource.fields.forEach(field => {
      let value =
        source[field.key] ??
        source[field.readKey ?? field.key] ??
        (field.type === 'multiselect' ? [] : (field.defaultValue ?? ''));
      if (field.type === 'multiselect' && field.options?.length && !Array.isArray(value)) {
        const mask = Number(value);
        if (Number.isFinite(mask) && mask > 0) {
          value = field.options
            .filter(
              option =>
                Number(option.value) > 0 && (mask & Number(option.value)) === Number(option.value)
            )
            .map(option => Number(option.value));
        }
      }
      if (field.type === 'select' && !field.textValue) {
        const blank = value === '' || value === null || value === undefined;
        const num = Number(value);
        const hasZeroOption = field.options?.some(option => Number(option.value) === 0) ?? false;
        if (blank) value = '';
        else if (hasZeroOption && num === 0) value = 0;
        else if (!Number.isFinite(num) || num <= 0) value = '';
        else value = num;
        if (value === '' && field.defaultValue !== undefined && field.defaultValue !== '') {
          value = Number(field.defaultValue);
        }
      }
      if (field.type === 'select' && field.textValue) {
        value = String(value ?? '').trim();
      }
      next[field.key] =
        field.type === 'contacts'
          ? parseAgentContacts(value)
          : field.type === 'urls'
            ? parseAgentUrls(value)
            : field.type === 'date'
              ? normalizeDateInput(value)
              : value;
    });
    next.id = source.id;
    if (source.tags !== undefined) next.tags = source.tags;
    initialValuesRef.current = next;
    setValues(next);
    if (resource.key === 'contracts') {
      setRenewals(parseContractRenewals(source.renewals));
      initialRenewalsRef.current = parseContractRenewals(source.renewals);
    }
  }, [details.data, prefillSource.data, prefillId, resource.fields, resource.key, row]);
  const save = useMutation({
    mutationFn: async () => {
      const payload = new FormData();
      const useMultipart = resource.multipart;
      if (useMultipart)
        resource.fields.forEach(field => {
          const value = values[field.key];
          if (field.type === 'file') {
            if (value instanceof File) payload.append(field.key, value);
          } else if (Array.isArray(value)) payload.append(field.key, value.join(','));
          else if (value !== '' && value !== undefined && value !== null)
            payload.append(field.key, String(value));
        });
      const changeNote = row?.id
        ? buildResourceChangeNote(resource, initialValuesRef.current, values, {
            pendingTags,
            pendingAreas: resource.key === 'locations' && locationPendingAreas.length > 0,
            pendingEvents: resource.key === 'contracts' && pendingContractEvents.length > 0,
            renewalsChanged:
              resource.key === 'contracts' &&
              serializeContractRenewals(renewals) !==
                serializeContractRenewals(initialRenewalsRef.current),
          })
        : '';
      if (useMultipart) {
        payload.append('changeNote', changeNote);
        if (resource.key === 'locations') {
          payload.append('areas', JSON.stringify(locationAreaNamesRef.current));
        }
      }
      const bodyObject = buildJsonPayload(
        resource,
        values,
        renewals,
        ['items', 'software', 'invoices', 'contracts'].includes(resource.key)
          ? Array.from(new Set(pendingCleanupFileLinks)).sort((left, right) => left - right)
          : []
      );
      if (!useMultipart && row?.id) bodyObject.changeNote = changeNote;
      const body = useMultipart ? payload : JSON.stringify(bodyObject);
      const result = await api<{ id?: number }>(
        `/api${resource.endpoint}${row?.id ? `/${row.id}` : ''}`,
        {
          method: row?.id ? 'PUT' : 'POST',
          body,
        }
      );
      if (resource.key === 'locations' && locationPendingAreas.length) {
        const savedId = row?.id ?? Number(result?.id ?? 0);
        if (!savedId) throw new Error('地点保存成功但未获取到编号，区域变更未提交');
        for (const op of locationPendingAreas) {
          const base = `/api/locations/${savedId}/areas`;
          if (op.action === 'create')
            await api(base, { method: 'POST', body: JSON.stringify({ areaName: op.areaName }) });
          else if (op.action === 'update')
            await api(`${base}/${op.areaId}`, {
              method: 'PUT',
              body: JSON.stringify({ areaName: op.areaName }),
            });
          else await api(`${base}/${op.areaId}`, { method: 'DELETE' });
        }
      }
      if (
        (resource.key === 'items' || resource.key === 'software') &&
        pendingTags &&
        (pendingTags.added.length > 0 || pendingTags.removed.length > 0)
      ) {
        const savedId = row?.id ?? Number(result?.id ?? 0);
        if (!savedId) throw new Error('保存成功但未获取到编号，标记变更未提交');
        for (const name of pendingTags.removed) {
          await api(`/api/${resource.key}/${savedId}/tags`, {
            method: 'POST',
            body: JSON.stringify({ name, action: 'remove' }),
          });
        }
        for (const name of pendingTags.added) {
          await api(`/api/${resource.key}/${savedId}/tags`, {
            method: 'POST',
            body: JSON.stringify({ name, action: 'add' }),
          });
        }
      }
      if (resource.key === 'contracts' && pendingContractEvents.length) {
        const savedId = row?.id ?? Number(result?.id ?? 0);
        if (!savedId) throw new Error('合同保存成功但未获取到编号，事件变更未提交');
        for (const op of pendingContractEvents) {
          const base = `/api/contracts/${savedId}/events`;
          if (op.action === 'create')
            await api(base, { method: 'POST', body: JSON.stringify(op.data) });
          else if (op.action === 'update')
            await api(`${base}/${op.eventId}`, { method: 'PUT', body: JSON.stringify(op.data) });
          else await api(`${base}/${op.eventId}`, { method: 'DELETE' });
        }
      }
      return result;
    },
    onSuccess: async result => {
      const savedId = row?.id ?? Number(result?.id ?? 0);
      toast.success(
        row?.id
          ? `${resource.title}编号 ${savedId || '-'} 已更新`
          : `${resource.title}编号 ${savedId || '-'} 已创建`
      );
      onSaved();
      if (pendingCleanupFileLinks.length > 0) {
        const previousRows = lookups.data?.files_ref ?? [];
        const nameById = new Map<number, string>(
          previousRows.map(row => {
            const fileId = Number(row.id);
            const name =
              String(row.fname ?? '').trim() ||
              String(row.title ?? '').trim() ||
              `文件 [ID:${fileId}]`;
            return [fileId, name];
          })
        );
        const invoiceTypeId = resolveInvoiceFileTypeId(lookups.data);
        try {
          const fresh = await queryClient.fetchQuery({
            queryKey: ['itdb', 'bootstrap'],
            queryFn: () => api<Lookups>('/api/bootstrap'),
            staleTime: 0,
          });
          const remaining = new Set((fresh?.files_ref ?? []).map(row => Number(row.id)));
          pendingCleanupFileLinks
            .filter(id => !remaining.has(id))
            .forEach(id => {
              const source = previousRows.find(item => Number(item.id) === id);
              const prefix =
                source && isInvoiceFileRow(source, invoiceTypeId) ? '发票文件' : '文件';
              toast.success(`${prefix} ${nameById.get(id) ?? `#${id}`} 已删除`);
            });
        } catch {}
      }
    },
    onError: error => showErrorToast(error.message),
  });
  const update = (field: ResourceField, value: unknown) => {
    if (field.key === 'fileLinks') {
      const ids = new Set(
        Array.isArray(value) ? value.map(Number).filter(id => Number.isFinite(id)) : []
      );
      setPendingCleanupFileLinks(current => current.filter(id => ids.has(id)));
    }
    setValues(previous => {
      if (field.key === 'locationId')
        return { ...previous, locationId: value, locAreaId: '', rackId: '', rackPosition: '' };
      if (field.key === 'locAreaId')
        return { ...previous, locAreaId: value, rackId: '', rackPosition: '' };
      if (field.key === 'rackId') return { ...previous, rackId: value, rackPosition: '' };
      if (field.key === 'typeId') return { ...previous, typeId: value, subTypeId: '' };
      return { ...previous, [field.key]: value };
    });
  };
  const tabs = editorTabs(resource.key, resource.fields);
  const visibleFields = resource.fields.filter(field =>
    editorTabMatches(resource.key, field, activeTab)
  );
  const isHardwareDataTab = resource.key === 'items' && activeTab === 'main';
  const isSoftwareDataTab = resource.key === 'software' && activeTab === 'main';
  const isInvoiceDataTab = resource.key === 'invoices' && activeTab === 'main';
  const isFixedHeightTab =
    isHardwareDataTab ||
    isSoftwareDataTab ||
    isInvoiceDataTab ||
    resource.key === 'contracts' ||
    resource.key === 'agents' ||
    resource.key === 'locations' ||
    resource.key === 'files' ||
    resource.key === 'racks' ||
    (resource.key === 'items' &&
      [
        'itemLinks',
        'invoiceLinks',
        'maintenanceInfo',
        'softwareLinks',
        'contractLinks',
        'fileLinks',
      ].includes(activeTab)) ||
    ((resource.key === 'software' || resource.key === 'invoices') &&
      ['itemLinks', 'invoiceLinks', 'softwareLinks', 'contractLinks', 'fileLinks'].includes(
        activeTab
      ));
  return (
    <Dialog open onOpenChange={open => !open && onClose()}>
      <DialogContent
        onOpenAutoFocus={event => {
          if (resource.key === 'racks') event.preventDefault();
        }}
        className="itdb-dialog-panel flex h-[min(900px,calc(100vh-3rem))] max-h-[calc(100vh-3rem)] max-w-7xl flex-col gap-0 p-0"
      >
        <DialogHeader className="shrink-0 border-b px-6 py-5">
          <DialogTitle>
            {row?.id ? `编辑记录 - ${resource.title}` : `新增记录 - ${resource.title}`}
          </DialogTitle>
          <DialogDescription>填写基础信息和关联数据，带 * 的字段为必填项</DialogDescription>
        </DialogHeader>
        <Tabs
          value={activeTab}
          onValueChange={setActiveTab}
          className="flex min-h-0 flex-1 flex-col"
        >
          {tabs.length > 1 ? (
            <div className="itdb-hidden-scrollbar shrink-0 overflow-x-auto border-b px-6 py-3">
              <TabsList className="h-auto min-w-max justify-start gap-1 rounded-xl border border-[var(--itdb-border)] bg-[var(--itdb-control-bg-soft)] p-1">
                {tabs.map(tab => (
                  <TabsTrigger
                    key={tab.value}
                    value={tab.value}
                    className="itdb-editor-primary-tab h-9 rounded-lg px-3 text-sm"
                  >
                    {tab.label}
                  </TabsTrigger>
                ))}
              </TabsList>
            </div>
          ) : null}
          <form
            id={formId}
            noValidate
            onSubmit={event => {
              event.preventDefault();
              const requiredError = validateRequiredFields(
                resource.fields,
                values,
                Boolean(row?.id)
              );
              if (requiredError) {
                toast.error(requiredError);
                return;
              }
              const portsError = validateItemPorts(resource.key, values);
              if (portsError) {
                toast.error(portsError);
                return;
              }
              const licenseQtyError = validateSoftwareLicenseQty(resource.key, values);
              if (licenseQtyError) {
                toast.error(licenseQtyError);
                return;
              }
              const rackError = validateItemRackPlacement(
                resource.key,
                values,
                lookups.data,
                Array.isArray(rackPlacementItems.data) ? rackPlacementItems.data : []
              );
              if (rackError) {
                toast.error(rackError);
                return;
              }
              const dateError = validateDateRelations(resource.key, values, renewals);
              if (dateError) {
                toast.error(dateError);
                return;
              }
              const fileError = validateInvoiceFileExtension(values);
              if (fileError) {
                toast.error(fileError);
                return;
              }
              void save.mutateAsync();
            }}
            className={`min-h-0 flex-1 px-6 py-5 ${isFixedHeightTab ? 'flex flex-col overflow-hidden' : 'itdb-hidden-scrollbar overflow-y-auto'}`}
          >
            {row?.id && details.isLoading ? (
              <div className="flex h-full min-h-0 flex-1 items-center justify-center text-sm text-[var(--itdb-text-muted)]">
                数据加载中
              </div>
            ) : isHardwareDataTab ? (
              <HardwareDataPane
                fields={resource.fields}
                values={values}
                lookups={lookups.data}
                activeSection={activeHardwareSection}
                onSectionChange={setActiveHardwareSection}
                onChange={update}
                itemId={row?.id}
                onPendingTagsChange={setPendingTags}
                onUnlinkFile={unlinkFileWithCleanup}
              />
            ) : isSoftwareDataTab ? (
              <SoftwareDataPane
                fields={resource.fields}
                values={values}
                lookups={lookups.data}
                activeSection={activeSoftwareSection}
                onSectionChange={setActiveSoftwareSection}
                onChange={update}
                onPendingTagsChange={setPendingTags}
                onUnlinkFile={unlinkFileWithCleanup}
              />
            ) : activeTab.endsWith('Links') ? (
              <HardwareRelationPane
                resourceKey={resource.key}
                tab={activeTab}
                field={visibleFields[0]}
                value={values[visibleFields[0]?.key]}
                lookups={lookups.data}
                onChange={value => visibleFields[0] && update(visibleFields[0], value)}
                currentId={row?.id ? Number(row.id) : undefined}
                currentTypeId={Number(values.typeId)}
                lockNote={
                  resource.key === 'items' && activeTab === 'softwareLinks'
                    ? itemTypeBlocksSoftware(lookups.data, Number(values.itemTypeId))
                    : ''
                }
                stretch={['items', 'software', 'invoices', 'contracts', 'files'].includes(
                  resource.key
                )}
              />
            ) : resource.key === 'items' && activeTab === 'maintenanceInfo' ? (
              <MaintenanceLogPane
                itemId={row?.id}
                exportName={(() => {
                  const source = details.data ?? row ?? {};
                  const model = String(source.model ?? '').trim();
                  const manufacturerId = Number(
                    source.manufacturerid ?? source.manufacturerId ?? 0
                  );
                  const manufacturer = String(
                    (lookups.data?.agents ?? []).find(agent => Number(agent.id) === manufacturerId)
                      ?.title ?? ''
                  ).trim();
                  return `${manufacturer} ${model}`.trim();
                })()}
              />
            ) : resource.key === 'locations' && activeTab === 'main' ? (
              <LocationDataPane
                fields={visibleFields}
                values={values}
                lookups={lookups.data}
                onChange={update}
                locationId={row?.id}
                floorplanName={String((details.data ?? row)?.floorplanfn ?? '')}
                onPendingAreasChange={setLocationPendingAreas}
                onAreasSnapshot={names => {
                  locationAreaNamesRef.current = names;
                }}
              />
            ) : resource.key === 'racks' && activeTab === 'main' ? (
              <RackDataPane
                fields={visibleFields}
                values={values}
                lookups={lookups.data}
                onChange={update}
                rackId={row?.id}
              />
            ) : resource.key === 'invoices' && activeTab === 'main' ? (
              <InvoiceDataPane
                fields={resource.fields}
                values={values}
                lookups={lookups.data}
                activeSection={activeInvoiceSection}
                onSectionChange={setActiveInvoiceSection}
                onChange={update}
                onUnlinkFile={unlinkFileWithCleanup}
              />
            ) : resource.key === 'agents' && activeTab === 'main' ? (
              <AgentDataPane
                fields={resource.fields}
                values={values}
                lookups={lookups.data}
                activeSection={activeAgentSection}
                onSectionChange={setActiveAgentSection}
                onChange={update}
                itemId={row?.id}
              />
            ) : resource.key === 'files' && activeTab === 'main' ? (
              <FileDataPane
                fields={visibleFields}
                values={values}
                lookups={lookups.data}
                onChange={update}
                itemId={row?.id}
                detail={(details.data ?? row) as Row | undefined}
              />
            ) : resource.key === 'contracts' && activeTab === 'main' ? (
              <ContractDataPane
                fields={resource.fields}
                values={values}
                lookups={lookups.data}
                activeSection={activeContractSection}
                onSectionChange={setActiveContractSection}
                onChange={update}
                renewals={renewals}
                onRenewalsChange={setRenewals}
                onUnlinkFile={unlinkFileWithCleanup}
              />
            ) : resource.key === 'contracts' && activeTab === 'events' ? (
              <ContractEventsPane
                contractId={row?.id}
                onPendingEventsChange={setPendingContractEvents}
              />
            ) : (
              <div className="grid gap-x-5 gap-y-4 md:grid-cols-2">
                {visibleFields.map((field, index) => {
                  const section = editorSection(resource.key, field.key, index);
                  return (
                    <Fragment key={field.key}>
                      {section ? (
                        <h3 className="col-span-full border-b border-[var(--itdb-border)] pb-2 pt-3 text-sm font-semibold text-[var(--itdb-text)] first:pt-0">
                          {section}
                        </h3>
                      ) : null}
                      <FieldEditor
                        field={field}
                        value={values[field.key]}
                        options={resolveFieldOptions(field, lookups.data, values, resource.key)}
                        onChange={value => update(field, value)}
                      />
                    </Fragment>
                  );
                })}
              </div>
            )}
          </form>
        </Tabs>
        <DialogFooter className="shrink-0 border-t px-6 py-4">
          <Button type="button" variant="outline" onClick={onClose}>
            取消
          </Button>
          <Button
            form={formId}
            type="submit"
            disabled={save.isPending || details.isLoading}
            className="itdb-action-button disabled:pointer-events-auto disabled:cursor-not-allowed"
            style={{
              borderColor: 'rgba(59,130,246,0.38)',
              background: 'rgba(59,130,246,0.1)',
              color: 'var(--itdb-accent-text)',
            }}
          >
            {save.isPending ? '保存中...' : row?.id ? '保存修改' : '创建'}
          </Button>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  );
}
