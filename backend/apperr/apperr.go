// Package apperr 定义业务错误码与统一错误类型。
//
// 约定：
//   - 4 位业务码返回给前端，前端按 code 分支处理，不解析文案；
//   - message 是可直接展示给用户的中文提示；
//   - 内部错误（err 字段）只写日志，不返回给客户端。
package apperr

import (
	"errors"
	"fmt"
	"net/http"
)

// Code 是返回给前端的业务错误码。
type Code int

const (
	CodeOK Code = 0

	// 1xxx 通用
	CodeInvalidParam Code = 1000
	CodeUnauthorized Code = 1001
	CodeTokenExpired Code = 1002
	CodeForbidden    Code = 1003
	CodeNotFound     Code = 1004
	CodeTooManyReq   Code = 1005

	// 2xxx 账号
	CodeEmailTaken      Code = 2001
	CodeUserNotFound    Code = 2002
	CodeBadCredentials  Code = 2003
	CodeAccountDisabled Code = 2004
	CodeGuestDisabled   Code = 2005
	CodeGuestExpired    Code = 2006

	// 3xxx Agent / 模型
	CodeAgentNotFound   Code = 3001
	CodeAgentNameTaken  Code = 3002
	CodeModelNotSupport Code = 3003
	CodeModelKeyMissing Code = 3004
	CodeModelCallFailed Code = 3005
	CodeEmptyCompletion Code = 3006
	CodeInferRunning    Code = 3007

	// 4xxx 知识库 / RAG
	CodeKnowledgeNotFound   Code = 4001
	CodeUnsupportedFile     Code = 4002
	CodeFileParseFailed     Code = 4003
	CodeKnowledgeNotMounted Code = 4004
	CodeEmbeddingFailed     Code = 4005

	// 6xxx 广场 / 分享 / 互动
	CodeShareInvalid       Code = 6001
	CodeShareExpired       Code = 6002
	CodeShareForbidden     Code = 6003
	CodeCanvasNotPublished Code = 6004
	CodeCommentNotFound    Code = 6005

	// 7xxx REPL 沙箱
	CodeReplUnavailable  Code = 7001
	CodeReplNotFound     Code = 7002
	CodeReplTooMany      Code = 7003
	CodeReplAttached     Code = 7004
	CodeReplLanguage     Code = 7005
	CodeReplCodeTooLarge Code = 7006

	// 5xxx 服务端
	CodeInternal Code = 5000
)

type codeMeta struct {
	httpStatus int
	message    string
}

