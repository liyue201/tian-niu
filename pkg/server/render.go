package server

import (
	"errors"
	"net/http"
	"runtime"

	"github.com/gin-gonic/gin"
	"github.com/liyue201/tian-niu/pkg/shared/log"
)

type Response struct {
	Code int         `json:"code"`
	Msg  string      `json:"msg"`
	More string      `json:"more,omitempty"`
	Data interface{} `json:"data"`
}

func respondSuccess(c *gin.Context, data interface{}) {
	respondData(c, StatusOK, data)
}

func respondData(c *gin.Context, code Code, data interface{}) {
	if code != StatusOK {
		_, file, line, _ := runtime.Caller(1)
		log.Errorf("%v:%v, response %v", file, line, code)
	}
	ret := Response{
		Code: int(code),
		Msg:  code.String(),
		Data: data,
	}
	c.JSON(http.StatusOK, ret)
}

func respondDataEx(c *gin.Context, code Code, msg string, data interface{}) {
	if msg == "" {
		msg = code.String()
	}
	if code != StatusOK {
		_, file, line, _ := runtime.Caller(1)
		log.Errorf("%v:%v, response %v, %v", file, line, code, msg)
	}

	ret := Response{
		Code: int(code),
		Msg:  msg,
		Data: data,
	}
	c.JSON(http.StatusOK, ret)
}

func respondError(c *gin.Context, code Code, e error) {
	if e == nil {
		e = errors.New("")
	}
	_, file, line, _ := runtime.Caller(1)
	log.Errorf("%v:%v, response %v, %v", file, line, code, e.Error())

	ret := Response{
		Code: int(code),
		Msg:  code.String(),
		Data: nil,
		More: e.Error(),
	}

	httpStatus := errorHTTPStatus(code)

	c.JSON(httpStatus, ret)
}

func errorHTTPStatus(code Code) int {
	switch code {
	case StatusInvalidParam:
		return http.StatusBadRequest
	case StatusUsernameError, StatusPasswordError:
		return http.StatusUnauthorized
	case StatusDuplicateEntry:
		return http.StatusConflict
	case StatusInternalServerError:
		return http.StatusInternalServerError
	default:
		return http.StatusOK
	}
}
