import { useId } from 'react';
import { defaultBrandSettings } from '@/lib/branding';

type BrandMarkProps = {
  className?: string;
  label?: string;
};

/**
 * BrandMark 以内联 SVG 渲染资产机架图标
 * 底座、描边等配色通过 --itdb-logo-* CSS 变量取值，随 data-itdb-theme 在深浅色主题间自动切换
 */
export function BrandMark({ className, label }: BrandMarkProps) {
  const uid = useId().replace(/:/g, '');
  const tileId = `itdb-logo-tile-${uid}`;
  const frameId = `itdb-logo-frame-${uid}`;
  const activeId = `itdb-logo-active-${uid}`;
  const glowBlueId = `itdb-logo-glow-blue-${uid}`;
  const glowCyanId = `itdb-logo-glow-cyan-${uid}`;

  return (
    <svg
      viewBox="0 0 512 512"
      className={className}
      role={label ? 'img' : undefined}
      aria-label={label}
      aria-hidden={label ? undefined : true}
      focusable="false"
    >
      <defs>
        <linearGradient id={tileId} x1="0" y1="0" x2="512" y2="512" gradientUnits="userSpaceOnUse">
          <stop offset="0" style={{ stopColor: 'var(--itdb-logo-tile-a)' }} />
          <stop offset="0.55" style={{ stopColor: 'var(--itdb-logo-tile-b)' }} />
          <stop offset="1" style={{ stopColor: 'var(--itdb-logo-tile-c)' }} />
        </linearGradient>
        <linearGradient
          id={frameId}
          x1="131"
          y1="111"
          x2="381"
          y2="401"
          gradientUnits="userSpaceOnUse"
        >
          <stop offset="0" style={{ stopColor: 'var(--itdb-logo-frame-a)' }} />
          <stop offset="1" style={{ stopColor: 'var(--itdb-logo-frame-b)' }} />
        </linearGradient>
        <linearGradient
          id={activeId}
          x1="163"
          y1="319"
          x2="349"
          y2="375"
          gradientUnits="userSpaceOnUse"
        >
          <stop offset="0" stopColor="#3b82f6" />
          <stop offset="1" stopColor="#06b6d4" />
        </linearGradient>
        <radialGradient id={glowBlueId} cx="0.16" cy="0.06" r="0.8">
          <stop
            offset="0"
            stopColor="#3b82f6"
            style={{ stopOpacity: 'var(--itdb-logo-glow-blue)' }}
          />
          <stop offset="1" stopColor="#3b82f6" stopOpacity="0" />
        </radialGradient>
        <radialGradient id={glowCyanId} cx="0.88" cy="0.96" r="0.75">
          <stop
            offset="0"
            stopColor="#06b6d4"
            style={{ stopOpacity: 'var(--itdb-logo-glow-cyan)' }}
          />
          <stop offset="1" stopColor="#06b6d4" stopOpacity="0" />
        </radialGradient>
      </defs>

      <rect width="512" height="512" rx="112" fill={`url(#${tileId})`} />
      <rect width="512" height="512" rx="112" fill={`url(#${glowBlueId})`} />
      <rect width="512" height="512" rx="112" fill={`url(#${glowCyanId})`} />
      <rect
        x="1.5"
        y="1.5"
        width="509"
        height="509"
        rx="110.5"
        fill="none"
        style={{ stroke: 'var(--itdb-logo-ring)' }}
        strokeWidth="3"
      />
      <g fill="none" strokeLinecap="round" strokeLinejoin="round">
        <rect
          x="131"
          y="111"
          width="250"
          height="290"
          rx="44"
          stroke={`url(#${frameId})`}
          strokeWidth="17"
        />
        <rect
          x="163"
          y="155"
          width="186"
          height="56"
          rx="16"
          style={{ stroke: 'var(--itdb-logo-dim)' }}
          strokeWidth="11"
        />
        <rect
          x="163"
          y="237"
          width="186"
          height="56"
          rx="16"
          style={{ stroke: 'var(--itdb-logo-dim)' }}
          strokeWidth="11"
        />
        <path
          d="M181 183h44M181 265h44"
          style={{ stroke: 'var(--itdb-logo-dim-line)' }}
          strokeWidth="10"
        />
        <rect x="163" y="319" width="186" height="56" rx="16" fill={`url(#${activeId})`} />
        <path d="M181 347h44" stroke="rgba(240,250,255,0.92)" strokeWidth="10" />
        <circle cx="317" cy="347" r="9.5" fill="#071021" />
      </g>
    </svg>
  );
}

type BrandIconProps = {
  iconData: string;
  alt: string;
  className?: string;
};

/**
 * BrandIcon 按品牌配置渲染图标
 * 默认图标走内联 BrandMark 以获得主题自适应配色，管理员上传的自定义图标回退为 img
 */
export function BrandIcon({ iconData, alt, className }: BrandIconProps) {
  if (iconData === defaultBrandSettings.iconData) {
    return <BrandMark className={className} label={alt} />;
  }
  return <img src={iconData} alt={alt} className={className} />;
}
