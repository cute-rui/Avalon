package utils

import "errors"

type TransmitEvent struct {
	ID          string
	Event       string
	Source      string
	Destination string
	Command     string
	Next        []string
	Data        Data
}

type Data struct {
	TaskList []string
}

type TransmitterPipes struct {
	In  chan *TransmitEvent
	Out chan *TransmitEvent
}

func (d *Data) NextAction(command string, args ...string) {

}

func (d *Data) Next() (string, error) {
	if d.TaskList == nil || len(d.TaskList) == 0 {
		return ``, errors.New(`no more tasks`)
	}
	s := d.TaskList[0]
	d.TaskList = d.TaskList[1:]
	return s, nil
}

func (d *Data) GetData() (string, error) {
	if d.TaskList == nil || len(d.TaskList) == 0 {
		return ``, errors.New(`no more tasks`)
	}
	s := d.TaskList[0]
	d.TaskList = d.TaskList[1:]
	return s, nil
}

func (e *TransmitEvent) RangeListeners(key, value interface{}) bool {
	if ch, ok := value.(chan *TransmitEvent); ok {
		ch <- e
		return true
	}

	return false
}
