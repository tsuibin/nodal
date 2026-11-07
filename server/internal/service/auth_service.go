// Package service 承载业务逻辑：只依赖 GORM 与 token 管理器，不感知 HTTP。
package service

import (
	"context"
	"encoding/json"
	"errors"
	"log/slog"
	"strings"
	"time"

	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"

	"github.com/nodal/nodal-server/internal/apperr"
	"github.com/nodal/nodal-server/internal/model"
	"github.com/nodal/nodal-server/internal/token"
)

// AuthService 处理注册、登录、当前用户、资料修改与游客会话。
type AuthService struct {
	db           *gorm.DB
	tokens       *token.Manager
	guestEnabled bool
	guestTTL     time.Duration
	now          func() time.Time
}

func NewAuthService(db *gorm.DB, tokens *token.Manager, guestEnabled bool, guestTTL time.Duration) *AuthService {
	return &AuthService{
		db:           db,
		tokens:       tokens,
		guestEnabled: guestEnabled,
		guestTTL:     guestTTL,
		now:          time.Now,
	}
}

// ---------- 入参 ----------

// Email 在反序列化时统一去空格并转小写：
// 用户在表单里粘贴「 A@Example.com 」不会被 email 校验误判，入库也天然大小写不敏感。
type Email string

func (e *Email) UnmarshalJSON(data []byte) error {
	var raw string
	if err := json.Unmarshal(data, &raw); err != nil {
		return err
	}
	*e = Email(strings.ToLower(strings.TrimSpace(raw)))
	return nil
}

func (e Email) String() string { return string(e) }

type RegisterInput struct {
	Email    Email  `json:"email" binding:"required,email,max=190"`
	Password string `json:"password" binding:"required,min=8,max=72"`
	Nickname string `json:"nickname" binding:"omitempty,max=32"`
}

type LoginInput struct {
	Email    Email  `json:"email" binding:"required,email,max=190"`
	Password string `json:"password" binding:"required,min=1,max=72"`
}

// UpdateProfileInput 使用指针字段区分「未传」与「显式置空」。
type UpdateProfileInput struct {
	Nickname  *string `json:"nickname" binding:"omitempty,max=32"`
	AvatarURL *string `json:"avatar_url" binding:"omitempty,max=512,url"`
}

// ---------- 出参 ----------

type UserView struct {
	ID        uint      `json:"id"`
	Email     string    `json:"email"`
	Nickname  string    `json:"nickname"`
	AvatarURL string    `json:"avatar_url"`
	CreatedAt time.Time `json:"created_at"`
}

type GuestView struct {
	ID        string    `json:"id"`
	ExpiresAt time.Time `json:"expires_at"`
}

// Session 是注册/登录/游客发放后的返回体。
type Session struct {
	Token     string     `json:"token"`
	TokenType string     `json:"token_type"`
	ExpiresAt time.Time  `json:"expires_at"`
	IsGuest   bool       `json:"is_guest"`
	User      *UserView  `json:"user,omitempty"`
	Guest     *GuestView `json:"guest,omitempty"`
}

// Profile 是 /auth/me 的返回体：用户与游客二选一。
type Profile struct {
	IsGuest bool       `json:"is_guest"`
	User    *UserView  `json:"user,omitempty"`
	Guest   *GuestView `json:"guest,omitempty"`
}

// ---------- 注册 / 登录 ----------

func (s *AuthService) Register(ctx context.Context, in RegisterInput) (*Session, error) {
	email := normalizeEmail(string(in.Email))
	if email == "" {
		return nil, apperr.New(apperr.CodeInvalidParam, "邮箱不能为空")
	}

	hash, err := bcrypt.GenerateFromPassword([]byte(in.Password), bcrypt.DefaultCost)
	if err != nil {
		return nil, apperr.Wrap(apperr.CodeInternal, "密码加密失败", err)
	}

	user := &model.User{
		Email:        email,
		PasswordHash: string(hash),
		Nickname:     defaultNickname(in.Nickname, email),
		Status:       model.UserStatusActive,
	}

	if err := s.db.WithContext(ctx).Create(user).Error; err != nil {
		if errors.Is(err, gorm.ErrDuplicatedKey) {
			return nil, apperr.New(apperr.CodeEmailTaken, "")
		}
		return nil, apperr.Wrap(apperr.CodeInternal, "创建用户失败", err)
	}

	slog.Info("user registered", "user_id", user.ID, "email", user.Email)
	return s.issueUserSession(user)
}

