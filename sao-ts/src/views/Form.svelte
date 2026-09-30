<script>
  import { onMount } from 'svelte'
  import { rpc, showToast } from '../lib/store.js'

  let { model, id } = $props()

  let fields = $state({})
  let values = $state({})
  let loading = $state(true)
  let saving = $state(false)
  let recordId = $state(id || 0)
  let editable = $state([])

  onMount(load)

  async function load() {
    loading = true
    recordId = id || 0
    try {
      fields = await $rpc.fieldsGet(model)
      editable = Object.keys(fields).filter((name) => {
        const field = fields[name]
        return field.type !== 'one2many' && field.type !== 'many2many' && field.readonly !== true
      })
      if (recordId > 0) {
        const records = await $rpc.read(model, [recordId], Object.keys(fields))
        values = records[0] || {}
      } else {
        values = {}
        await applyDefaults()
      }
    } catch (error) {
      showToast(error.message)
    } finally {
      loading = false
    }
  }

  async function applyDefaults() {
    try {
      const defaults = await $rpc.call(`model.${model}.default_get`, [editable, {}])
      values = { ...defaults }
    } catch {
      values = {}
    }
  }

  async function onBlur(field) {
    try {
      const changes = await $rpc.onChanged(model, values, [field])
      if (changes && Object.keys(changes).length > 0) {
        values = { ...values, ...changes }
      }
    } catch {
      // on_change is best-effort
    }
  }

  async function save() {
    saving = true
    try {
      if (recordId > 0) {
        await $rpc.write(model, [recordId], values)
      } else {
        const ids = await $rpc.create(model, values)
        recordId = ids[0]
        window.location.hash = `#/record/${model}/${recordId}`
      }
      showToast('Saved', 'success')
    } catch (error) {
      showToast(error.message)
    } finally {
      saving = false
    }
  }

  async function remove() {
    if (recordId <= 0) return
    saving = true
    try {
      await $rpc.remove(model, [recordId])
      window.location.hash = `#/model/${model}`
    } catch (error) {
      showToast(error.message)
    } finally {
      saving = false
    }
  }

  function setValue(name, value) {
    values = { ...values, [name]: value }
  }

  function inputType(type) {
    switch (type) {
      case 'integer':
      case 'float':
      case 'numeric':
        return 'number'
      case 'date':
        return 'date'
      case 'datetime':
        return 'datetime-local'
      case 'time':
        return 'time'
      default:
        return 'text'
    }
  }
</script>

<div class="toolbar">
  <h2 style="margin: 0;">{model} {recordId > 0 ? `#${recordId}` : '(new)'}</h2>
  <button class="primary" onclick={save} disabled={saving}>{saving ? 'Saving…' : 'Save'}</button>
  {#if recordId > 0}
    <button class="danger" onclick={remove} disabled={saving}>Delete</button>
  {/if}
  <button onclick={() => (window.location.hash = `#/model/${model}`)}>Close</button>
</div>

{#if loading}
  <p class="muted">Loading…</p>
{:else}
  <div class="form-grid">
    {#each editable as name}
      {@const field = fields[name]}
      <div class="field">
        <label for={`field-${name}`}>{field.string || name}</label>
        {#if field.type === 'boolean'}
          <input
            id={`field-${name}`}
            type="checkbox"
            checked={Boolean(values[name])}
            onchange={(event) => setValue(name, event.currentTarget.checked)}
          />
        {:else if field.type === 'selection' && field.selection}
          <select
            id={`field-${name}`}
            value={values[name] ?? ''}
            onchange={(event) => setValue(name, event.currentTarget.value)}
          >
            <option value=""></option>
            {#each field.selection as option}
              <option value={option[0]}>{option[1]}</option>
            {/each}
          </select>
        {:else if field.type === 'text'}
          <textarea
            id={`field-${name}`}
            rows="3"
            value={values[name] ?? ''}
            oninput={(event) => setValue(name, event.currentTarget.value)}
            onblur={() => onBlur(name)}
          ></textarea>
        {:else}
          <input
            id={`field-${name}`}
            type={inputType(field.type)}
            value={values[name] ?? ''}
            oninput={(event) => setValue(name, event.currentTarget.value)}
            onblur={() => onBlur(name)}
          />
        {/if}
      </div>
    {/each}
  </div>
{/if}
