package messages

type DLMessage struct {
	URL string
}

func (d *DLMessage) Reciver() string {
	return "downloader"
}

func (d *DLMessage) Describe() string {
	return "downloader message"
}

func (d *DLMessage) Message() interface{} {
	return d.URL
}

func (d *DLMessage) String() string {
	return d.URL
}
