package messages

type Message interface {
	Reciver() string
	Describe() string
	Message() interface{}
	String() string
}

type BaseMessage struct {
	T string
	D string
	M interface{}
}

func (bm *BaseMessage) Reciver() string {
	return bm.T
}

func (bm *BaseMessage) Describe() string {
	return bm.D
}

func (bm *BaseMessage) Message() interface{} {
	return bm.M
}

func (bm *BaseMessage) String() string {
	return bm.M.(string)
}
