package service

import "fmt"

// ConflictError 表示业务唯一性或占用冲突（如标题重复、记录被引用），
// handler 层应以 409 状态码返回并透传其中的中文提示
type ConflictError struct {
	Message string
}

func (e *ConflictError) Error() string {
	return e.Message
}

// NewConflictError 构造业务冲突错误
func NewConflictError(format string, args ...interface{}) *ConflictError {
	return &ConflictError{Message: fmt.Sprintf(format, args...)}
}
