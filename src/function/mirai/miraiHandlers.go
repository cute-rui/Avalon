package mirai

import (
	"avalon-core/src/dao"
	"avalon-core/src/function/Moefunc"
	"avalon-core/src/function/bilibili"
	"avalon-core/src/log"
	"avalon-core/src/utils"
	"strconv"
	"strings"
)

func MoectlUtils(m *Message, ch chan *utils.TransmitEvent) {
	messageChain := m.GetMessageChain()
	if messageChain == nil {
		return
	}

	for i := range messageChain {
		if messageChain[i].GetType() != MessageObjectType_Plain {
			continue
		}

		var cmdArr []string
		if !strings.HasPrefix(messageChain[i].GetText(), `moectl `) {
			continue
		} else {
			cmdArr = strings.Split(strings.TrimPrefix(messageChain[i].GetText(), `moectl `), ` `)
		}

		MoectlCommandDispatch(ch, m.GetType(), m.GetSender().GetId(), m.GetSender().GetGroup().GetId(), cmdArr...)
	}
}

func MoectlCommandDispatch(ch chan *utils.TransmitEvent, MsgType MessageType, QQ int64, GroupId int64, args ...string) {
	i := len(args)
	if i <= 1 {
		return
	}

	switch {
	case strings.EqualFold(args[0], `sub`):
		MoectlSubscribe(ch, MsgType, QQ, args[1:]...)
	case strings.EqualFold(args[0], `download`):
		MoectlDownload(ch, MsgType, QQ, args[1:]...)
	case strings.EqualFold(args[0], `music`):
		//under construction
	case strings.EqualFold(args[0], `article`):
		//under construction
	default:
		return
	}
}

func MoectlSubscribe(ch chan *utils.TransmitEvent, MsgType MessageType, QQ int64, Data ...string) {
	if MsgType == MessageType_FriendMessage {
		if strings.Contains(Data[0], `bilibili`) || strings.Contains(Data[0], `b23`) {
			BilibiliDownload(QQ, Data[0])
		} else if strings.Contains(Data[0], `mikan`) {
			MikanSubscription(QQ, Data[0])
		}
	}
}

func MoectlDownload(ch chan *utils.TransmitEvent, MsgType MessageType, QQ int64, Data ...string) {
	if MsgType == MessageType_FriendMessage {
		if strings.Contains(Data[0], `bilibili`) || strings.Contains(Data[0], `b23`) {
			BilibiliDownload(QQ, Data[0])
		}
	}
}

func BilibiliDownload(QQ int64, data string) {
	user, err := dao.GetUserByQQ(QQ)
	if err != nil {
		log.Logger.Error(err)
		return
	}

	var MCB MessageChainBuilder
	if Moefunc.IsTaskOnGoing(data) {
		err = SendFriendMessage(QQ, MCB.Plain(`资源正在加载中，请稍后`).Done())
		if err != nil {
			log.Logger.Error(err)
		}
	}

	response := Moefunc.BilibiliUniversalDownload(user, data)
	if response == `` {
		err = SendFriendMessage(QQ, MCB.Plain(`发生错误`).Done())
		if err != nil {
			log.Logger.Error(err)
		}
		return
	}

	err = SendFriendMessage(QQ, MCB.Plain(utils.StringBuilder("操作成功，目前已更新资源列表：\n", response)).Done())
	if err != nil {
		log.Logger.Error(err)
	}
}

func MikanSubscription(QQ int64, data string) {
	user, err := dao.GetUserByQQ(QQ)
	if err != nil {
		log.Logger.Error(err)
		return
	}

	var MCB MessageChainBuilder
	err = SendFriendMessage(QQ, MCB.Plain(Moefunc.MikanSubscribe(user, data)).Done())
	if err != nil {
		log.Logger.Error(err)
	}
}

func BilibiliParseUtil(m *Message, ch chan *utils.TransmitEvent) {
	if m.GetType() != MessageType_FriendMessage && m.GetType() != MessageType_GroupMessage {
		return
	}

	messageChain := m.GetMessageChain()
	if messageChain == nil {
		return
	}

	for i := range messageChain {
		var s string
		switch messageChain[i].GetType() {
		case MessageObjectType_Plain:
			if !CheckIsBilibili(messageChain[i].GetText()) {
				continue
			}
			s = messageChain[i].GetText()
		case MessageObjectType_App:
			if !CheckIsBilibili(messageChain[i].GetText()) {
				continue
			}
			s = messageChain[i].GetContent()
		case MessageObjectType_Xml:
			if !CheckIsBilibili(messageChain[i].GetText()) {
				continue
			}
			s = messageChain[i].GetXml()
		default:
			continue
		}

		info, err := bilibili.GetInfo(s)
		if err != nil {
			log.Logger.Error(err)
			continue
		}

		if info == nil {
			continue
		}

		pic, str := info.GetString()
		var MCB MessageChainBuilder
		err = SendFriendMessage(m.GetSender().GetId(), MCB.ImageByURL(pic).Plain(str).Done())
		if err != nil {
			log.Logger.Error(err)
			continue
		}
	}
}

