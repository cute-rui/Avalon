package Moefunc

import (
	"avalon-core/src/dao"
	"avalon-core/src/log"
	"avalon-core/src/utils"
	"context"
	"strconv"
)

var GlobalSystemPipe chan *utils.TransmitEvent

func MoeFuncListener(ctx context.Context, pipe *utils.TransmitterPipes) {
	GlobalSystemPipe = pipe.Out

	for {
		select {
		case e := <-pipe.In:
			log.Logger.Info(e)
		case <-ctx.Done():
			return
		}
	}
}

func BilibiliSubscriptionSendMiraiBroadCast(data string, id int) error {
	users, err := dao.GetAllUsersByBilibiliSubscriptionID(id)
	if err != nil {
		return err
	}

	for i := range users {
		if users[i].QQ != 0 {
			SendMirai(data, users[i].QQ)
		}
	}

	return nil
}

func MikanSubscriptionSendMiraiBroadCast(data string, id int) error {
	users, err := dao.GetAllUsersByMikanSubscriptionID(id)
	if err != nil {
		return err
	}

	for i := range users {
		if users[i].QQ != 0 {
			SendMirai(data, users[i].QQ)
		}
	}

	return nil
}

func SendMirai(str string, target int64) {
	e := utils.TransmitEvent{
		ID:          utils.RandString(10),
		Event:       `SEND_FRIEND_MSG`,
		Source:      `moefunc`,
		Destination: `mirai`,
		Command:     strconv.FormatInt(target, 10),
		Next:        nil,
		Data:        utils.Data{TaskList: []string{`PLAIN`, str}},
	}

	GlobalSystemPipe <- &e
}
