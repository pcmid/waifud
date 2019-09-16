package client

import (
	"encoding/json"
	"github.com/pcmid/waifud/messages"
	"github.com/pcmid/waifud/service/database"
	log "github.com/sirupsen/logrus"
	"io/ioutil"
	"net/http"
)

type JsonClient struct {
	BaseClient
	sms chan messages.Message
}

func (jc *JsonClient) Init() {
	//panic("implement me")
}

func (jc *JsonClient) Name() string {
	return "json client"
}

type Data struct {
	Op  string `json:"op"`
	URL string `json:"url"`
}

func (jc *JsonClient) Serve() {
	//panic("implement me")


	server := func(w http.ResponseWriter, r *http.Request) {
		body, _ := ioutil.ReadAll(r.Body)

		var data Data


		if err := json.Unmarshal(body, &data); err == nil {
			log.Trace(data)
			switch data.Op {
			case "sub":
				jc.Send(&messages.DBMessage{
					Code: database.AddFeed,
					URL:  data.URL,
				})

			case "unsub":
				jc.Send(&messages.DBMessage{
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

func (jc *JsonClient) Handle(message messages.Message) {
	//panic("implement me")
}

func (jc *JsonClient) SetMessageChan(ms chan messages.Message) {
	jc.sms = ms
}

func (jc *JsonClient) Send(message messages.Message) {
	if jc.sms == nil {
		return
	}

	jc.sms <- message
}
