import { Printer } from 'lucide-react';
import { type Ref } from 'react';
import { Button } from '@/components/ui/button';
import { paperSizeMM, type LabelConfig } from './labels-config';
import {
  buildLabelBodyLines,
  headerTextLines,
  type LabelCell,
  type LabelRow,
} from './labels-print';

const PREVIEW_PX = 3.2;

type PreviewPanelProps = {
  ref: Ref<HTMLDivElement>;
  rows: LabelRow[];
  qrImages: Map<number, string>;
  config: LabelConfig;
  onPrint?: () => void;
  onClear: () => void;
};

/* 预览面板：按纸张与布局配置逐页渲染标签（mm→px 按 3.2 换算，纸张过大时整体缩放） */
export function PreviewPanel(props: PreviewPanelProps) {
  const paper = paperSizeMM(props.config.papersize);
  const scalePx = PREVIEW_PX;
  const sheetW = paper.w * scalePx;
  const scale = Math.min(1, 900 / sheetW);
  const labelW = Math.max(10, Number(props.config.lwidth) || 66) * scalePx;
  const labelH = Math.max(10, Number(props.config.lheight) || 35) * scalePx;
  const hpitch = Math.max(labelW, (Number(props.config.hpitch) || 0) * scalePx);
  const vpitch = Math.max(labelH, (Number(props.config.vpitch) || 0) * scalePx);
  const pages = computePreviewPages(props.rows, props.config);
  const borderGray = Math.min(255, Math.max(0, Number(props.config.border) || 200));
  const qrSize = Math.min(160, Math.max(36, (Number(props.config.barcodesize) || 20) * scalePx));
  return (
    <section
      ref={props.ref}
      className="itdb-card-hover flex flex-col rounded-xl p-5"
      style={{
        background: 'var(--itdb-card)',
        border: '1px solid var(--itdb-border)',
        boxShadow: 'var(--shadow-card)',
      }}
    >
      <div className="flex shrink-0 flex-wrap items-center justify-between gap-2 border-b border-[var(--itdb-border)] pb-3">
        <h3 className="text-base font-semibold text-[var(--itdb-text)]">
          预览结果（{props.rows.length}）
        </h3>
        <div className="flex flex-wrap items-center gap-2 text-xs text-[var(--itdb-text-muted)]">
          <span className="rounded-full bg-[rgba(59,130,246,0.12)] px-2.5 py-1">
            {props.config.papersize} {paper.w}×{paper.h} mm
          </span>
          <span className="rounded-full bg-[rgba(59,130,246,0.12)] px-2.5 py-1">
            {props.config.cols} 列 × {props.config.rows} 行
          </span>
          <span className="rounded-full bg-[rgba(59,130,246,0.12)] px-2.5 py-1">
            跳过 {Number(props.config.labelskip) || 0} 个
          </span>
          <span className="rounded-full bg-[rgba(59,130,246,0.12)] px-2.5 py-1">
            水平间距 {Number(props.config.hpitch) || 0}mm / 垂直间距{' '}
            {Number(props.config.vpitch) || 0}mm
          </span>
          {String(props.config.name ?? '').trim() ? (
            <span className="rounded-full bg-[rgba(59,130,246,0.12)] px-2.5 py-1">
              {String(props.config.name).trim()}
            </span>
          ) : null}
          {props.onPrint ? (
            <Button size="sm" className="itdb-action-button" onClick={props.onPrint}>
              <Printer size={14} />
              打印 / 导出 PDF
            </Button>
          ) : null}
          <Button
            size="sm"
            variant="outline"
            className="itdb-action-button"
            onClick={props.onClear}
          >
            清除预览
          </Button>
        </div>
      </div>
      {props.rows.length === 0 ? (
        <p className="grid min-h-48 place-items-center text-sm text-[var(--itdb-text-muted)]">
          勾选硬件后点击生成标签预览
        </p>
      ) : (
        <div className="mt-3 space-y-4">
          {pages.map((cells, pageIndex) => (
            <div
              key={pageIndex}
              className="rounded-xl border border-[var(--itdb-border)] bg-[var(--itdb-card)] p-4"
              style={{ boxShadow: 'var(--shadow-card)' }}
            >
              <p className="mb-2 text-xs font-bold text-[var(--itdb-accent-text)]">
                第 {pageIndex + 1} 页
              </p>
              <div className="overflow-x-auto">
                <div
                  className="itdb-label-sheet-canvas mx-auto w-fit"
                  style={{
                    width: sheetW * scale,
                    height: paper.h * scalePx * scale,
                    overflow: 'hidden',
                  }}
                >
                  <div
                    style={{
                      width: sheetW,
                      transform: `scale(${scale})`,
                      transformOrigin: 'top left',
                      paddingTop: (Number(props.config.tmargin) || 0) * scalePx,
                      paddingRight: (Number(props.config.rmargin) || 0) * scalePx,
                      paddingBottom: (Number(props.config.bmargin) || 0) * scalePx,
                      paddingLeft: (Number(props.config.lmargin) || 0) * scalePx,
                      display: 'grid',
                      gridTemplateColumns: `repeat(${props.config.cols}, ${labelW}px)`,
                      columnGap: Math.max(0, hpitch - labelW),
                      rowGap: Math.max(0, vpitch - labelH),
                    }}
                  >
                    {cells.map((cell, cellIndex) => (
                      <PreviewCell
                        key={cellIndex}
                        cell={cell}
                        config={props.config}
                        qrImages={props.qrImages}
                        borderGray={borderGray}
                        labelH={labelH}
                        qrSize={qrSize}
                      />
                    ))}
                  </div>
                </div>
              </div>
            </div>
          ))}
        </div>
      )}
    </section>
  );
}

