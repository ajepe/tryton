<script>
  import { onMount, onDestroy } from 'svelte'
  import { RPC } from './lib/rpc.js'
  import { Bus } from './lib/bus.js'
  import {
    rpc as rpcStore,
    session,
    toast,
    menuOpen,
    notifications,
    notificationCount,
    showToast
  } from './lib/store.js'
  import Login from './views/Login.svelte'
  import Menu from './views/Menu.svelte'
  import Tree from './views/Tree.svelte'
  import Form from './views/Form.svelte'
  import Notifications from './views/Notifications.svelte'

  let rpc = null
  let bus = null
  let route = $state({ view: 'home' })
  let showNotifications = $state(false)

  function parseHash() {
    const hash = window.location.hash.replace(/^#\/?/, '')
    const parts = hash.split('/')
    if (parts[0] === 'model' && parts[1]) {
      return { view: 'model', model: parts[1] }
    }
    if (parts[0] === 'record' && parts[1]) {
      return { view: 'record', model: parts[1], id: Number(parts[2] || 0) }
    }
    return { view: 'home' }
  }

  function onHashChange() {
    route = parseHash()
  }

  onMount(() => {
    window.addEventListener('hashchange', onHashChange)
    return () => window.removeEventListener('hashchange', onHashChange)
  })

  onDestroy(() => {
    if (bus) bus.stop()
  })

  export function openModel(model) {
    window.location.hash = `#/model/${model}`
  }

  function openRecord(model, id) {
    window.location.hash = `#/record/${model}/${id}`
  }

  function handleLoggedIn({ database, userId }) {
    rpc = new RPC({ database, onUnauthorized: handleLogout })
    rpcStore.set(rpc)
    session.set({ database, userId })
    startBus(database, userId)
  }

  function handleLogout() {
    if (bus) {
      bus.stop()
      bus = null
    }
    rpcStore.set(null)
    session.set(null)
    notifications.set([])
    notificationCount.set(0)
    window.location.hash = ''
    route = { view: 'home' }
  }

  async function logout() {
    if (rpc) await rpc.logout()
    handleLogout()
  }

  function startBus(database, userId) {
    bus = new Bus({
      database,
      onMessage: (item) => {
        notifications.update((list) => [item.message, ...list])
        notificationCount.update((count) => count + 1)
      },
      onError: () => {}
    })
    bus.setChannels([`notification:${userId}`])
    bus.start()
  }

  async function loadNotifications() {
    showNotifications = !showNotifications
    if (!showNotifications || !rpc) return
    try {
      const records = await rpc.searchRead(
        'res.notification',
        [['read', '=', false]],
        ['title', 'body', 'read'],
        { limit: 20 }
      )
      notifications.set(records)
      notificationCount.set(records.length)
    } catch (error) {
      showToast(error.message)
    }
  }
</script>

{#if !$session}
  <Login onLoggedIn={handleLoggedIn} />
{:else}
  <header class="app-header">
    <button onclick={() => (menuOpen.update((value) => !value))} aria-label="Toggle menu">☰</button>
    <h1>Talon</h1>
    <button onclick={loadNotifications}>
      Notifications
      {#if $notificationCount > 0}<span class="badge">{$notificationCount}</span>{/if}
    </button>
    <button onclick={logout}>Logout</button>
  </header>

  <div class="layout">
    <Menu onOpenModel={openModel} />
    <main class="content">
      {#if route.view === 'model'}
        {#key route.model}
          <Tree model={route.model} onOpenRecord={openRecord} />
        {/key}
      {:else if route.view === 'record'}
        {#key `${route.model}:${route.id}`}
          <Form model={route.model} id={route.id} />
        {/key}
      {:else}
        <h2>Welcome</h2>
        <p class="muted">Select a menu entry or open a model from the sidebar.</p>
      {/if}
    </main>
  </div>

  {#if showNotifications}
    <Notifications onClose={() => (showNotifications = false)} />
  {/if}
{/if}

{#if $toast}
  <div class="toast {$toast.type}">{$toast.message}</div>
{/if}
