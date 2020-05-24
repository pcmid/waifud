package core

type Message struct {
	Type string
	Content string

	Extra map[string]interface{}
}

func (m *Message) Get(e string) interface{} {
	return m.Extra[e]
}

func (m *Message)Set(e string, v interface{})  {
	if m.Extra == nil {
		m.Extra = make(map[string]interface{})
	}

	m.Extra[e] = v
}
