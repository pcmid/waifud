package messages

type DBMessage struct {
	Code int
	URL  string
}

func (m *DBMessage) Reciver() string {
	return "database"
}

func (m *DBMessage) Describe() string {
	return "messages for database"
}

func (m *DBMessage) Message() interface{} {
	return m
}
