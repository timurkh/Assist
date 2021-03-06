package db

import (
	"container/list"
	"context"
	"log"
	"sync"

	"cloud.google.com/go/firestore"
)

const maxUpdateHandlers = 4

type updatesList struct {
	doc    *firestore.DocumentRef
	values *list.List
}

type updateValue struct {
	field string
	val   interface{}
}

type AsyncUpdater struct {
	sync.Mutex
	cmdMap   map[string]*updatesList
	semaChan chan struct{}
}

// Map command and launch command handler
func (u *AsyncUpdater) dispatchCommand(doc *firestore.DocumentRef, field string, value interface{}) {
	key := doc.Path
	val := updateValue{
		field,
		value,
	}

	u.Lock()
	defer u.Unlock()

	updates, ok := u.cmdMap[key]
	if !ok {
		updates = &updatesList{
			doc,
			list.New(),
		}
		u.cmdMap[key] = updates
	}
	updates.values.PushBack(val)
	go u.commandHandler(key)
}

// Amount of simultaneously launched command handlers is limited
func (u *AsyncUpdater) commandHandler(key string) {
	u.semaChan <- struct{}{}
	defer func() {
		<-u.semaChan
	}()

	u.Lock()
	// take key out of map
	updates := u.cmdMap[key]
	delete(u.cmdMap, key)
	u.Unlock()

	if updates != nil { // other goroutine could already do the job before we took they key out of map
		valuesMap := make(map[string]interface{}, 0)
		for v := updates.values.Front(); v != nil; v = v.Next() {
			uv := v.Value.(updateValue)
			valuesMap[uv.field] = uv.val
		}

		updateArray := make([]firestore.Update, 0)
		for k, v := range valuesMap {
			updateArray = append(updateArray, firestore.Update{
				Path:  k,
				Value: v,
			})
		}
		_, err := updates.doc.Update(context.Background(), updateArray)
		if err != nil {
			log.Printf("Error while updating %v with %+v: %v\n", key, updateArray, err)
		}
	}
}

func initAsyncUpdater() *AsyncUpdater {

	u := &AsyncUpdater{
		cmdMap:   make(map[string]*updatesList),
		semaChan: make(chan struct{}, maxUpdateHandlers),
	}

	return u
}
