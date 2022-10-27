package challenge

import (
	"sync"

	"github.com/ozean12/pritunl-zero/event"
	"github.com/ozean12/pritunl-zero/requires"
	"github.com/pritunl/mongo-go-driver/bson/primitive"
)

var (
	registry     = map[string]map[primitive.ObjectID]func(){}
	registryLock = sync.Mutex{}
)

func Register(token string, callback func()) primitive.ObjectID {
	listernerId := primitive.NewObjectID()

	registryLock.Lock()
	defer registryLock.Unlock()

	callbacks, ok := registry[token]
	if !ok {
		callbacks = map[primitive.ObjectID]func(){}
	}
	callbacks[listernerId] = callback
	registry[token] = callbacks

	return listernerId
}

func Unregister(token string, listenerId primitive.ObjectID) {
	registryLock.Lock()
	defer registryLock.Unlock()

	callbacks, ok := registry[token]
	if ok {
		delete(callbacks, listenerId)
		if len(callbacks) == 0 {
			delete(registry, token)
		} else {
			registry[token] = callbacks
		}
	}
}

func callback(evt *event.EventPublish) {
	token := evt.Data.(string)

	registryLock.Lock()
	defer registryLock.Unlock()

	callbacks, ok := registry[token]
	if ok {
		for _, callback := range callbacks {
			go callback()
		}
	}
}

func init() {
	module := requires.New("sshcert")
	module.After("settings")
	module.Before("event")

	module.Handler = func() (err error) {
		event.Register("ssh_challenge", callback)
		return
	}
}