func CheckIsBilibili(str string) bool {
	if strings.Contains(str, `bilibili.com`) {
		return true
	} else if strings.Contains(str, `b23.tv`) {
		return true
	} else if strings.Contains(str, `bv`) || strings.Contains(str, `BV`) {
		return true
	} else if strings.Contains(str, `av`) || strings.Contains(str, `AV`) {
		return true
	}

	return false
}

func UserRegRequest(m *Message, ch chan *utils.TransmitEvent) {
	t := m.GetType()
	if t != MessageType_NewFriendRequestEvent {
		return
	}

	token := utils.RandString(20)
	err := SetNewFriendRuntimeData(m, token)
	if err != nil {
		log.Logger.Error(err)
		return
	}
	err = SendBroadCastToAdmin(GetNewFriendRequest(m), token)
	if err != nil {
		log.Logger.Error(err)
	}
}

func SetNewFriendRuntimeData(m *Message, token string) error {
	return dao.NewFriendRequest(strings.Join([]string{strconv.Itoa(int(m.GetEventId())), strconv.Itoa(int(m.GetFromId())), token}, `.`))
}

func GetNewFriendRequest(m *Message) string {
	return utils.StringBuilder(`新的好友请求, 账号:`, strconv.Itoa(int(m.GetFromId())), "\n", `昵称:`, m.GetNick(), "\n", `请求信息:`, m.GetMessage())
}

func SendBroadCastToAdmin(str string, token string) error {
	authReply := utils.StringBuilder(`请回复本条消息进行操作, accept:同意, deny:拒绝, black:小黑屋, TOKEN:`, token)

	admins, err := dao.GetAdmin()
	if err != nil {
		return err
	}

	for i := range admins {
		var MCB MessageChainBuilder
		err := SendFriendMessage(admins[i].QQ, MCB.Plain(str).Done())
		if err != nil {
			return err
		}

		err = SendFriendMessage(admins[i].QQ, MCB.Plain(authReply).Done())
		if err != nil {
			return err
		}
	}

	return nil
}

func UserRegOperation(m *Message, ch chan *utils.TransmitEvent) {
	if m.GetType() != MessageType_FriendMessage {
		return
	}

	var token string
	t := -1
	messageChain := m.GetMessageChain()
	for i := range messageChain {
		if messageChain[i].GetType() == MessageObjectType_Quote {
			Quote := messageChain[i].GetOrigin()
			for j := range Quote {
				if Quote[j].GetType() != MessageObjectType_Plain {
					continue
				}

				if !strings.HasPrefix(Quote[j].GetText(), `请回复本条消息进行操作, accept:同意, deny:拒绝, black:小黑屋, TOKEN:`) {
					continue
				}

				token = strings.TrimPrefix(Quote[j].GetText(), `请回复本条消息进行操作, accept:同意, deny:拒绝, black:小黑屋, TOKEN:`)
			}
		}

		if messageChain[i].GetType() == MessageObjectType_Plain {
			switch {
			case strings.Contains(messageChain[i].GetText(), `accept`):
				t = 0
			case strings.Contains(messageChain[i].GetText(), `deny`):
				t = 1
			case strings.Contains(messageChain[i].GetText(), `black`):
				t = 2
			}
		}
	}

	if token == `` || t == -1 {
		return
	}

	if !dao.IsInAdmin(m.GetSender().GetId()) {
		return
	}

	UserRegFromQQ(token, t)
}

func UserRegFromQQ(token string, t int) {
	runtime, err := dao.GetFriendRequest(token)
	if err != nil {
		log.Logger.Error(err)
		return
	}

	if runtime == nil {
		return
	}

	strs := strings.Split(runtime.Data, `.`)
	if len(strs) != 3 {
		err := dao.DeleteFriendRequest(token)
		if err != nil {
			log.Logger.Error(err)
		}
		return
	}

	event, err := strconv.ParseInt(strs[0], 10, 64)
	qq, err := strconv.ParseInt(strs[1], 10, 64)
	if t == 0 {
		if err != nil {
			log.Logger.Error(err)
			return
		}
		err = dao.CreateUserWithQQ(qq)
		if err != nil {
			log.Logger.Error(err)
			return
		}
	}

	err = dao.DeleteFriendRequest(token)
	if err != nil {
		log.Logger.Error(err)
	}

	err = SendFriendRequestResponse(event, qq, t)
	if err != nil {
		log.Logger.Error(err)
	}
}

func GroupRegUtil(m *Message, ch chan *utils.TransmitEvent) {

}

func TransmitWrapper(dst string, data []string) *utils.TransmitEvent {
	return nil
}
