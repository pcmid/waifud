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

func (m Message) WaitResponse() Message {
	if v, ok := m["_response"]; ok {
		return <-v.(chan Message)
	}
	return nil
}

func (m Message) Reply(r Message) {
	if v, ok := m["_response"]; ok {
		v.(chan Message) <- r
	}
}

func (m Message) Close() {
	if v, ok := m["_response"]; ok {
		close(v.(chan Message))
	}
}