function computePreviewPages(rows: LabelRow[], config: LabelConfig): LabelCell[][] {
  const capacity = Math.max(1, config.cols * config.rows);
  const skip = Math.max(0, Math.min(capacity - 1, Number(config.labelskip) || 0));
  const cells: LabelCell[] = [
    ...Array.from({ length: skip }, () => ({ kind: 'skip' }) as LabelCell),
    ...rows.map(row => ({ kind: 'label', row }) as LabelCell),
  ];
  const pages: LabelCell[][] = [];
  for (let index = 0; index < cells.length; index += capacity) {
    const chunk = cells.slice(index, index + capacity);
    while (chunk.length < capacity) chunk.push({ kind: 'blank' });
    pages.push(chunk);
  }
  return pages.length > 0 ? pages : [[]];
}

function PreviewCell({
  cell,
  config,
  qrImages,
  borderGray,
  labelH,
  qrSize,
}: {
  cell: LabelCell;
  config: LabelConfig;
  qrImages: Map<number, string>;
  borderGray: number;
  labelH: number;
  qrSize: number;
}) {
  if (cell.kind === 'skip') {
    return (
      <div className="itdb-label-cell-skip text-xs" style={{ height: labelH }}>
        跳过
      </div>
    );
  }
  if (cell.kind === 'blank')
    return <div className="itdb-label-cell-blank" style={{ height: labelH }} />;
  const lines = buildLabelBodyLines(cell.row, config.wantnotext);
  const qr = qrImages.get(cell.row.id);
  const rightText = config.wantbarcode && config.wantraligntext;
  const body = (
    <div
      className={`min-h-0 flex-1 ${
        rightText
          ? 'grid grid-cols-[auto_1fr] items-start gap-2'
          : config.wantnotext
            ? 'flex flex-col items-center justify-center gap-2'
            : 'flex flex-col items-start gap-2'
      }`}
    >
      {config.wantbarcode ? (
        <div
          className="flex flex-col items-center gap-1"
          style={{ width: qrSize, flex: '0 0 auto' }}
        >
          <div
            className="grid place-items-center rounded-[4px] border border-dashed border-[#274763] bg-white"
            style={{ width: qrSize, height: qrSize, padding: 3 }}
          >
            {qr ? (
              <img src={qr} alt="QR" style={{ width: '100%', height: '100%' }} />
            ) : (
              <span className="text-[10px] text-[#6b7280]">QR</span>
            )}
          </div>
          {config.wantnotext ? null : (
            <span className="max-w-full text-center text-[10px] leading-tight text-[#374151] [overflow-wrap:anywhere]">
              {config.qrtext.trim() ? cell.row.qrText : cell.row.id}
            </span>
          )}
        </div>
      ) : null}
      {config.wantnotext ? null : (
        <div className="flex min-w-0 flex-1 flex-col">
          {lines.map((line, index) => (
            <span
              key={index}
              className="truncate text-[#1f2937]"
              style={{
                fontSize:
                  index === 0 ? Number(config.idfontsize) * 1.45 : Number(config.fontsize) * 1.45,
                fontWeight: index === 0 ? 700 : 400,
                lineHeight: 1.35,
              }}
            >
              {line}
            </span>
          ))}
        </div>
      )}
    </div>
  );
  return (
    <div
      className="flex flex-col overflow-hidden rounded-[4px] bg-white"
      style={{
        height: labelH,
        border: `0.6px solid rgb(${borderGray},${borderGray},${borderGray})`,
        padding: (Number(config.padding) || 0) * PREVIEW_PX,
      }}
    >
      {config.wantheaderimage || config.wantheadertext ? (
        <div className="mb-1 flex items-start gap-1.5">
          {config.wantheaderimage ? (
            <img
              key={
                /^(\/|data:|https?:)/.test(config.image.trim())
                  ? config.image.trim()
                  : `/${config.image.trim()}`
              }
              src={
                /^(\/|data:|https?:)/.test(config.image.trim())
                  ? config.image.trim()
                  : `/${config.image.trim()}`
              }
              alt=""
              className="shrink-0 object-contain"
              style={{
                height: (Number(config.imageheight) || 5) * PREVIEW_PX,
                width: (Number(config.imagewidth) || 5) * PREVIEW_PX,
              }}
              onError={event => {
                (event.target as HTMLImageElement).style.display = 'none';
              }}
            />
          ) : null}
          {config.wantheadertext ? (
            <div className="flex min-w-0 flex-col">
              {headerTextLines(config.headertext).map((line, index) => (
                <span
                  key={index}
                  className="font-bold leading-snug"
                  style={{ color: '#004664', fontSize: Number(config.headerfontsize) * 1.45 }}
                >
                  {line}
                </span>
              ))}
            </div>
          ) : null}
        </div>
      ) : null}
      {body}
    </div>
  );
}
