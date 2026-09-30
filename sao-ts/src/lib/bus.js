export class Bus {
  constructor({ database, onMessage, onError }) {
    this.database = database
    this.onMessage = onMessage
    this.onError = onError
    this.lastMessage = null
    this.running = false
    this.channels = []
  }

  setChannels(channels) {
    this.channels = channels
  }

  async start() {
    if (this.running) return
    this.running = true
    while (this.running) {
      try {
        await this.poll()
      } catch (error) {
        if (!this.running) break
        if (this.onError) this.onError(error)
        await sleep(3000)
      }
    }
  }

  stop() {
    this.running = false
  }

  async poll() {
    if (this.channels.length === 0) {
      await sleep(1000)
      return
    }
    const response = await fetch(`/${encodeURIComponent(this.database)}/bus`, {
      method: 'POST',
      credentials: 'include',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({
        channels: this.channels,
        last_message: this.lastMessage
      })
    })
    if (!response.ok) {
      throw new Error(`bus error ${response.status}`)
    }
    const messages = await response.json()
    for (const item of messages || []) {
      this.lastMessage = item.message?.id || this.lastMessage
      if (this.onMessage) this.onMessage(item)
    }
  }
}

function sleep(ms) {
  return new Promise((resolve) => setTimeout(resolve, ms))
}
