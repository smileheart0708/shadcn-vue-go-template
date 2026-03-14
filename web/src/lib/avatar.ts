export function getAvatarFallbackText(name: string | null | undefined): string {
  const normalizedName = name?.trim()

  if (!normalizedName) {
    return '??'
  }

  return Array.from(normalizedName).slice(0, 2).join('').toUpperCase()
}

export const SUPPORTED_AVATAR_FILE_TYPES = ['image/jpeg', 'image/png', 'image/webp'] as const

export const MAX_AVATAR_FILE_SIZE_BYTES = 5 * 1024 * 1024
