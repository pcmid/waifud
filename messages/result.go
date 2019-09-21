package messages

import "fmt"

type ResultMessage struct {
	Status string
	Msg    string
}

func (r *ResultMessage) Reciver() string {
	return "notifier"
}

func (r *ResultMessage) Describe() string {
	return "Result message"
}

func (r *ResultMessage) Message() interface{} {
	return r.Msg
}

func (r *ResultMessage) String() string {
	return fmt.Sprintf("%#v", r)
}
