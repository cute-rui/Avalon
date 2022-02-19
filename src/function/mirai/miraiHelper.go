package mirai

import (
	"avalon-core/src/config"
	"avalon-core/src/log"
	"context"
	"errors"
	"google.golang.org/grpc"
	"google.golang.org/grpc/status"
	"io"
	"time"
)

var Conn *grpc.ClientConn

func init() {
	c, err := grpc.Dial(config.Conf.GetString(`worker.mirai.grpc.addr`), grpc.WithInsecure())
	if err != nil {
		log.Logger.Error(err)
	}

	Conn = c
}

func SubscribeMirai(c chan *Message) {
	client := NewMiraiAgentClient(Conn)
	var channel MessageChannel
	switch config.Conf.GetString(`worker.mirai.account.channel`) {
	case `ALL`:
		channel = MessageChannel_all
	case `MESSAGE`:
		channel = MessageChannel_message
	case `EVENT`:
		channel = MessageChannel_event
	default:
		channel = MessageChannel_all
	}

	stream, err := client.Subscribe(context.Background(), &InitParam{
		VerifyKey:      config.Conf.GetString(`worker.mirai.account.verifyKey`),
		SessionKey:     `empty`,
		Qq:             config.Conf.GetInt64(`worker.mirai.account.qq`),
		MessageChannel: channel,
	})
	if err != nil {
		log.Logger.Error(err)
		return
	}
	for {
		message, err := stream.Recv()
		if err == io.EOF {
			return
		}
		if err != nil {
			if s, ok := status.FromError(err); ok {
				log.Logger.Error(err)
				if s.Code() == 14 {
					time.Sleep(20 * time.Second)
				}
			}
		}

		c <- message
	}
}

func SendFriendMessage(target int64, messageChain []*MessageObject) error {
	client := NewMiraiAgentClient(Conn)
	qq := config.Conf.GetInt64(`worker.mirai.account.qq`)

	message, err := client.SendFriendMessage(context.Background(), &SendFriendMessageParam{
		Target:       target,
		MessageChain: messageChain,
		BotQQNumber:  &qq,
	})

	if message.GetCode() != 0 {
		return errors.New(message.GetMsg())
	}

	return err
}

func SendFriendRequestResponse(event, target int64, operate int) error {
	client := NewMiraiAgentClient(Conn)
	qq := config.Conf.GetInt64(`worker.mirai.account.qq`)

	message, err := client.SendNewFriendRequestEventResponse(context.Background(), &NewFriendRequestEventResponse{
		EventId:     event,
		FromId:      target,
		Operate:     int32(operate),
		Message:     "",
		BotQQNumber: &qq,
	})

	if message.GetCode() != 0 {
		return errors.New(message.GetMsg())
	}

	return err
}
