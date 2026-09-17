import QRCode from 'qrcode';
import type { LabelConfig } from './labels-config';

export type LabelRow = {
  id: number;
  itemType: string;
  manufacturer: string;
  model: string;
  sn: string;
  label: string;
  dnsName: string;
  ipv4: string;
  ipv6: string;
  text: string;
  headerText: string;
  qrText: string;
};

export type LabelCell = { kind: 'label'; row: LabelRow } | { kind: 'skip' } | { kind: 'blank' };

const qrCache = new Map<string, string>();

/* 生成 QR 二维码的 SVG Data URL（按内容缓存） */
export async function qrImageDataUrl(text: string): Promise<string> {
  const cached = qrCache.get(text);
  if (cached) return cached;
  const svg = await QRCode.toString(text, {
    type: 'svg',
    margin: 0,
    errorCorrectionLevel: 'M',
    color: { dark: '#102a43', light: '#ffffff' },
  });
  const url = `data:image/svg+xml;charset=utf-8,${encodeURIComponent(svg)}`;
  qrCache.set(text, url);
  return url;
}

const trunc = (value: string, max: number) =>
  Array.from(value).length > max ? `${Array.from(value).slice(0, max).join('')}…` : value;

/* 标签文本行：ID/标签/SN/厂商型号/IP/DNS，有值才输出（与原 itdb 口径一致） */
export function buildLabelBodyLines(row: LabelRow, hideId: boolean): string[] {
  const lines: string[] = [];
  if (!hideId) lines.push(`ID:${String(row.id).padStart(4, '0')}`);
  if (row.label.trim()) lines.push(`LBL:${row.label.trim()}`);
  if (row.sn.trim()) lines.push(`SN:${row.sn.trim()}`);
  const maker = `${row.manufacturer.trim()}/${row.model.trim()}`;
  if (maker.replace(/[/\s]/g, '')) lines.push(trunc(maker, 37));
  if (row.ipv4.trim()) lines.push(`IPv4:${trunc(row.ipv4.trim(), 15)}`);
  if (row.ipv6.trim()) lines.push(`IPv6:${row.ipv6.trim()}`);
  if (row.dnsName.trim()) lines.push(`HName:${row.dnsName.trim()}`);
  return lines;
}

export function headerTextLines(headertext: string): string[] {
  return headertext
    .replace(/_NL_/g, '\n')
    .split('\n')
    .map(line => line.trim())
    .filter(Boolean);
}

