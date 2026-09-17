export type LabelConfig = {
  name: string;
  papersize: string;
  rows: number;
  cols: number;
  lwidth: string;
  lheight: string;
  vpitch: string;
  hpitch: string;
  tmargin: string;
  bmargin: string;
  lmargin: string;
  rmargin: string;
  border: string;
  padding: string;
  fontsize: string;
  idfontsize: string;
  headerfontsize: string;
  barcodesize: string;
  image: string;
  imagewidth: string;
  imageheight: string;
  headertext: string;
  qrtext: string;
  wantbarcode: boolean;
  wantheadertext: boolean;
  wantheaderimage: boolean;
  wantnotext: boolean;
  wantraligntext: boolean;
  labelskip: string;
};

export const DEFAULT_LABEL_CONFIG: LabelConfig = {
  name: '',
  papersize: 'A4',
  rows: 8,
  cols: 3,
  lwidth: '66',
  lheight: '35',
  vpitch: '35',
  hpitch: '70',
  tmargin: '12',
  bmargin: '12',
  lmargin: '6',
  rmargin: '6',
  border: '200',
  padding: '1',
  fontsize: '6',
  idfontsize: '7',
  headerfontsize: '6',
  barcodesize: '20',
  image: 'images/itdb.png',
  imagewidth: '5',
  imageheight: '5',
  headertext: 'IT资产标签',
  qrtext: '',
  wantbarcode: true,
  wantheadertext: true,
  wantheaderimage: false,
  wantnotext: false,
  wantraligntext: false,
  labelskip: '0',
};

/* 内置常用纸张（mm），未知纸张名回退 A4 */
export const PAPER_SIZES: Record<string, { w: number; h: number }> = {
  A3: { w: 297, h: 420 },
  A4: { w: 210, h: 297 },
  A5: { w: 148, h: 210 },
  A6: { w: 105, h: 148 },
  B4: { w: 257, h: 364 },
  B5: { w: 182, h: 257 },
  B6: { w: 128, h: 182 },
  LETTER: { w: 215.9, h: 279.4 },
  LEGAL: { w: 215.9, h: 355.6 },
  EXECUTIVE: { w: 184.2, h: 266.7 },
  TABLOID: { w: 279.4, h: 431.8 },
};

export function paperSizeMM(name: string): { w: number; h: number } {
  const key = name.trim().toUpperCase();
  if (PAPER_SIZES[key]) return PAPER_SIZES[key];
  const iso = /^A([0-9]|10)$/.exec(key);
  if (iso) {
    const index = Number(iso[1]);
    let w = 841;
    let h = 1189;
    for (let step = 0; step <= index; step++) {
      const next = Math.floor(w / 2);
      w = h;
      h = next;
    }
    return { w, h };
  }
  return PAPER_SIZES.A4;
}

/* 后端预设行的 0/1 布尔 → 前端布尔 */
function flag(value: unknown) {
  return Number(value ?? 0) === 1;
}

export function labelConfigFromPreset(row: Record<string, unknown>): LabelConfig {
  const text = (key: string) => String(row[key] ?? '').trim();
  return {
    name: text('name'),
    papersize: text('papersize') || DEFAULT_LABEL_CONFIG.papersize,
    rows: Number(row.rows ?? DEFAULT_LABEL_CONFIG.rows) || DEFAULT_LABEL_CONFIG.rows,
    cols: Number(row.cols ?? DEFAULT_LABEL_CONFIG.cols) || DEFAULT_LABEL_CONFIG.cols,
    lwidth: text('lwidth') || DEFAULT_LABEL_CONFIG.lwidth,
    lheight: text('lheight') || DEFAULT_LABEL_CONFIG.lheight,
    vpitch: text('vpitch') || DEFAULT_LABEL_CONFIG.vpitch,
    hpitch: text('hpitch') || DEFAULT_LABEL_CONFIG.hpitch,
    tmargin: text('tmargin') || DEFAULT_LABEL_CONFIG.tmargin,
    bmargin: text('bmargin') || DEFAULT_LABEL_CONFIG.bmargin,
    lmargin: text('lmargin') || DEFAULT_LABEL_CONFIG.lmargin,
    rmargin: text('rmargin') || DEFAULT_LABEL_CONFIG.rmargin,
    border: text('border') || DEFAULT_LABEL_CONFIG.border,
    padding: text('padding') || DEFAULT_LABEL_CONFIG.padding,
    fontsize: text('fontsize') || DEFAULT_LABEL_CONFIG.fontsize,
    idfontsize: text('idfontsize') || DEFAULT_LABEL_CONFIG.idfontsize,
    headerfontsize: text('headerfontsize') || DEFAULT_LABEL_CONFIG.headerfontsize,
    barcodesize: text('barcodesize') || DEFAULT_LABEL_CONFIG.barcodesize,
    image: text('image') || DEFAULT_LABEL_CONFIG.image,
    imagewidth: text('imagewidth') || DEFAULT_LABEL_CONFIG.imagewidth,
    imageheight: text('imageheight') || DEFAULT_LABEL_CONFIG.imageheight,
    headertext: text('headertext').replace(/_NL_/g, '\n'),
    qrtext: text('qrtext'),
    wantbarcode: flag(row.wantbarcode),
    wantheadertext: flag(row.wantheadertext),
    wantheaderimage: flag(row.wantheaderimage),
    wantnotext: flag(row.wantnotext),
    wantraligntext: flag(row.wantraligntext),
    labelskip: text('labelskip') || DEFAULT_LABEL_CONFIG.labelskip,
  };
}

/* 保存预设时换行统一转回 _NL_，与既有 labelpapers 数据格式保持兼容 */
export function labelPresetPayload(config: LabelConfig) {
  return {
    name: config.name,
    papersize: config.papersize,
    rows: config.rows,
    cols: config.cols,
    lwidth: config.lwidth,
    lheight: config.lheight,
    vpitch: config.vpitch,
    hpitch: config.hpitch,
    tmargin: config.tmargin,
    bmargin: config.bmargin,
    lmargin: config.lmargin,
    rmargin: config.rmargin,
    border: config.border,
    padding: config.padding,
    fontsize: config.fontsize,
    headerfontsize: config.headerfontsize,
    barcodesize: config.barcodesize,
    idfontsize: config.idfontsize,
    wantbarcode: config.wantbarcode ? 1 : 0,
    wantheadertext: config.wantheadertext ? 1 : 0,
    wantheaderimage: config.wantheaderimage ? 1 : 0,
    headertext: config.headertext.replace(/\r?\n/g, '_NL_'),
    image: config.image,
    imagewidth: config.imagewidth,
    imageheight: config.imageheight,
    qrtext: config.qrtext,
    wantnotext: config.wantnotext ? 1 : 0,
    wantraligntext: config.wantraligntext ? 1 : 0,
    labelskip: config.labelskip,
  };
}
