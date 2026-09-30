<script>
  import { onMount } from 'svelte'
  import { rpc } from '../lib/store.js'

  let { onOpenModel } = $props()

  let menus = $state([])
  let models = $state([])
  let modelFilter = $state('')
  let loading = $state(true)

  onMount(load)

  async function load() {
    loading = true
    try {
      menus = await $rpc.searchRead('ir.ui.menu', [], ['name', 'action', 'parent'], { limit: 200, order: 'name ASC' })
    } catch {
      menus = []
    }
    try {
      models = await $rpc.searchRead('ir.model', [], ['name', 'model'], { limit: 200, order: 'name ASC' })
    } catch {
      models = []
    }
    loading = false
  }

  async function openMenu(menu) {
    const action = menu.action
    if (!action) return
    const actionId = typeof action === 'string' ? Number(action.split(',')[1]) : action
    if (!actionId) return
    try {
      const value = await $rpc.getActionValue(actionId)
      if (value.res_model) onOpenModel(value.res_model)
    } catch {
      // ignore unknown actions
    }
  }

  const filteredModels = $derived(
    models.filter((model) =>
      `${model.name} ${model.model}`.toLowerCase().includes(modelFilter.toLowerCase())
    )
  )
</script>

<aside class="sidebar">
  <h3>Menu</h3>
  {#if loading}
    <p class="muted">Loading…</p>
  {:else if menus.length === 0}
    <p class="muted">No menus yet.</p>
  {:else}
    {#each menus as menu}
      <button class="menu-item" onclick={() => openMenu(menu)}>{menu.name}</button>
    {/each}
  {/if}

  <h3>Models</h3>
  <input
    type="search"
    placeholder="Filter models…"
    bind:value={modelFilter}
    oninput={(event) => event.stopPropagation()}
  />
  <div style="margin-top: 0.5rem; max-height: 320px; overflow-y: auto;">
    {#each filteredModels.slice(0, 100) as model}
      <button class="menu-item" onclick={() => onOpenModel(model.model)}>{model.name}</button>
    {/each}
    {#if filteredModels.length === 0 && modelFilter}
      <button class="menu-item" onclick={() => onOpenModel(modelFilter)}>
        Open “{modelFilter}”
      </button>
    {/if}
  </div>
</aside>
