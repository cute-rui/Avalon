package controller

import (
	"avalon-core/src/utils"
	"context"
)

type Avalon struct {
	Ctx context.Context

	Transmitters map[string]func(context.Context, *utils.TransmitterPipes)
	Crons        map[string]func()
}

func Default() Avalon {
	return Avalon{Transmitters: map[string]func(context.Context, *utils.TransmitterPipes){}, Crons: map[string]func(){}}
}

func (c *Avalon) Load() {
	c.Ctx = context.Background()
	SysPipe = make(chan *utils.TransmitEvent, 4)
	for k, v := range c.Transmitters {
		in := make(chan *utils.TransmitEvent, 2)

		Pipe := utils.TransmitterPipes{
			In:  in,
			Out: SysPipe,
		}

		RegisterListener(k, in)
		go func(v func(context.Context, *utils.TransmitterPipes), pipe utils.TransmitterPipes) {
			v(c.Ctx, &Pipe)
		}(v, Pipe)
	}

	for _, v := range c.Crons {
		go func(v func()) {
			v()
		}(v)
	}

	Broadcast()
}
