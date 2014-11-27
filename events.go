package db

import (
	//"log"
	"reflect"
)

type EntityEventHandler func(entity interface{})

type EntityEvents struct {
	events map[string][]EntityEventHandler
}

var entityEventHandlers = make(map[string]EntityEvents)

func getFullName(typ reflect.Type) string {
	return typ.PkgPath() + "." + typ.Name()
}

func getEntityName(model interface{}) string {
	val := reflect.ValueOf(model)
	ind := reflect.Indirect(val)
	return getFullName(ind.Type())
}

func OnEntityEvent(model interface{}, event string, handler EntityEventHandler) {
	modelName := getEntityName(model)
	events, exists := entityEventHandlers[modelName]
	//log.Println("register:", modelName, event, entityEventHandlers)

	if !exists {
		//log.Println("not exist!")
		events = EntityEvents{make(map[string][]EntityEventHandler)}
		entityEventHandlers[modelName] = events
	}

	events.registerHandler(event, handler)
}

func TriggerEntityEvent(model interface{}, event string, entity interface{}) {
	modelName := getEntityName(model)
	events, exists := entityEventHandlers[modelName]

	if exists {
		events.triggerEvent(event, entity)
	}
}

///

func (f *EntityEvents) registerHandler(event string, handler EntityEventHandler) {
	handlers, exists := f.events[event]

	if exists {
		handlers = append(handlers, handler)
	} else {
		handlers = make([]EntityEventHandler, 8)
		handlers = append(handlers[0:0], handler)
	}

	f.events[event] = handlers
}

func (f *EntityEvents) triggerEvent(event string, entity interface{}) {
	handlers, exists := f.events[event]
	if exists {
		for idx := range handlers {
			handlers[idx](entity)
		}
	}
}
