// Package handler 只做 HTTP 适配：绑定参数、取身份、调用 service、把错误交给统一出口。
package handler

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/go-playground/validator/v10"

	"github.com/nodal/nodal-server/internal/apperr"
	"github.com/nodal/nodal-server/internal/middleware"
	"github.com/nodal/nodal-server/internal/response"
	"github.com/nodal/nodal-server/internal/service"
)

type AuthHandler struct {
	auth *service.AuthService
}

func NewAuthHandler(auth *service.AuthService) *AuthHandler {
	return &AuthHandler{auth: auth}
}

// Register POST /api/auth/register
func (h *AuthHandler) Register(c *gin.Context) {
	var in service.RegisterInput
	if err := c.ShouldBindJSON(&in); err != nil {
		c.Error(bindError(err))
		return
	}
	session, err := h.auth.Register(c.Request.Context(), in)
	if err != nil {
		c.Error(err)
		return
	}
	response.OK(c, session)
}

// Login POST /api/auth/login
func (h *AuthHandler) Login(c *gin.Context) {
	var in service.LoginInput
	if err := c.ShouldBindJSON(&in); err != nil {
		c.Error(bindError(err))
		return
	}
	session, err := h.auth.Login(c.Request.Context(), in)
	if err != nil {
		c.Error(err)
		return
	}
	response.OK(c, session)
}

// Me GET /api/auth/me —— 用户与游客都可访问，用于前端启动时恢复会话。
func (h *AuthHandler) Me(c *gin.Context) {
	id, err := middleware.MustIdentity(c)
	if err != nil {
		c.Error(err)
		return
	}
	profile, err := h.auth.Me(c.Request.Context(), id)
	if err != nil {
		c.Error(err)
		return
	}
	response.OK(c, profile)
}

// UpdateProfile PUT /api/auth/profile —— 仅注册用户。
func (h *AuthHandler) UpdateProfile(c *gin.Context) {
	id, err := middleware.MustIdentity(c)
	if err != nil {
		c.Error(err)
		return
	}

	var in service.UpdateProfileInput
	if err := c.ShouldBindJSON(&in); err != nil {
		c.Error(bindError(err))
		return
	}

	profile, err := h.auth.UpdateProfile(c.Request.Context(), id, in)
	if err != nil {
		c.Error(err)
		return
	}
	response.OK(c, profile)
}

// Guest POST /api/auth/guest —— 显式领取游客临时会话（临时画布入口）。
func (h *AuthHandler) Guest(c *gin.Context) {
	session, err := h.auth.CreateGuestSession(c.Request.Context(), c.ClientIP(), c.Request.UserAgent())
	if err != nil {
		c.Error(err)
		return
	}
	c.Header(middleware.GuestTokenHeader, session.Token)
	response.OK(c, session)
}

func bindError(err error) error {
	var ve validator.ValidationErrors
	if errors.As(err, &ve) {
		fields := make(map[string]string, len(ve))
		for _, fe := range ve {
			fields[toSnake(fe.Field())] = validationMessage(fe)
		}
		return apperr.New(apperr.CodeInvalidParam, "请求参数不正确").WithDetails(fields)
	}

	var syntaxErr *json.SyntaxError
	var typeErr *json.UnmarshalTypeError
	var maxBytesErr *http.MaxBytesError
	switch {
	case errors.As(err, &maxBytesErr):
		return apperr.Newf(apperr.CodeInvalidParam, "请求体过大（上限 %d MB）", maxBytesErr.Limit>>20)
	case errors.Is(err, io.EOF):
		return apperr.New(apperr.CodeInvalidParam, "请求体不能为空")
	case errors.As(err, &syntaxErr), errors.As(err, &typeErr):
		return apperr.New(apperr.CodeInvalidParam, "请求体格式不正确，请检查 JSON 字段")
	default:
		return apperr.New(apperr.CodeInvalidParam, "请求参数不正确")
	}
}

func validationMessage(fe validator.FieldError) string {
	switch fe.Tag() {
	case "required":
		return "不能为空"
	case "email":
		return "邮箱格式不正确"
	case "url":
		return "必须是合法的 URL"
	case "min":
		return fmt.Sprintf("长度不能少于 %s 个字符", fe.Param())
	case "max":
		return fmt.Sprintf("长度不能超过 %s 个字符", fe.Param())
	default:
		return "格式不正确"
	}
}

// toSnake 把 AvatarURL 这类字段名转成 avatar_url，方便前端定位字段。
func toSnake(s string) string {
	var b strings.Builder
	for i, r := range s {
		if r >= 'A' && r <= 'Z' {
			if i > 0 {
				b.WriteByte('_')
			}
			b.WriteRune(r + ('a' - 'A'))
			continue
		}
		b.WriteRune(r)
	}
	return b.String()
}
