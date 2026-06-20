export function normalizeBackupItems(items) {
  if (!Array.isArray(items)) {
    return []
  }

  return items.map((item) => ({
    ...item,
    icon_base64: item?.icon_base64 || '',
  }))
}
