<script>
  import { rpc, showToast } from '../lib/store.js'

  let { model, onOpenRecord } = $props()

  let fields = $state({})
  let columns = $state([])
  let records = $state([])
  let total = $state(0)
  let page = $state(0)
  let pageSize = 40
  let search = $state('')
  let loading = $state(true)

  let preferred = ['name', 'code', 'number', 'state', 'date', 'amount', 'active', 'rec_name']

  async function load() {
    if (!model) return
    loading = true
    try {
      fields = await $rpc.fieldsGet(model)
      const available = Object.keys(fields)
      columns = preferred.filter((name) => available.includes(name)).slice(0, 7)
      if (columns.length === 0) {
        columns = available.filter((name) => fields[name].type !== 'one2many' && fields[name].type !== 'many2many').slice(0, 6)
      }
      await reload()
    } catch (error) {
      showToast(error.message)
      records = []
      total = 0
    } finally {
      loading = false
    }
  }

  async function reload() {
    const domain = buildDomain()
    const result = await $rpc.searchRead(model, domain, columns, {
      offset: page * pageSize,
      limit: pageSize
    })
    records = result
    try {
      total = await $rpc.searchCount(model, domain)
    } catch {
      total = result.length
    }
  }

  function buildDomain() {
    if (!search.trim()) return []
    const term = search.trim()
    if (fields.name) return [['name', 'ilike', term]]
    if (fields.code) return [['code', 'ilike', term]]
    return [['rec_name', 'ilike', term]]
  }

  function formatValue(value) {
    if (value === null || value === undefined) return ''
    if (value instanceof Date) return value.toISOString().slice(0, 10)
    if (typeof value === 'boolean') return value ? '✓' : ''
    return String(value)
  }

  async function submitSearch(event) {
    event.preventDefault()
    page = 0
    try {
      await reload()
    } catch (error) {
      showToast(error.message)
    }
  }

  async function changePage(delta) {
    const next = page + delta
    if (next < 0 || next * pageSize >= total) return
    page = next
    try {
      await reload()
    } catch (error) {
      showToast(error.message)
    }
  }

  $effect(() => {
    model
    page = 0
    search = ''
    load()
  })
</script>

<div class="toolbar">
  <h2 style="margin: 0;">{model}</h2>
  <form onsubmit={submitSearch} style="display: flex; gap: 0.5rem;">
    <input type="search" bind:value={search} placeholder="Search…" />
    <button type="submit">Search</button>
  </form>
  <button class="primary" onclick={() => onOpenRecord(model, 0)}>New</button>
</div>

{#if loading}
  <p class="muted">Loading…</p>
{:else if records.length === 0}
  <p class="muted">No records.</p>
{:else}
  <table>
    <thead>
      <tr>
        {#each columns as column}
          <th>{fields[column]?.string || column}</th>
        {/each}
      </tr>
    </thead>
    <tbody>
      {#each records as record}
        <tr onclick={() => onOpenRecord(model, record.id)} style="cursor: pointer;">
          {#each columns as column}
            <td>{formatValue(record[column])}</td>
          {/each}
        </tr>
      {/each}
    </tbody>
  </table>

  <div class="toolbar" style="margin-top: 0.75rem;">
    <button onclick={() => changePage(-1)} disabled={page === 0}>Previous</button>
    <span class="muted">
      {page * pageSize + 1}–{Math.min((page + 1) * pageSize, total)} of {total}
    </span>
    <button onclick={() => changePage(1)} disabled={(page + 1) * pageSize >= total}>Next</button>
  </div>
{/if}
