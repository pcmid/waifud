package messages



type Message interface {
	Reciver() string
	Describe() string
	Message() interface{}
}

type BaseMessage struct {
	T string
	D string
	M interface{}
}

type ResultMessage struct {
	M string
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

func (r *ResultMessage) Reciver() string {
	return "ResultMessage"
}

func (r *ResultMessage) Describe() string {
	return "ResultMessage"
}

func (r *ResultMessage) Message() interface{} {
	return r.M
}
