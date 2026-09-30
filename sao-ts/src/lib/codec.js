const CLASS_KEY = '__class__'

export function encode(value) {
  if (value === null || value === undefined) return value
  if (value instanceof Date) {
    if (value.getHours() === 0 && value.getMinutes() === 0 && value.getSeconds() === 0 &&
        value.getTime() === startOfDay(value)) {
      return {
        [CLASS_KEY]: 'date',
        year: value.getFullYear(),
        month: value.getMonth() + 1,
        day: value.getDate()
      }
    }
    return {
      [CLASS_KEY]: 'datetime',
      year: value.getFullYear(),
      month: value.getMonth() + 1,
      day: value.getDate(),
      hour: value.getHours(),
      minute: value.getMinutes(),
      second: value.getSeconds()
    }
  }
  if (typeof value === 'bigint') {
    return { [CLASS_KEY]: 'Decimal', decimal: value.toString() }
  }
  if (value instanceof Uint8Array) {
    return { [CLASS_KEY]: 'bytes', base64: base64Encode(value) }
  }
  if (Array.isArray(value)) {
    return value.map(encode)
  }
  if (typeof value === 'object') {
    const result = {}
    for (const [key, item] of Object.entries(value)) {
      result[key] = encode(item)
    }
    return result
  }
  return value
}

export function decode(value) {
  if (value === null || value === undefined) return value
  if (Array.isArray(value)) {
    return value.map(decode)
  }
  if (typeof value !== 'object') return value

  const className = value[CLASS_KEY]
  switch (className) {
    case 'datetime':
      return new Date(
        value.year, value.month - 1, value.day,
        value.hour || 0, value.minute || 0, value.second || 0
      )
    case 'date':
      return new Date(value.year, value.month - 1, value.day)
    case 'time': {
      const date = new Date(1970, 0, 1, value.hour || 0, value.minute || 0, value.second || 0)
      return date
    }
    case 'timedelta':
      return value.seconds * 1000
    case 'Decimal':
      return Number(value.decimal)
    case 'bytes':
      return base64Decode(value.base64)
    default: {
      const result = {}
      for (const [key, item] of Object.entries(value)) {
        result[key] = decode(item)
      }
      return result
    }
  }
}

function startOfDay(date) {
  return new Date(date.getFullYear(), date.getMonth(), date.getDate()).getTime()
}

export function formatDate(value, language) {
  if (!(value instanceof Date)) return value
  const pad = (n) => String(n).padStart(2, '0')
  return `${value.getFullYear()}-${pad(value.getMonth() + 1)}-${pad(value.getDate())}`
}

export function formatDateTime(value) {
  if (!(value instanceof Date)) return value
  return `${formatDate(value)} ${String(value.getHours()).padStart(2, '0')}:${String(value.getMinutes()).padStart(2, '0')}:${String(value.getSeconds()).padStart(2, '0')}`
}

function base64Encode(bytes) {
  let binary = ''
  for (const byte of bytes) binary += String.fromCharCode(byte)
  return btoa(binary)
}

function base64Decode(text) {
  const binary = atob(text)
  const bytes = new Uint8Array(binary.length)
  for (let i = 0; i < binary.length; i++) bytes[i] = binary.charCodeAt(i)
  return bytes
}
