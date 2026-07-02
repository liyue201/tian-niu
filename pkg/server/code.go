package server

const (
	StatusOK             = 0
	StatusInvalidParam   = 10001
	StatusDuplicateEntry = 10004
	StatusUsernameError  = 10002
	StatusPasswordError  = 10003

	StatusInternalServerError = 50001
)

var statusText map[int]string

func init() {
	statusText = make(map[int]string)
	statusText[StatusOK] = "ok"
	statusText[StatusInvalidParam] = "invalid parameter"
	statusText[StatusUsernameError] = "username error"
	statusText[StatusPasswordError] = "password error"
	statusText[StatusDuplicateEntry] = "duplicate entry"
	statusText[StatusInternalServerError] = "internal server error"
}

type Code int

func (c Code) String() string {
	msg := statusText[int(c)]
	return msg
}
