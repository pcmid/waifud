package client

import (
	"encoding/json"
	"github.com/pcmid/waifud/messages"
	"github.com/pcmid/waifud/service/database"
	log "github.com/sirupsen/logrus"
	"io/ioutil"
	"net/http"
)

type JsonAPI struct {
	BaseClient
	sms chan messages.Message
}

func (j *JsonAPI) Init() {
	//panic("implement me")
}

func (j *JsonAPI) Name() string {
	return "JsonAPI"
}

type Data struct {
	Op  string `json:"op"`
	URL string `json:"url"`
}

func (j *JsonAPI) Serve() {
	//panic("implement me")


	server := func(w http.ResponseWriter, r *http.Request) {
		body, _ := ioutil.ReadAll(r.Body)

		var data Data


		if err := json.Unmarshal(body, &data); err == nil {
			log.Trace(data)
			switch data.Op {
			case "sub":
				j.Send(&messages.DBMessage{
					Code: database.AddFeed,
					URL:  data.URL,
				})

			case "unsub":
				j.Send(&messages.DBMessage{
					Code: database.DelFeed,
					URL:  data.URL,
				})
			}

		} else {
			log.Error(err)
		}
	}

	http.HandleFunc("/", server)

	if err := http.ListenAndServe("127.0.0.1:8080", nil); err != nil {
		log.Error("ListenAndServe: ", err)
	}
}

func (j *JsonAPI) Handle(message messages.Message) {
	//panic("implement me")
}

func (j *JsonAPI) SetMessageChan(ms chan messages.Message) {
	j.sms = ms
}

func (j *JsonAPI) Send(message messages.Message) {
	if j.sms == nil {
		return
	}

	j.sms <- message
}
