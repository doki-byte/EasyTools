package proxy

import "fmt"

type Response struct {
	Code    int
	Message string
	Data    interface{}
}

func (p *Proxy) errorResponse(message ...interface{}) Response {
	return Response{
		Code:    400,
		Message: fmt.Sprint(message...),
		Data:    nil,
	}
}

func (p *Proxy) successResponse(msg string, data interface{}) Response {
	return Response{
		Code:    200,
		Message: msg,
		Data:    data,
	}
}