var codeTable = map[Code]codeMeta{
	CodeOK:                  {http.StatusOK, "ok"},
	CodeInvalidParam:        {http.StatusBadRequest, "请求参数不正确"},
	CodeUnauthorized:        {http.StatusUnauthorized, "请先登录"},
	CodeTokenExpired:        {http.StatusUnauthorized, "登录已过期，请重新登录"},
	CodeForbidden:           {http.StatusForbidden, "没有权限执行该操作"},
	CodeNotFound:            {http.StatusNotFound, "请求的资源不存在"},
	CodeTooManyReq:          {http.StatusTooManyRequests, "请求过于频繁，请稍后再试"},
	CodeEmailTaken:          {http.StatusConflict, "该邮箱已被注册"},
	CodeUserNotFound:        {http.StatusNotFound, "用户不存在"},
	CodeBadCredentials:      {http.StatusUnauthorized, "邮箱或密码不正确"},
	CodeAccountDisabled:     {http.StatusForbidden, "账号已被禁用"},
	CodeGuestDisabled:       {http.StatusForbidden, "当前环境未开放游客访问"},
	CodeGuestExpired:        {http.StatusUnauthorized, "游客会话已过期，请重新进入"},
	CodeAgentNotFound:       {http.StatusNotFound, "Agent 不存在"},
	CodeAgentNameTaken:      {http.StatusConflict, "同名的 Agent 已存在"},
	CodeModelNotSupport:     {http.StatusBadRequest, "不支持的模型"},
	CodeModelKeyMissing:     {http.StatusBadRequest, "该模型需要先配置 API Key"},
	CodeModelCallFailed:     {http.StatusBadGateway, "模型调用失败，请稍后重试"},
	CodeEmptyCompletion:     {http.StatusBadGateway, "模型没有返回可用内容"},
	CodeInferRunning:        {http.StatusConflict, "该画布已有推演在进行中"},
	CodeKnowledgeNotFound:   {http.StatusNotFound, "知识库文件不存在"},
	CodeUnsupportedFile:     {http.StatusBadRequest, "不支持的文件类型"},
	CodeFileParseFailed:     {http.StatusBadRequest, "文件解析失败或没有可用文本"},
	CodeKnowledgeNotMounted: {http.StatusBadRequest, "该画布未挂载知识库"},
	CodeEmbeddingFailed:     {http.StatusBadGateway, "文本向量化失败"},
	CodeShareInvalid:        {http.StatusNotFound, "分享链接无效或已停用"},
	CodeShareExpired:        {http.StatusGone, "分享链接已过期"},
	CodeShareForbidden:      {http.StatusForbidden, "该分享链接不允许 Fork"},
	CodeCanvasNotPublished:  {http.StatusBadRequest, "该画布尚未发布到广场"},
	CodeCommentNotFound:     {http.StatusNotFound, "评论不存在或已删除"},
	CodeReplUnavailable:     {http.StatusServiceUnavailable, "REPL 沙箱暂不可用"},
	CodeReplNotFound:        {http.StatusNotFound, "REPL 会话不存在或已结束"},
	CodeReplTooMany:         {http.StatusTooManyRequests, "同时运行的 REPL 会话过多"},
	CodeReplAttached:        {http.StatusConflict, "该 REPL 会话已被其他终端占用"},
	CodeReplLanguage:        {http.StatusBadRequest, "不支持的 REPL 语言"},
	CodeReplCodeTooLarge:    {http.StatusBadRequest, "代码过长"},
	CodeInternal:            {http.StatusInternalServerError, "服务开小差了，请稍后重试"},
}

// Error 是贯穿 service/handler/middleware 的统一业务错误。
type Error struct {
	Code       Code
	HTTPStatus int
	Message    string
	// Details 可选，用于参数校验等场景，原样返回给前端。
	Details any
	// cause 只在日志中出现，绝不序列化给客户端。
	cause error
}

// New 按业务码创建错误，message 为空时使用默认文案。
func New(code Code, message string) *Error {
	e := &Error{Code: code, HTTPStatus: http.StatusInternalServerError, Message: "服务异常"}
	if meta, ok := codeTable[code]; ok {
		e.HTTPStatus = meta.httpStatus
		e.Message = meta.message
	} else {
		e.HTTPStatus = http.StatusBadRequest
	}
	if message != "" {
		e.Message = message
	}
	return e
}

// Newf 使用格式化文案创建错误。
func Newf(code Code, format string, args ...any) *Error {
	return New(code, fmt.Sprintf(format, args...))
}

// Wrap 在保留业务码的同时记录内部原因（仅日志可见）。
func Wrap(code Code, message string, cause error) *Error {
	e := New(code, message)
	e.cause = cause
	return e
}

// WithDetails 附加结构化的字段级错误信息。
func (e *Error) WithDetails(details any) *Error {
	e.Details = details
	return e
}

// WithMessage 覆盖默认文案。
func (e *Error) WithMessage(message string) *Error {
	e.Message = message
	return e
}

func (e *Error) Error() string {
	if e.cause != nil {
		return fmt.Sprintf("biz error code=%d msg=%s cause=%v", e.Code, e.Message, e.cause)
	}
	return fmt.Sprintf("biz error code=%d msg=%s", e.Code, e.Message)
}

func (e *Error) Unwrap() error { return e.cause }

// Status 返回应写入的 HTTP 状态码。
func (e *Error) Status() int {
	if e.HTTPStatus <= 0 {
		return http.StatusInternalServerError
	}
	return e.HTTPStatus
}

// As 把任意 error 转成 *Error；非业务错误统一按 5000 处理。
func As(err error) (*Error, bool) {
	var appErr *Error
	if errors.As(err, &appErr) {
		return appErr, true
	}
	return nil, false
}

// From 把任意 error 归一化成业务错误，便于在响应层统一输出。
func From(err error) *Error {
	if err == nil {
		return nil
	}
	if appErr, ok := As(err); ok {
		return appErr
	}
	return Wrap(CodeInternal, "", err)
}
