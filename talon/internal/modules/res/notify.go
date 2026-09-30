package res

import "sync"

type Notifier interface {
	Notify(channel string, data map[string]interface{})
}

var (
	notifierMu sync.RWMutex
	notifier   Notifier
)

func SetNotifier(n Notifier) {
	notifierMu.Lock()
	defer notifierMu.Unlock()
	notifier = n
}

func publish(channel string, data map[string]interface{}) {
	notifierMu.RLock()
	defer notifierMu.RUnlock()
	if notifier != nil {
		notifier.Notify(channel, data)
	}
}
