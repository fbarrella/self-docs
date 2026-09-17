export interface AvatarProps {
  initials: string
  size?: 'sm' | 'md'
  label?: string
}

/** Avatar shows a user's initials in a circular accent badge. */
export function Avatar({ initials, size = 'md', label }: AvatarProps) {
  const classes = ['ui-avatar', size === 'sm' ? 'ui-avatar--sm' : ''].filter(Boolean).join(' ')
  return (
    <span className={classes} role="img" aria-label={label ?? initials}>
      {initials}
    </span>
  )
}
