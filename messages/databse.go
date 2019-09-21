package messages

import "fmt"

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

func (m *DBMessage) String() string {
	return fmt.Sprintf("%#v", m)
}
