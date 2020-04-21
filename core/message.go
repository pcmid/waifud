package core

type Message struct {
	Type string
	Msg  interface{}
}

func (m *Message) Message() interface{} {
	return m.Msg
}
