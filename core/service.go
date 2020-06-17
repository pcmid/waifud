package core

var Services map[string]Service

func Register(s Service) {
	if Services == nil {
		Services = make(map[string]Service)
	}

	Services[s.Name()] = s
}

type R interface {
	Reply(source Message, message Message)
	Handle(message Message)
	PostHandle(message Message)
}

type Receiver struct{}

func (r *Receiver) Reply(source Message, message Message) {
	source.Get("_response").(chan Message) <- message
}

func (r *Receiver) PostHandle(message Message) {
	close(message.Get("_response").(chan Message))
}
func (r *Receiver) Handle(message Message) {}

type S interface {
	SetMessageChan(chan Message)
	Send(message Message) (response Message)
}

type Sender struct {
	senderChan chan Message
}

func (s *Sender) SetMessageChan(c chan Message) {
	s.senderChan = c
}

func (s *Sender) Send(message Message) Message {
	channel := make(chan Message)
	message.Set("_response", channel)

	s.senderChan <- message

	return <-channel
}

type Service interface {
	Name() string
	ListeningTypes() []string
	Start()

	R
	S
}
