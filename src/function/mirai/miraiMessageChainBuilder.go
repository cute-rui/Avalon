package mirai

import "errors"

type MessageChainBuilder struct {
	MessageObject []*MessageObject
}

func GetMessageChainFromTaskList(s []string) ([]*MessageObject, error) {
	if s == nil || len(s) == 0 {
		return nil, errors.New(`unexpected data`)
	}
	var (
		i   int
		MCB MessageChainBuilder
	)
	for i < len(s) {
		switch s[i] {
		case `PLAIN`:
			i++
			var data string
			for i < len(s) {
				data = s[i]
				i++
			}
			MCB.Plain(data)

		}
	}

	return MCB.Done(), nil
}

func (m *MessageChainBuilder) Plain(str string) *MessageChainBuilder {
	m.MessageObject = append(m.MessageObject, &MessageObject{
		Type: MessageObjectType_Plain,
		Text: &str,
	})

	return m
}

func (m *MessageChainBuilder) ImageByID(imageID string) *MessageChainBuilder {
	m.MessageObject = append(m.MessageObject, &MessageObject{Type: MessageObjectType_Image, ImageId: &imageID})

	return m
}

func (m *MessageChainBuilder) ImageByURL(url string) *MessageChainBuilder {
	m.MessageObject = append(m.MessageObject, &MessageObject{Type: MessageObjectType_Image, Url: &url})

	return m
}

func (m *MessageChainBuilder) ImageByPath(path string) *MessageChainBuilder {
	m.MessageObject = append(m.MessageObject, &MessageObject{Type: MessageObjectType_Image, Path: &path})

	return m
}

func (m *MessageChainBuilder) ImageByBase64(b64 string) *MessageChainBuilder {
	m.MessageObject = append(m.MessageObject, &MessageObject{Type: MessageObjectType_Image, Base64: &b64})

	return m
}

func (m *MessageChainBuilder) Done() []*MessageObject {
	v := m.MessageObject
	m.MessageObject = nil
	return v
}
