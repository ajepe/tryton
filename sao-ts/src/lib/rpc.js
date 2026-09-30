import { encode, decode } from './codec.js'

function orderParams(order) {
  if (!order) return []
  return order
    .split(',')
    .map((item) => item.trim())
    .filter(Boolean)
    .map((item) => {
      const [field, direction] = item.split(/\s+/)
      return [field, (direction || 'ASC').toUpperCase()]
    })
}

export class RPCError extends Error {
  constructor(type, args) {
    super(Array.isArray(args) ? args[0] : String(args))
    this.name = 'RPCError'
    this.type = type
    this.args = args
  }
}

export class RPC {
  constructor({ database = '', onUnauthorized = null } = {}) {
    this.database = database
    this.onUnauthorized = onUnauthorized
    this.cache = new Map()
  }

  url(path) {
    return `/${encodeURIComponent(this.database)}${path}`
  }

  async call(method, params = [], { cache = 0 } = {}) {
    const key = cache ? `${method}:${JSON.stringify(params)}` : null
    if (key && this.cache.has(key)) {
      const entry = this.cache.get(key)
      if (entry.expires > Date.now()) return entry.value
    }

    const response = await fetch(this.url('/rpc/'), {
      method: 'POST',
      credentials: 'include',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ id: 1, method, params: params.map(encode) })
    })

    if (response.status === 401 || response.status === 403) {
      if (this.onUnauthorized) this.onUnauthorized()
      throw new RPCError('LoginException', ['session expired'])
    }

    const payload = await response.json()
    if (payload.error) {
      throw new RPCError(payload.error[0], payload.error[1])
    }
    const result = decode(payload.result)
    if (key && cache > 0) {
      this.cache.set(key, { value: result, expires: Date.now() + cache * 1000 })
    }
    return result
  }

  invalidate() {
    this.cache.clear()
  }

  async listDatabases() {
    return this.call('common.db.list', [])
  }

  async login(database, login, password) {
    this.database = database
    const response = await fetch(this.url('/session/login'), {
      method: 'POST',
      credentials: 'include',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({
        id: 0,
        method: 'common.db.login',
        params: [login, { password }, 'en']
      })
    })
    const payload = await response.json()
    if (payload.error) {
      throw new RPCError(payload.error[0], payload.error[1])
    }
    return decode(payload.result)
  }

  async logout() {
    try {
      await fetch(this.url('/session/logout'), {
        method: 'POST',
        credentials: 'include',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ id: 0, method: 'common.db.logout', params: [] })
      })
    } finally {
      this.cache.clear()
    }
  }

  async searchRead(model, domain, fields, { offset = 0, limit = 80, order = '' } = {}) {
    return this.call(`model.${model}.search_read`, [domain, offset, limit, orderParams(order), fields, {}])
  }

  async searchCount(model, domain) {
    return this.call(`model.${model}.search_count`, [domain, {}])
  }

  async read(model, ids, fields) {
    return this.call(`model.${model}.read`, [ids, fields, {}])
  }

  async create(model, values) {
    const ids = await this.call(`model.${model}.create`, [[values], {}])
    this.invalidate()
    return ids
  }

  async write(model, ids, values) {
    await this.call(`model.${model}.write`, [ids, values, {}])
    this.invalidate()
  }

  async remove(model, ids) {
    await this.call(`model.${model}.delete`, [ids, {}])
    this.invalidate()
  }

  async fieldsGet(model, context = {}) {
    return this.call(`model.${model}.fields_get`, [[], { language: 'en', ...context }], { cache: 300 })
  }

  async onChanged(model, values, fields) {
    return this.call(`model.${model}.on_change`, [values, fields, {}])
  }

  async getActionValue(actionId) {
    return this.call('model.ir.action.get_action_value', [actionId, {}])
  }

  async executeButton(model, action, ids) {
    const result = await this.call(`model.${model}.${action}`, [ids, {}])
    this.invalidate()
    return result
  }
}
