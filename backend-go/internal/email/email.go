package email

import (
	"crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"
	"os"
	"strings"
	"sync"
	"time"

	mail "github.com/go-mail/mail/v2"
)

type Code struct {
	Email     string
	Code      string
	Attempts  int
	ExpiresAt time.Time
}

type Store struct {
	m           sync.Map
	TTL         time.Duration
	MaxAttempts int
}

func NewStore(ttl time.Duration, maxAttempts int) *Store {
	return &Store{TTL: ttl, MaxAttempts: maxAttempts}
}

func randomID() string { b := make([]byte, 16); rand.Read(b); return hex.EncodeToString(b) }

func randomCode(n int) string {
	b := make([]byte, n)
	rand.Read(b)
	return strings.ToUpper(hex.EncodeToString(b)[:n])
}

func (s *Store) Generate(email string, length int) (id, code string) {
	id = randomID()
	code = randomCode(length)
	s.m.Store(id, &Code{Email: email, Code: code, Attempts: 0, ExpiresAt: time.Now().Add(s.TTL)})
	return
}

func (s *Store) Verify(id, email, input string) error {
	v, ok := s.m.Load(id)
	if !ok {
		return errors.New("invalid or expired code")
	}
	c := v.(*Code)
	if time.Now().After(c.ExpiresAt) {
		s.m.Delete(id)
		return errors.New("code expired")
	}
	if c.Email != email {
		return errors.New("email mismatch")
	}
	if c.Attempts >= s.MaxAttempts {
		s.m.Delete(id)
		return errors.New("too many attempts")
	}
	c.Attempts++
	if strings.ToUpper(input) != c.Code {
		return errors.New("invalid code")
	}
	s.m.Delete(id)
	return nil
}

func (s *Store) Cleanup() {
	now := time.Now()
	s.m.Range(func(key, value any) bool {
		c := value.(*Code)
		if now.After(c.ExpiresAt) {
			s.m.Delete(key)
		}
		return true
	})
}

func Send(to, code string) error {
	host := os.Getenv("EMAIL_HOST")
	user := os.Getenv("EMAIL_USER")
	pass := os.Getenv("EMAIL_PASS")
	port := os.Getenv("EMAIL_PORT")
	if host == "" || user == "" || pass == "" {
		return fmt.Errorf("email config incomplete")
	}
	p := 587
	fmt.Sscanf(port, "%d", &p)
	d := mail.NewDialer(host, p, user, pass)
	m := mail.NewMessage()
	from := os.Getenv("EMAIL_FROM")
	if from == "" {
		from = user
	}
	m.SetHeader("From", from)
	m.SetHeader("To", to)
	m.SetHeader("Subject", "TodoIng 邮箱验证码")
	m.SetBody("text/html", fmt.Sprintf("<p>您的验证码: <b>%s</b> (10分钟内有效)</p>", code))
	return d.DialAndSend(m)
}

// SendGeneric 发送通用邮件
// SendGeneric 发送通用邮件（提醒等）。
// 优先使用 REMINDER_EMAIL_* 一组变量；如果不完整，则回退到 EMAIL_* 验证码登录邮箱配置。
// 变量说明：
// REMINDER_EMAIL_HOST / USER / PASS / PORT / FROM （可选）
// EMAIL_HOST / USER / PASS / PORT / FROM （登录验证码原有配置，作为回退）
func SendGeneric(to, subject, htmlBody string) error {
	// 优先读取提醒专用配置
	host := os.Getenv("REMINDER_EMAIL_HOST")
	user := os.Getenv("REMINDER_EMAIL_USER")
	pass := os.Getenv("REMINDER_EMAIL_PASS")
	port := os.Getenv("REMINDER_EMAIL_PORT")
	from := os.Getenv("REMINDER_EMAIL_FROM")

	// 如果任一核心字段缺失则回退到通用邮箱配置
	if host == "" || user == "" || pass == "" {
		host = os.Getenv("EMAIL_HOST")
		user = os.Getenv("EMAIL_USER")
		pass = os.Getenv("EMAIL_PASS")
		port = os.Getenv("EMAIL_PORT")
		if from == "" { // 只有在专用 FROM 未设置时才尝试通用 FROM
			from = os.Getenv("EMAIL_FROM")
		}
	}

	if host == "" || user == "" || pass == "" { // 仍不完整
		return fmt.Errorf("email config incomplete (reminder & fallback both missing)")
	}

	if port == "" { // 允许未提供端口，默认 587
		port = "587"
	}

	p := 587
	fmt.Sscanf(port, "%d", &p)
	d := mail.NewDialer(host, p, user, pass)
	m := mail.NewMessage()
	if from == "" {
		from = user
	}
	m.SetHeader("From", from)
	m.SetHeader("To", to)
	m.SetHeader("Subject", subject)
	m.SetBody("text/html", htmlBody)
	return d.DialAndSend(m)
}