function escapeHtml(value: string) {
  return value
    .replace(/&/g, '&amp;')
    .replace(/</g, '&lt;')
    .replace(/>/g, '&gt;')
    .replace(/"/g, '&quot;');
}

function resolveImagePath(image: string) {
  const raw = image.trim();
  if (!raw) return '';
  if (/^(\/|data:|https?:)/.test(raw)) return raw;
  return `/${raw}`;
}

function buildLabelCellHtml(cell: LabelCell, qrImages: Map<number, string>, config: LabelConfig) {
  if (cell.kind === 'blank') {
    return '<div class="cell blank-cell"></div>';
  }
  if (cell.kind === 'skip') {
    return '<div class="cell skip-cell">跳过</div>';
  }
  const qr = config.wantbarcode
    ? `<div class="label-qr"><div class="label-qr-box">${
        qrImages.get(cell.row.id)
          ? `<img src="${qrImages.get(cell.row.id)}" alt="QR"/>`
          : '<span class="label-qr-placeholder">QR</span>'
      }</div></div>`
    : '';
  const lines = buildLabelBodyLines(cell.row, config.wantnotext);
  const body = config.wantnotext
    ? ''
    : `<div class="label-lines">${lines
        .map(line => `<div class="label-line">${escapeHtml(line)}</div>`)
        .join('')}</div>`;
  const headerParts: string[] = [];
  if (config.wantheaderimage) {
    const src = resolveImagePath(config.image);
    if (src)
      headerParts.push(
        `<img class="label-header-image" src="${escapeHtml(src)}" onerror="this.style.display='none'"/>`
      );
  }
  const headerLines = config.wantheadertext
    ? headerTextLines(config.headertext)
        .map(line => `<div class="label-header-line">${escapeHtml(line)}</div>`)
        .join('')
    : '';
  if (headerLines) headerParts.push(`<div class="label-header-text">${headerLines}</div>`);
  const header = headerParts.length
    ? `<div class="label-header">${headerParts.join('')}</div>`
    : '';
  const bodyClass =
    config.wantbarcode && config.wantraligntext
      ? 'label-body is-split'
      : config.wantnotext
        ? 'label-body is-center'
        : 'label-body';
  return `<div class="cell label-cell">${header}<div class="${bodyClass}">${qr}${body}</div></div>`;
}

/* 构建打印弹窗的完整 HTML 文档：物理单位排版，加载完成后自动调起打印 */
export function buildPrintDocumentHtml(
  pages: LabelCell[][],
  qrImages: Map<number, string>,
  config: LabelConfig,
  paper: { w: number; h: number }
) {
  const borderGray = Math.min(255, Math.max(0, Number(config.border) || 200));
  const paddingMM = Math.min(8, Math.max(0, Number(config.padding) || 0));
  const bodyFontSize = Math.min(18, Math.max(4, Number(config.fontsize) || 6));
  const idFontSize = Math.min(20, Math.max(4, Number(config.idfontsize) || 7));
  const headerFontSize = Math.min(18, Math.max(4, Number(config.headerfontsize) || 6));
  const qrMM = Math.min(40, Math.max(8, Number(config.barcodesize) || 20));
  const imageW = Math.min(40, Math.max(0, Number(config.imagewidth) || 5));
  const imageH = Math.min(40, Math.max(0, Number(config.imageheight) || 5));
  const labelW = Math.max(10, Number(config.lwidth) || 66);
  const labelH = Math.max(10, Number(config.lheight) || 35);
  const hpitch = Math.max(labelW, Number(config.hpitch) || labelW);
  const vpitch = Math.max(labelH, Number(config.vpitch) || labelH);
  const gapX = hpitch - labelW;
  const gapY = vpitch - labelH;
  const sheets = pages
    .map(
      cells =>
        `<section class="sheet"><div class="grid">${cells
          .map(cell => buildLabelCellHtml(cell, qrImages, config))
          .join('')}</div></section>`
    )
    .join('');
  return `<!doctype html>
<html lang="zh-CN">
<head>
<meta charset="utf-8"/>
<title>打印标签</title>
<style>
  @page { size: ${paper.w}mm ${paper.h}mm; margin: 0; }
  * { box-sizing: border-box; }
  body { margin: 0; font-family: "Microsoft YaHei", "PingFang SC", sans-serif; -webkit-print-color-adjust: exact; print-color-adjust: exact; }
  .sheet { width: ${paper.w}mm; min-height: ${paper.h}mm; padding: ${config.tmargin}mm ${config.rmargin}mm ${config.bmargin}mm ${config.lmargin}mm; break-after: page; page-break-after: always; }
  .sheet:last-of-type { break-after: auto; page-break-after: auto; }
  .grid { display: grid; grid-template-columns: repeat(${config.cols}, ${labelW}mm); column-gap: ${gapX}mm; row-gap: ${gapY}mm; }
  .cell { height: ${labelH}mm; }
  .label-cell { display: flex; flex-direction: column; border: 0.2mm solid rgb(${borderGray}, ${borderGray}, ${borderGray}); padding: ${paddingMM}mm; overflow: hidden; background: #fff; }
  .skip-cell { display: grid; place-items: center; border: 0.2mm dashed #b6c2d4; color: #8a97ab; font-size: ${bodyFontSize}pt; }
  .blank-cell { border: none; visibility: hidden; }
  .label-header { display: flex; align-items: flex-start; gap: 1.8mm; margin-bottom: 1mm; }
  .label-header-image { flex: 0 0 auto; width: ${imageW}mm; height: ${imageH}mm; object-fit: contain; }
  .label-header-text { min-width: 0; display: flex; flex-direction: column; gap: 0.3mm; }
  .label-header-line { color: #004664; font-weight: 700; font-size: ${headerFontSize}pt; line-height: 1.3; }
  .label-body { display: flex; flex: 1; min-height: 0; flex-direction: column; align-items: flex-start; gap: 1.2mm; }
  .label-body.is-split { display: grid; grid-template-columns: auto 1fr; column-gap: 1.5mm; align-items: start; }
  .label-body.is-center { align-items: center; justify-content: center; }
  .label-qr { display: flex; flex-direction: column; align-items: center; gap: 0.5mm; }
  .label-qr-box { width: ${qrMM}mm; height: ${qrMM}mm; border: 0.2mm dashed #274763; border-radius: 1mm; background: #fff; padding: 1mm; display: grid; place-items: center; }
  .label-qr-box img { width: 100%; height: 100%; }
  .label-qr-placeholder { color: #8a97ab; font-size: ${bodyFontSize}pt; }
  .label-lines { display: flex; flex-direction: column; min-width: 0; flex: 1; }
  .label-body.is-split .label-lines { flex: none; }
  .label-line { font-size: ${bodyFontSize}pt; line-height: 1.35; white-space: nowrap; overflow: hidden; text-overflow: ellipsis; }
  .label-line:first-child { font-size: ${idFontSize}pt; font-weight: 700; }
  @media screen { body { background: #46536a; } .sheet { margin: 6mm auto; box-shadow: 0 4px 18px rgba(0,0,0,0.4); background: #fff; } }
</style>
</head>
<body>
${sheets}
<script>
  window.addEventListener('load', function () {
    var images = Array.prototype.slice.call(document.images);
    Promise.all(images.map(function (img) {
      return img.complete ? Promise.resolve() : new Promise(function (done) { img.onload = img.onerror = done; });
    })).then(function () { setTimeout(function () { window.print(); }, 120); });
  });
</script>
</body>
</html>`;
}
