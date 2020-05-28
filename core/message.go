package core

type Message map[string]interface{}

func NewMessage(t string) Message {
	m := make(map[string]interface{})
	m["type"] = t
	return m
}

func (m Message) Type() string {
	return m["type"].(string)
}

func (m Message) Get(e string) interface{} {
	if v, ok := m[e]; ok {
		return v
	}
	return nil
}

func (m Message) Set(e string, v interface{}) Message {
	m[e] = v
	return m
}
