<script>
  import { rpc, notifications, notificationCount, showToast } from '../lib/store.js'

  let { onClose } = $props()

  async function markAllRead() {
    const ids = $notifications.filter((item) => item.id).map((item) => item.id)
    if (ids.length === 0) {
      onClose()
      return
    }
    try {
      await $rpc.executeButton('res.notification', 'mark_read', ids)
      notifications.set([])
      notificationCount.set(0)
      onClose()
    } catch (error) {
      showToast(error.message)
    }
  }
</script>

<aside class="notifications">
  <div class="toolbar">
    <strong>Notifications</strong>
    <button onclick={markAllRead}>Mark all read</button>
    <button onclick={onClose}>Close</button>
  </div>
  {#if $notifications.length === 0}
    <p class="muted">No notifications.</p>
  {:else}
    {#each $notifications as item}
      <div class="notification">
        <strong>{item.title || item.data?.title || 'Notification'}</strong>
        <p class="muted">{item.body || item.data?.body || ''}</p>
      </div>
    {/each}
  {/if}
</aside>