func (s *AuthService) Login(ctx context.Context, in LoginInput) (*Session, error) {
	email := normalizeEmail(string(in.Email))

	var user model.User
	err := s.db.WithContext(ctx).Where("email = ?", email).First(&user).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			// 不区分「邮箱不存在」和「密码错误」，避免账号枚举。
			return nil, apperr.New(apperr.CodeBadCredentials, "")
		}
		return nil, apperr.Wrap(apperr.CodeInternal, "查询用户失败", err)
	}

	if err := bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(in.Password)); err != nil {
		return nil, apperr.New(apperr.CodeBadCredentials, "")
	}
	if !user.IsActive() {
		return nil, apperr.New(apperr.CodeAccountDisabled, "")
	}

	slog.Info("user logged in", "user_id", user.ID)
	return s.issueUserSession(&user)
}

func (s *AuthService) issueUserSession(user *model.User) (*Session, error) {
	sessionID := token.NewID()
	tk, err := s.tokens.IssueUser(user.ID, sessionID)
	if err != nil {
		return nil, err
	}
	return &Session{
		Token:     tk.Value,
		TokenType: "Bearer",
		ExpiresAt: tk.ExpiresAt,
		IsGuest:   false,
		User:      userView(user),
	}, nil
}

// ---------- 当前用户 / 资料 ----------

func (s *AuthService) Me(ctx context.Context, id *token.Identity) (*Profile, error) {
	switch {
	case id.IsUser():
		var user model.User
		err := s.db.WithContext(ctx).First(&user, id.UserID).Error
		if err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return nil, apperr.New(apperr.CodeUnauthorized, "账号不存在或已注销，请重新登录")
			}
			return nil, apperr.Wrap(apperr.CodeInternal, "查询用户失败", err)
		}
		if !user.IsActive() {
			return nil, apperr.New(apperr.CodeAccountDisabled, "")
		}
		return &Profile{IsGuest: false, User: userView(&user)}, nil

	case id.IsGuest():
		guest, err := s.ensureGuestSession(ctx, id)
		if err != nil {
			return nil, err
		}
		return &Profile{IsGuest: true, Guest: guestView(guest)}, nil

	default:
		return nil, apperr.New(apperr.CodeUnauthorized, "")
	}
}

func (s *AuthService) UpdateProfile(ctx context.Context, id *token.Identity, in UpdateProfileInput) (*Profile, error) {
	if id.IsGuest() {
		return nil, apperr.New(apperr.CodeForbidden, "游客无法修改资料，请先注册账号")
	}
	if !id.IsUser() {
		return nil, apperr.New(apperr.CodeUnauthorized, "")
	}

	updates := map[string]any{}
	if in.Nickname != nil {
		nickname := strings.TrimSpace(*in.Nickname)
		if nickname == "" {
			return nil, apperr.New(apperr.CodeInvalidParam, "昵称不能为空")
		}
		updates["nickname"] = nickname
	}
	if in.AvatarURL != nil {
		updates["avatar_url"] = strings.TrimSpace(*in.AvatarURL)
	}
	if len(updates) == 0 {
		return s.Me(ctx, id)
	}

	var user model.User
	err := s.db.WithContext(ctx).First(&user, id.UserID).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, apperr.New(apperr.CodeUnauthorized, "账号不存在或已注销，请重新登录")
		}
		return nil, apperr.Wrap(apperr.CodeInternal, "查询用户失败", err)
	}
	if !user.IsActive() {
		return nil, apperr.New(apperr.CodeAccountDisabled, "")
	}

	if err := s.db.WithContext(ctx).Model(&user).Updates(updates).Error; err != nil {
		return nil, apperr.Wrap(apperr.CodeInternal, "保存资料失败", err)
	}

	slog.Info("user profile updated", "user_id", user.ID, "fields", mapKeys(updates))
	return s.Me(ctx, id)
}

// ---------- 游客临时会话 ----------

