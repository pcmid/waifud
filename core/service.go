package core

import "github.com/pcmid/waifud/messages"

type Reciver interface {
	Handle(message messages.Message)
}

type Sender interface {
	SetMessageChan(chan messages.Message)
	Send(message messages.Message)
}

type Service interface {
	Type() string
	Name() string
	Init()
	Serve()

	Reciver
	Sender
}
