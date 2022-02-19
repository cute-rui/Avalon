package controller

import (
	"avalon-core/src/utils"
	"context"
)

func (c *Avalon) MountTransmitters(name string, transmitter func(context.Context, *utils.TransmitterPipes)) {
	c.Transmitters[name] = transmitter
}

func (c *Avalon) MountCrons(name string, cron func()) {
	c.Crons[name] = cron
}
