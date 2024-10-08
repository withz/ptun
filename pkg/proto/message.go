package proto

import (
	"encoding/json"
	"errors"
	"fmt"
	"reflect"
	"sync/atomic"
)

var messageRegistry = make(map[string]reflect.Type)

var sequence uint32 = 0

func Next() uint32 {
	return atomic.AddUint32(&sequence, 1)
}

type Request struct {
	Key     string
	Id      uint32
	DataRaw json.RawMessage `json:"DataRaw,omitempty"`

	data any
}

func NewRequest(data any) (req *Request) {
	req = &Request{
		Key:  typeOf(data).Name(),
		Id:   Next(),
		data: data,
	}
	return req
}

func (req *Request) GetKey() string {
	return req.Key
}

func (req *Request) Payload() any {
	return req.data
}

func (req *Request) Pack() (p []byte, err error) {
	p, err = json.Marshal(req.data)
	if err != nil {
		return nil, err
	}
	req.DataRaw = p
	return json.Marshal(req)
}

type Response struct {
	Key     string
	Id      uint32
	Code    int
	Message string
	DataRaw json.RawMessage `json:"DataRaw,omitempty"`

	data any
}

func NewIdResponse(id uint32, code int, message string, data any) (resp *Response) {
	resp = &Response{
		Key:     typeOf(data).Name(),
		Id:      id,
		Code:    code,
		Message: message,
		data:    data,
	}
	return resp
}

func (resp *Response) GetKey() string {
	return resp.Key
}

func (resp *Response) Payload() any {
	return resp.data
}

func (resp *Response) Pack() (p []byte, err error) {
	p, err = json.Marshal(resp.data)
	if err != nil {
		return nil, err
	}
	resp.DataRaw = p
	return json.Marshal(resp)
}

func UnpackRequest(p []byte) (req *Request, err error) {
	req = &Request{}
	err = json.Unmarshal(p, req)
	if err != nil {
		return nil, err
	}
	t, ok := messageRegistry[req.Key]
	if !ok {
		return nil, fmt.Errorf("invalid message")
	}
	if req.DataRaw == nil {
		return req, nil
	}
	req.data = reflect.New(t).Interface()
	err = json.Unmarshal(req.DataRaw, req.data)
	return req, err
}

func UnpackResponse(p []byte) (resp *Response, err error) {
	resp = &Response{}
	err = json.Unmarshal(p, resp)
	if err != nil {
		return nil, err
	}
	t, ok := messageRegistry[resp.Key]
	if !ok {
		return nil, fmt.Errorf("invalid message")
	}
	resp.data = reflect.New(t).Interface()
	err = json.Unmarshal(resp.DataRaw, resp.data)
	if err != nil {
		return nil, err
	}
	if resp.Code < 0 {
		return resp, errors.New(resp.Message)
	}
	return resp, nil
}

func GetResponsePayload[T any](resp *Response) (*T, error) {
	err := CheckType[T](resp.Payload())
	if err != nil {
		return nil, err
	}
	result := resp.Payload().(*T)
	return result, nil
}
