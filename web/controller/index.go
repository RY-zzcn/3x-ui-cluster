package controller

import (
	"net/http"
	"sync"
	"text/template"
	"time"

	"github.com/mhsanaei/3x-ui/v2/logger"
	"github.com/mhsanaei/3x-ui/v2/web/service"
	"github.com/mhsanaei/3x-ui/v2/web/session"

	"github.com/gin-contrib/sessions"
	"github.com/gin-gonic/gin"
)

var (
	// Rate limiting for login attempts
	loginAttempts = make(map[string]*loginAttemptTracker)
	loginMutex    sync.RWMutex
)

type loginAttemptTracker struct {
	count      int
	lastAttempt time.Time
	lockedUntil time.Time
}

const (
	maxLoginAttempts = 5
	lockoutDuration  = 15 * time.Minute
	attemptWindow    = 5 * time.Minute
)

// LoginForm represents the login request structure.
type LoginForm struct {
	Username      string `json:"username" form:"username"`
	Password      string `json:"password" form:"password"`
	TwoFactorCode string `json:"twoFactorCode" form:"twoFactorCode"`
}

// IndexController handles the main index and login-related routes.
type IndexController struct {
	BaseController

	settingService service.SettingService
	userService    service.UserService
	tgbot          service.Tgbot
}

// NewIndexController creates a new IndexController and initializes its routes.
func NewIndexController(g *gin.RouterGroup) *IndexController {
	a := &IndexController{}
	a.initRouter(g)
	return a
}

// initRouter sets up the routes for index, login, logout, and two-factor authentication.
func (a *IndexController) initRouter(g *gin.RouterGroup) {
	g.GET("/", a.index)
	g.GET("/logout", a.logout)

	g.POST("/login", a.login)
	g.POST("/getTwoFactorEnable", a.getTwoFactorEnable)
}

// index handles the root route, redirecting logged-in users to the panel or showing the login page.
func (a *IndexController) index(c *gin.Context) {
	if session.IsLogin(c) {
		c.Redirect(http.StatusTemporaryRedirect, "panel/")
		return
	}
	html(c, "login.html", "pages.login.title", nil)
}

// login handles user authentication and session creation.
func (a *IndexController) login(c *gin.Context) {
	var form LoginForm

	if err := c.ShouldBind(&form); err != nil {
		pureJsonMsg(c, http.StatusOK, false, I18nWeb(c, "pages.login.toasts.invalidFormData"))
		return
	}
	if form.Username == "" {
		pureJsonMsg(c, http.StatusOK, false, I18nWeb(c, "pages.login.toasts.emptyUsername"))
		return
	}
	if form.Password == "" {
		pureJsonMsg(c, http.StatusOK, false, I18nWeb(c, "pages.login.toasts.emptyPassword"))
		return
	}

	// Rate limiting check
	clientIP := getRemoteIp(c)
	if isRateLimited(clientIP) {
		logger.Warningf("Login rate limited for IP: %s", clientIP)
		pureJsonMsg(c, http.StatusTooManyRequests, false, I18nWeb(c, "pages.login.toasts.tooManyAttempts"))
		return
	}

	user := a.userService.CheckUser(form.Username, form.Password, form.TwoFactorCode)
	timeStr := time.Now().Format("2006-01-02 15:04:05")
	safeUser := template.HTMLEscapeString(form.Username)

	if user == nil {
		// Record failed attempt
		recordLoginAttempt(clientIP, false)
		
		// Do not log password - security risk
		logger.Warningf("Failed login attempt for username: \"%s\", IP: \"%s\"", safeUser, clientIP)
		a.tgbot.UserLoginNotify(safeUser, "***", clientIP, timeStr, 0)
		pureJsonMsg(c, http.StatusOK, false, I18nWeb(c, "pages.login.toasts.wrongUsernameOrPassword"))
		return
	}

	// Successful login - clear attempts
	recordLoginAttempt(clientIP, true)

	logger.Infof("%s logged in successfully, Ip Address: %s\n", safeUser, clientIP)
	a.tgbot.UserLoginNotify(safeUser, ``, clientIP, timeStr, 1)

	sessionMaxAge, err := a.settingService.GetSessionMaxAge()
	if err != nil {
		logger.Warning("Unable to get session's max age from DB")
	}

	session.SetMaxAge(c, sessionMaxAge*60)
	session.SetLoginUser(c, user)
	if err := sessions.Default(c).Save(); err != nil {
		logger.Warning("Unable to save session: ", err)
		return
	}

	logger.Infof("%s logged in successfully", safeUser)
	jsonMsg(c, I18nWeb(c, "pages.login.toasts.successLogin"), nil)
}

// logout handles user logout by clearing the session and redirecting to the login page.
func (a *IndexController) logout(c *gin.Context) {
	user := session.GetLoginUser(c)
	if user != nil {
		logger.Infof("%s logged out successfully", user.Username)
	}
	session.ClearSession(c)
	if err := sessions.Default(c).Save(); err != nil {
		logger.Warning("Unable to save session after clearing:", err)
	}
	c.Redirect(http.StatusTemporaryRedirect, c.GetString("base_path"))
}

// getTwoFactorEnable retrieves the current status of two-factor authentication.
func (a *IndexController) getTwoFactorEnable(c *gin.Context) {
	status, err := a.settingService.GetTwoFactorEnable()
	if err == nil {
		jsonObj(c, status, nil)
	}
}

// isRateLimited checks if an IP is currently rate limited
func isRateLimited(ip string) bool {
	loginMutex.RLock()
	tracker, exists := loginAttempts[ip]
	loginMutex.RUnlock()

	if !exists {
		return false
	}

	// Check if still locked out
	if time.Now().Before(tracker.lockedUntil) {
		return true
	}

	// Check if within attempt window
	if time.Since(tracker.lastAttempt) > attemptWindow {
		return false
	}

	return tracker.count >= maxLoginAttempts
}

// recordLoginAttempt records a login attempt for rate limiting
func recordLoginAttempt(ip string, success bool) {
	loginMutex.Lock()
	defer loginMutex.Unlock()

	if success {
		// Clear attempts on successful login
		delete(loginAttempts, ip)
		return
	}

	tracker, exists := loginAttempts[ip]
	if !exists {
		tracker = &loginAttemptTracker{}
		loginAttempts[ip] = tracker
	}

	// Reset counter if outside attempt window
	if time.Since(tracker.lastAttempt) > attemptWindow {
		tracker.count = 0
	}

	tracker.count++
	tracker.lastAttempt = time.Now()

	// Lock out if max attempts reached
	if tracker.count >= maxLoginAttempts {
		tracker.lockedUntil = time.Now().Add(lockoutDuration)
		logger.Warningf("IP %s locked out for %v after %d failed attempts", ip, lockoutDuration, tracker.count)
	}

	// Cleanup old entries periodically
	if len(loginAttempts) > 10000 {
		for k, v := range loginAttempts {
			if time.Since(v.lastAttempt) > 24*time.Hour {
				delete(loginAttempts, k)
			}
		}
	}
}
