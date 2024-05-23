package httpserver

import (
	"dcron/server"
	"encoding/json"

	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/proto"
)

type Pong struct {
	Pong string `protobuf:"bytes,1,opt,name=pong,proto3" json:"pong,omitempty"`
}

type SuccessRes struct {
	Success bool          `json:"success"`
	Errors  []interface{} `json:"errors"`
}

type DataRespSchema struct {
	Data   interface{}   `json:"data"`
	Errors []interface{} `json:"errors"`
}

type Response struct {
	Success interface{}   `json:"success"`
	Errors  []interface{} `json:"errors"`
}

func ProtoJsonMarshal(m proto.Message) ([]byte, error) {
	pm := &protojson.MarshalOptions{
		EmitUnpopulated: true,
		UseProtoNames:   true,
	}
	return pm.Marshal(m)
}

// ErrorResponse
func ErrorResponse(err interface{}) Response {
	if err != nil {
		return Response{
			Success: false,
			Errors:  []interface{}{err},
		}
	}
	return Response{}
}

func ErrorDataRes(err interface{}) DataRespSchema {
	if err != nil {
		return DataRespSchema{
			Data:   make([]interface{}, 0),
			Errors: []interface{}{err},
		}
	}
	return DataRespSchema{}
}

func DataMarshal(template interface{}) []byte {
	var respTemplate = []byte(`{"data":[], "errors":[]}`)
	b, err := json.Marshal(template)
	if err != nil {
		server.GetServerInstance().GetLogger().Error("responsebyte json error: ", err)
		return respTemplate
	}
	return b
}

func DataSuccess(success bool) []byte {
	template := &SuccessRes{
		Success: success,
		Errors:  make([]interface{}, 0),
	}

	return DataMarshal(template)
}

func DataResp(data interface{}) []byte {
	template := &DataRespSchema{
		Data:   data,
		Errors: make([]interface{}, 0),
	}
	return DataMarshal(template)
}
