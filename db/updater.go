package db

import (
	"container/list"
	"context"
	"log"
	"sync"

	"cloud.google.com/go/firestore"
)

const maxUpdateHandlers = 4

type updateKey struct {
	path  string
	field string
}

type updateValue struct {
	doc *firestore.DocumentRef
	val interface{}
}

type updateCommand struct {
	updateKey
	updateValue
}

type AsyncUpdater struct {
	sync.Mutex
	cmdMap   map[updateKey]*list.List
	semaChan chan struct{}
}

// Map command and launch command handler
func (u *AsyncUpdater) dispatchCommand(doc *firestore.DocumentRef, field string, val interface{}) {
	cmd := updateCommand{
		updateKey{
			doc.Path,
			field},
		updateValue{
			doc,
			val},
	}

	u.Lock()
	defer u.Unlock()

	listOfCommands, ok := u.cmdMap[cmd.updateKey]
	if !ok {
		listOfCommands = list.New()
		u.cmdMap[cmd.updateKey] = listOfCommands
	}
	listOfCommands.PushBack(cmd.updateValue)
	go u.commandHandler(cmd.updateKey)
}

// Amount of simultaneously launched command handlers is limited
func (u *AsyncUpdater) commandHandler(updateKey updateKey) {
	u.semaChan <- struct{}{}
	defer func() {
		<-u.semaChan
	}()

	u.Lock()
	cmdList := u.cmdMap[updateKey]
	delete(u.cmdMap, updateKey)
	u.Unlock()

	if cmdList != nil { // other goroutine could already do the job
		updateValue := cmdList.Back().Value.(updateValue)

		_, err := updateValue.doc.Update(context.Background(), []firestore.Update{
			{
				Path:  updateKey.field,
				Value: updateValue.val,
			},
		})
		if err != nil {
			log.Printf("Error while updating %v.%v to '%v':%v\n", updateKey.path, updateKey.field, updateValue.val, err)
		}
	}
}

func initAsyncUpdater() *AsyncUpdater {

	u := &AsyncUpdater{
		cmdMap:   make(map[updateKey]*list.List),
		semaChan: make(chan struct{}, maxUpdateHandlers),
	}

	return u
}
