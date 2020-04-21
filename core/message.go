package message

type Message struct {
	Receiver string
	Msg      interface{}
}

//func (m *NotifierMessage) Reciver() string {
//	return "notifier"
//}

func (m *Message) Describe() string {
	return "Result message"
}

func (m *Message) Message() interface{} {
	return m.Msg
}
