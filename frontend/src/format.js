export const MAX_ONGOING = 3

export const fmt = (iso) => {
  if (!iso) return '--/--/--'
  const x = new Date(iso)
  return `${x.getMonth() + 1}/${x.getDate()}/${String(x.getFullYear()).slice(2)}`
}

export const fmtTime = (iso) =>
  iso ? new Date(iso).toLocaleString([], { month: 'short', day: 'numeric', year: 'numeric', hour: 'numeric', minute: '2-digit' }) : '—'
