package nsqtarget

import (
	"dcron/server"
	"encoding/json"
)

type NSQProducer interface {
	Publish(topic string, body []byte) error
}

var p NSQProducer

func ConfigInit() {
	p = server.GetServerInstance().GetNSQProducer()
}

func Publish(topic string, message string) error {
	var data map[string]interface{}

	err := json.Unmarshal([]byte(message), &data)
	if err != nil {
		return err
	}
	b, _ := json.Marshal(data)

	return p.Publish(topic, b)
}
