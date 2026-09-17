export interface SkeletonProps {
  width?: string | number
  height?: string | number
  radius?: string
  className?: string
}

/** Skeleton is a shimmering placeholder shown while content loads. */
export function Skeleton({ width = '100%', height = '1rem', radius, className }: SkeletonProps) {
  return (
    <span
      className={['ui-skeleton', className ?? ''].filter(Boolean).join(' ')}
      style={{ display: 'block', width, height, borderRadius: radius }}
      aria-hidden="true"
    />
  )
}
