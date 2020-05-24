package core

var Services map[string]Service

func Register(s Service) {
	if Services == nil {
		Services = make(map[string]Service)
	}

	Services[s.Name()] = s
}

type Receiver interface {
	Handle(message *Message)
}

type Sender interface {
	SetMessageChan(chan *Message)
	Send(message *Message)
}

type Service interface {
	Name() string
	ListeningTypes() []string
	Init()
	Serve()

	Receiver
	Sender
}
