package core

type Receiver interface {
	Handle(message Message)
}

type Sender interface {
	SetMessageChan(chan Message)
	Send(message Message)
}

type Service interface {
	Name() string
	ListeningTypes() []string
	Init()
	Serve()

	Receiver
	Sender
}
