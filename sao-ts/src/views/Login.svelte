<script>
  import { onMount } from 'svelte'
  import { RPC } from '../lib/rpc.js'

  let { onLoggedIn } = $props()

  let databases = $state([])
  let database = $state('')
  let login = $state('admin')
  let password = $state('')
  let error = $state('')
  let loading = $state(false)

  onMount(async () => {
    try {
      const client = new RPC({ database: '' })
      databases = await client.listDatabases()
      if (databases.length > 0) database = databases[0]
    } catch (err) {
      error = err.message
    }
  })

  async function submit(event) {
    event.preventDefault()
    error = ''
    loading = true
    try {
      const client = new RPC({ database })
      const result = await client.login(database, login, password)
      onLoggedIn({ database, userId: result[0] })
    } catch (err) {
      error = err.message
    } finally {
      loading = false
    }
  }
</script>

<form class="login" onsubmit={submit}>
  <h1>Talon</h1>
  <div class="field">
    <label for="database">Database</label>
    {#if databases.length > 0}
      <select id="database" bind:value={database}>
        {#each databases as name}
          <option value={name}>{name}</option>
        {/each}
      </select>
    {:else}
      <input id="database" bind:value={database} placeholder="database" />
    {/if}
  </div>
  <div class="field">
    <label for="login">Login</label>
    <input id="login" bind:value={login} autocomplete="username" />
  </div>
  <div class="field">
    <label for="password">Password</label>
    <input id="password" type="password" bind:value={password} autocomplete="current-password" />
  </div>
  {#if error}
    <p class="error">{error}</p>
  {/if}
  <button class="primary" type="submit" disabled={loading || !database}>
    {loading ? 'Signing in…' : 'Sign in'}
  </button>
</form>
