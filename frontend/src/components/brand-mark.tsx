type BrandIconProps = {
  iconData: string;
  alt: string;
  className?: string;
};

// BrandIcon 按品牌配置渲染图标，默认与自定义图标统一以 favicon.svg 图片展示
export function BrandIcon({ iconData, alt, className }: BrandIconProps) {
  return <img src={iconData} alt={alt} className={className} />;
}