// CreateGuestSession 显式发放游客会话（POST /api/auth/guest）。
func (s *AuthService) CreateGuestSession(ctx context.Context, clientIP, userAgent string) (*Session, error) {
	if !s.guestEnabled {
		return nil, apperr.New(apperr.CodeGuestDisabled, "")
	}

	now := s.now().UTC()
	guest := &model.GuestSession{
		ID:         token.NewID(),
		ClientIP:   truncate(clientIP, 64),
		UserAgent:  truncate(userAgent, 255),
		CreatedAt:  now,
		ExpiresAt:  now.Add(s.guestTTL),
		LastSeenAt: now,
	}
	if err := s.db.WithContext(ctx).Create(guest).Error; err != nil {
		return nil, apperr.Wrap(apperr.CodeInternal, "创建游客会话失败", err)
	}

	tk, err := s.tokens.IssueGuest(guest.ID, token.NewID())
	if err != nil {
		return nil, err
	}
	return &Session{
		Token:     tk.Value,
		TokenType: "Bearer",
		ExpiresAt: tk.ExpiresAt,
		IsGuest:   true,
		Guest:     guestView(guest),
	}, nil
}

// ensureGuestSession 保证 token 里的游客 ID 在库里有会话记录：
// 中间件为匿名请求即时签发的游客身份，第一次访问 /auth/me 时在这里落库。
func (s *AuthService) ensureGuestSession(ctx context.Context, id *token.Identity) (*model.GuestSession, error) {
	if !s.guestEnabled {
		return nil, apperr.New(apperr.CodeGuestDisabled, "")
	}

	var guest model.GuestSession
	err := s.db.WithContext(ctx).Where("id = ?", id.GuestID).First(&guest).Error
	switch {
	case err == nil:
		if guest.Expired(s.now().UTC()) {
			return nil, apperr.New(apperr.CodeGuestExpired, "")
		}
		// 节流更新 last_seen_at，避免每个请求都写库。
		now := s.now().UTC()
		if now.Sub(guest.LastSeenAt) > time.Minute {
			s.db.WithContext(ctx).Model(&model.GuestSession{}).
				Where("id = ?", guest.ID).Update("last_seen_at", now)
			guest.LastSeenAt = now
		}
		return &guest, nil

	case errors.Is(err, gorm.ErrRecordNotFound):
		now := s.now().UTC()
		guest = model.GuestSession{
			ID:         id.GuestID,
			CreatedAt:  now,
			ExpiresAt:  now.Add(s.guestTTL),
			LastSeenAt: now,
		}
		if createErr := s.db.WithContext(ctx).Create(&guest).Error; createErr != nil {
			// 并发下可能被另一个请求抢先创建，重查一次即可。
			if errors.Is(createErr, gorm.ErrDuplicatedKey) {
				if e := s.db.WithContext(ctx).Where("id = ?", id.GuestID).First(&guest).Error; e == nil {
					return &guest, nil
				}
			}
			return nil, apperr.Wrap(apperr.CodeInternal, "创建游客会话失败", createErr)
		}
		return &guest, nil

	default:
		return nil, apperr.Wrap(apperr.CodeInternal, "查询游客会话失败", err)
	}
}

// PurgeExpiredGuests 清理过期游客会话，可由定时任务调用。
func (s *AuthService) PurgeExpiredGuests(ctx context.Context, limit int) (int64, error) {
	if limit <= 0 {
		limit = 500
	}
	var ids []string
	err := s.db.WithContext(ctx).Model(&model.GuestSession{}).
		Where("expires_at < ?", s.now().UTC()).
		Limit(limit).
		Pluck("id", &ids).Error
	if err != nil {
		return 0, apperr.Wrap(apperr.CodeInternal, "查询过期游客会话失败", err)
	}
	if len(ids) == 0 {
		return 0, nil
	}
	res := s.db.WithContext(ctx).Where("id IN ?", ids).Delete(&model.GuestSession{})
	return res.RowsAffected, res.Error
}

// ---------- helpers ----------

func userView(u *model.User) *UserView {
	return &UserView{
		ID:        u.ID,
		Email:     u.Email,
		Nickname:  u.Nickname,
		AvatarURL: u.AvatarURL,
		CreatedAt: u.CreatedAt,
	}
}

func guestView(g *model.GuestSession) *GuestView {
	return &GuestView{ID: g.ID, ExpiresAt: g.ExpiresAt}
}

func normalizeEmail(s string) string { return strings.ToLower(strings.TrimSpace(s)) }

func defaultNickname(nickname, email string) string {
	if n := strings.TrimSpace(nickname); n != "" {
		return n
	}
	if at := strings.IndexByte(email, '@'); at > 0 {
		return email[:at]
	}
	return "Nodal 用户"
}

func truncate(s string, max int) string {
	s = strings.TrimSpace(s)
	if len(s) <= max {
		return s
	}
	return s[:max]
}

func mapKeys(m map[string]any) []string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	return keys
}
