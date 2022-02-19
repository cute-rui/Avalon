package controller

import (
	"avalon-core/src/utils"
	"sync"
)

var Listeners = sync.Map{}

var SysPipe chan *utils.TransmitEvent

func Broadcast() {
	for {
		e := <-SysPipe
		go func(event *utils.TransmitEvent) {
			if event.Destination == `` {
				Listeners.Range(event.RangeListeners)
				return
			}

			v, ok := Listeners.Load(event.Destination)
			if !ok {
				return
			}

			if ch, ok := v.(chan *utils.TransmitEvent); ok {
				ch <- event
				return
			}

			return
		}(e)
	}
}

func RegisterListener(name string, ch chan *utils.TransmitEvent) {
	Listeners.Store(name, ch)
}
