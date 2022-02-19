package mirai

import (
	"avalon-core/src/log"
	"avalon-core/src/utils"
	"context"
	"strconv"
	"sync"
)

var Pool sync.Pool

var MiraiContext MessageContext

type MessageContext struct {
	Message *Message

	SyncId    int
	Index     int
	Handlers  []MiraiHandler
	WaitGroup sync.WaitGroup
}

type MiraiHandler func(*Message, chan *utils.TransmitEvent)

func MiraiListener(ctx context.Context, pipe *utils.TransmitterPipes) {
	MountMessageHandlers()

	miraiChan := make(chan *Message, 2)
	go SubscribeMirai(miraiChan)

	for {
		select {
		case e := <-pipe.In:
			go MiraiCommandDispatch(e)
		case m := <-miraiChan:
			go HandleMessage(m, pipe.Out)
		case <-ctx.Done():
			return
		}
	}
}

func MountMessageHandlers() {
	MiraiContext = MessageContext{
		Handlers: []MiraiHandler{},
	}

	MiraiContext.Handlers = append(MiraiContext.Handlers, MoectlUtils)
	MiraiContext.Handlers = append(MiraiContext.Handlers, UserRegRequest)
	MiraiContext.Handlers = append(MiraiContext.Handlers, UserRegOperation)
	MiraiContext.Handlers = append(MiraiContext.Handlers, BilibiliParseUtil)

	Pool = sync.Pool{New: func() interface{} { return &MessageContext{Handlers: MiraiContext.Handlers} }}
}

func HandleMessage(m *Message, outPipe chan *utils.TransmitEvent) {
	ctx := Pool.Get().(*MessageContext)
	ctx.Reset()
	ctx.Message = m
	ctx.Next(outPipe)
	Pool.Put(ctx)
}

func (ctx *MessageContext) Reset() {
	ctx.Index = 0
	ctx.Message = nil
}

func (ctx *MessageContext) Next(outPipe chan *utils.TransmitEvent) {
	for ctx.Index < len(ctx.Handlers) {
		if ctx.Message == nil {
			break
		}
		ctx.WaitGroup.Add(1)
		go func(i int) {
			ctx.Handlers[i](ctx.Message, outPipe)
			ctx.WaitGroup.Done()
		}(ctx.Index)
		ctx.Index++
	}
	ctx.WaitGroup.Wait()
}

func MiraiCommandDispatch(event *utils.TransmitEvent) {
	switch event.Event {
	case `SEND_FRIEND_MSG`:
		mc, err := GetMessageChainFromTaskList(event.Data.TaskList)
		if err != nil {
			log.Logger.Error(err)
			return
		}

		if mc == nil {
			return
		}

		target, err := strconv.ParseInt(event.Command, 10, 64)
		if err != nil {
			log.Logger.Error(err)
			return
		}

		err = SendFriendMessage(target, mc)
		if err != nil {
			log.Logger.Error(err)
			return
		}
	default:
		return
	}
}
