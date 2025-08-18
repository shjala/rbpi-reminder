package main

import (
	"crypto/rand"
	"crypto/subtle"
	"encoding/base64"
	"errors"
	"fmt"
	"html/template"
	"io"
	"net/http"
	"os"
	"sync"
	"time"

	"golang.org/x/crypto/bcrypt"
	"gopkg.in/yaml.v2"
)

const (
	webServerPort     = "8080"
	webServerAddr     = "0.0.0.0"
	sessionTimeout    = 30 * time.Minute // Session expires after 30 minutes
	sessionCookieName = "simple_reminder_session"
)

type Session struct {
	id         string
	csrfToken  string
	created    time.Time
	lastAccess time.Time
}

// SessionManager manages user sessions
type SessionManager struct {
	sessions map[string]*Session
	mutex    sync.RWMutex
}

// newSessionManager creates a new session manager
func newSessionManager() *SessionManager {
	sm := &SessionManager{
		sessions: make(map[string]*Session),
	}

	// remove expired sessions periodically
	go sm.cleanupExpiredSessions()

	return sm
}

// createSession creates a new session and returns the session ID
func (sm *SessionManager) createSession() (string, error) {
	sm.mutex.Lock()
	defer sm.mutex.Unlock()

	// Generate a random session ID
	sessionBytes := make([]byte, 32)
	if _, err := rand.Read(sessionBytes); err != nil {
		return "", err
	}

	// Generate CSRF token
	csrfBytes := make([]byte, 32)
	if _, err := rand.Read(csrfBytes); err != nil {
		return "", err
	}

	sessionID := base64.URLEncoding.EncodeToString(sessionBytes)
	csrfToken := base64.URLEncoding.EncodeToString(csrfBytes)
	sm.sessions[sessionID] = &Session{
		id:         sessionID,
		csrfToken:  csrfToken,
		created:    time.Now(),
		lastAccess: time.Now(),
	}

	return sessionID, nil
}

// isValidSession checks if a session ID is valid and updates last access time
func (sm *SessionManager) isValidSession(sessionID string) bool {
	sm.mutex.Lock()
	defer sm.mutex.Unlock()

	session, exists := sm.sessions[sessionID]
	if !exists {
		return false
	}

	// Check if session has expired
	if time.Since(session.lastAccess) > sessionTimeout {
		delete(sm.sessions, sessionID)
		return false
	}

	session.lastAccess = time.Now()
	return true
}

// deleteSession removes a session
func (sm *SessionManager) deleteSession(sessionID string) {
	sm.mutex.Lock()
	defer sm.mutex.Unlock()
	delete(sm.sessions, sessionID)
}

// validateCSRFToken validates CSRF token for a session
func (sm *SessionManager) validateCSRFToken(sessionID, token string) bool {
	sm.mutex.RLock()
	defer sm.mutex.RUnlock()

	session, exists := sm.sessions[sessionID]
	if !exists {
		return false
	}

	return subtle.ConstantTimeCompare([]byte(session.csrfToken), []byte(token)) == 1
}

// getCSRFToken returns the CSRF token for a session
func (sm *SessionManager) getCSRFToken(sessionID string) (string, error) {
	sm.mutex.RLock()
	defer sm.mutex.RUnlock()

	session, exists := sm.sessions[sessionID]
	if !exists {
		return "", errors.New("session not found")
	}

	return session.csrfToken, nil
}

// cleanupExpiredSessions periodically removes expired sessions
func (sm *SessionManager) cleanupExpiredSessions() {
	ticker := time.NewTicker(5 * time.Minute)
	defer ticker.Stop()

	for range ticker.C {
		sm.mutex.Lock()
		for id, session := range sm.sessions {
			if time.Since(session.lastAccess) > sessionTimeout {
				delete(sm.sessions, id)
			}
		}
		sm.mutex.Unlock()
	}
}

// webServer handles the HTTP server for configuration management
type webServer struct {
	server         *http.Server
	sessionManager *SessionManager
	templates      *template.Template
}

// genericError returns a generic error message to avoid information disclosure
func genericError(w http.ResponseWriter, logMsg string, err error, statusCode int) {
	logError(logMsg+": %v", err)
	http.Error(w, "An internal error occurred", statusCode)
}

// validateFileSize validates file size
func validateFileSize(data []byte, maxSize int64) error {
	if int64(len(data)) > maxSize {
		return fmt.Errorf("file too large: %d bytes (max: %d)", len(data), maxSize)
	}
	return nil
}

// newWebServer creates a new web server instance
func newWebServer() *webServer {
	templates := template.Must(template.ParseGlob("web/templates/*.html"))
	return &webServer{
		sessionManager: newSessionManager(),
		templates:      templates,
	}
}

// start initializes and starts the web server
func (ws *webServer) start() error {
	mux := http.NewServeMux()

	// Static file serving
	fs := http.FileServer(http.Dir("web/static/"))
	mux.Handle("/static/", addSecurityHeaders(http.StripPrefix("/static/", fs).ServeHTTP))

	// Public endpoints (no authentication required)
	mux.HandleFunc("/login", addSecurityHeaders(ws.handleLogin))
	mux.HandleFunc("/logout", addSecurityHeaders(ws.handleLogout))

	// Protected endpoints (require authentication)
	mux.HandleFunc("/", addSecurityHeaders(ws.requireAuth(ws.handleIndex)))
	mux.HandleFunc("/api/csrf-token", addSecurityHeaders(ws.requireAuth(ws.handleCSRFToken)))
	mux.HandleFunc("/api/config", addSecurityHeaders(ws.requireAuth(ws.handleConfig)))
	mux.HandleFunc("/api/config/save", addSecurityHeaders(ws.requireAuth(ws.handleConfigSave)))
	mux.HandleFunc("/api/secrets", addSecurityHeaders(ws.requireAuth(ws.handleSecrets)))
	mux.HandleFunc("/api/secrets/save", addSecurityHeaders(ws.requireAuth(ws.handleSecretsSave)))
	mux.HandleFunc("/api/logs", addSecurityHeaders(ws.requireAuth(ws.handleLogs)))
	mux.HandleFunc("/api/logs/clear", addSecurityHeaders(ws.requireAuth(ws.handleLogsClear)))

	ws.server = &http.Server{
		Addr:    fmt.Sprintf("%s:%s", webServerAddr, webServerPort),
		Handler: mux,
	}

	logInfo("Starting web server on http://%s:%s", webServerAddr, webServerPort)
	return ws.server.ListenAndServe()
}

// Stop gracefully stops the web server
func (ws *webServer) Stop() error {
	if ws.server != nil {
		return ws.server.Close()
	}
	return nil
}

// addSecurityHeaders adds security headers to all responses
func addSecurityHeaders(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("X-Content-Type-Options", "nosniff")
		w.Header().Set("X-Frame-Options", "DENY")
		w.Header().Set("X-XSS-Protection", "1; mode=block")
		w.Header().Set("Referrer-Policy", "strict-origin-when-cross-origin")
		w.Header().Set("Content-Security-Policy", "default-src 'self'; script-src 'self' 'unsafe-inline'; style-src 'self' 'unsafe-inline'; img-src 'self' data:; font-src 'self'")
		next(w, r)
	}
}

// requireAuth is middleware that requires authentication
func (ws *webServer) requireAuth(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		cookie, err := r.Cookie(sessionCookieName)
		if err != nil {
			logDebug("No session cookie found for %s: %v", r.RemoteAddr, err)
			http.Redirect(w, r, "/login", http.StatusFound)
			return
		}

		if !ws.sessionManager.isValidSession(cookie.Value) {
			logDebug("Invalid session %s for %s", cookie.Value, r.RemoteAddr)
			http.Redirect(w, r, "/login", http.StatusFound)
			return
		}

		next(w, r)
	}
}

// handleLogin handles the login page and authentication
func (ws *webServer) handleLogin(w http.ResponseWriter, r *http.Request) {
	if r.Method == http.MethodPost {
		password := r.FormValue("password")

		// Get expected password from config or environment
		expectedPassword := os.Getenv("WEB_SERVER_PASSWORD")
		if expectedPassword == "" {
			expectedPassword = SysSecrets.WebServerPassword
		}

		if expectedPassword == "" {
			logError("No web server password configured")
			genericError(w, "Server configuration error", errors.New("no web server password configured"), http.StatusInternalServerError)
			return
		}

		err := bcrypt.CompareHashAndPassword([]byte(expectedPassword), []byte(password))
		if err == nil {
			// Password is correct, create session
			sessionID, err := ws.sessionManager.createSession()
			if err != nil {
				logError("Failed to create session: %v", err)
				http.Error(w, "Internal Server Error", http.StatusInternalServerError)
				return
			}

			// Set session cookie (httpOnly, secure flags for security)
			cookie := &http.Cookie{
				Name:     sessionCookieName,
				Value:    sessionID,
				Path:     "/",
				MaxAge:   int(sessionTimeout.Seconds()),
				HttpOnly: true,  // do not allow access to cookie from javascript
				Secure:   false, // TODO: fix this later use https
				SameSite: http.SameSiteStrictMode,
			}
			logInfo("User %s logged in", r.RemoteAddr)
			http.SetCookie(w, cookie)
			http.Redirect(w, r, "/", http.StatusFound)
			return
		} else {
			logError("Failed login attempt from %s", r.RemoteAddr)
			data := struct {
				ErrorMessage string
			}{
				ErrorMessage: "Invalid password. Please try again.",
			}
			w.WriteHeader(http.StatusUnauthorized)
			err := ws.templates.ExecuteTemplate(w, "login.html", data)
			if err != nil {
				logError("Template execution error: %v", err)
				http.Error(w, "Internal server error", http.StatusInternalServerError)
			}
			return
		}
	}

	// Show login form
	data := struct {
		ErrorMessage string
	}{
		ErrorMessage: "",
	}
	err := ws.templates.ExecuteTemplate(w, "login.html", data)
	if err != nil {
		logError("Template execution error: %v", err)
		http.Error(w, "Internal server error", http.StatusInternalServerError)
	}
}

// handleLogout handles user logout
func (ws *webServer) handleLogout(w http.ResponseWriter, r *http.Request) {
	cookie, err := r.Cookie(sessionCookieName)
	if err == nil {
		ws.sessionManager.deleteSession(cookie.Value)
	}

	// Clear session cookie
	clearCookie := &http.Cookie{
		Name:     sessionCookieName,
		Value:    "",
		Path:     "/",
		MaxAge:   -1,
		HttpOnly: true,
	}

	logInfo("User logged out from %s", r.RemoteAddr)
	http.SetCookie(w, clearCookie)
	http.Redirect(w, r, "/login", http.StatusFound)
}

// handleCSRFToken returns the CSRF token for the current session
func (ws *webServer) handleCSRFToken(w http.ResponseWriter, r *http.Request) {
	cookie, err := r.Cookie(sessionCookieName)
	if err != nil {
		http.Error(w, "No session found", http.StatusUnauthorized)
		return
	}

	csrfToken, err := ws.sessionManager.getCSRFToken(cookie.Value)
	if err != nil {
		genericError(w, "Failed to get CSRF token", err, http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "text/plain")
	w.Write([]byte(csrfToken))
}

// handleIndex serves the main configuration page
func (ws *webServer) handleIndex(w http.ResponseWriter, r *http.Request) {
	err := ws.templates.ExecuteTemplate(w, "index.html", nil)
	if err != nil {
		logError("Template execution error: %v", err)
		http.Error(w, "Internal server error", http.StatusInternalServerError)
	}
}

// handleConfig serves the current configuration as YAML
func (ws *webServer) handleConfig(w http.ResponseWriter, r *http.Request) {
	configPath := realPath(defaultConfig)
	configData, err := os.ReadFile(configPath)
	if err != nil {
		genericError(w, "Failed to read config file", err, http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "text/plain")
	w.Header().Set("X-Content-Type-Options", "nosniff")
	w.Write(configData)
}

// handleConfigSave saves the updated configuration
func (ws *webServer) handleConfigSave(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Validate CSRF token
	cookie, err := r.Cookie(sessionCookieName)
	if err != nil {
		http.Error(w, "Invalid session", http.StatusUnauthorized)
		return
	}

	csrfToken := r.Header.Get("X-CSRF-Token")
	if csrfToken == "" {
		csrfToken = r.FormValue("csrf_token")
	}

	if !ws.sessionManager.validateCSRFToken(cookie.Value, csrfToken) {
		logError("CSRF token validation failed for session %s", cookie.Value)
		http.Error(w, "Invalid CSRF token", http.StatusForbidden)
		return
	}

	// read config
	configData, err := io.ReadAll(r.Body)
	if err != nil {
		genericError(w, "Failed to read request body", err, http.StatusBadRequest)
		return
	}
	defer r.Body.Close()
	if err := validateFileSize(configData, 1024*1024); err != nil {
		logError("Config file too large: %v", err)
		http.Error(w, "File too large", http.StatusRequestEntityTooLarge)
		return
	}

	// Validate YAML before saving
	var tempConfig Config
	if err := yaml.Unmarshal(configData, &tempConfig); err != nil {
		logError("Invalid YAML in config save: %v", err)
		http.Error(w, "Invalid YAML format", http.StatusBadRequest)
		return
	}

	if err := writeFileAtomically(realPath(defaultConfig), configData); err != nil {
		logError("Failed to save config file: %v", err)
		http.Error(w, "Failed to save config file", http.StatusInternalServerError)
		return
	}

	// Reload configuration in memory
	if err := loadConfig(); err != nil {
		logError("Failed to reload configuration after web save: %v", err)
	}

	w.WriteHeader(http.StatusOK)
	w.Write([]byte("Configuration saved successfully"))
}

// handleSecrets serves the current secrets configuration as YAML
func (ws *webServer) handleSecrets(w http.ResponseWriter, r *http.Request) {
	secretsPath := realPath(defaultSecrets)
	secretsData, err := os.ReadFile(secretsPath)
	if err != nil {
		genericError(w, "Failed to read secrets file", err, http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "text/plain")
	w.Header().Set("X-Content-Type-Options", "nosniff")
	w.Write(secretsData)
}

// handleSecretsSave saves the updated secrets configuration
func (ws *webServer) handleSecretsSave(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Validate CSRF token
	cookie, err := r.Cookie(sessionCookieName)
	if err != nil {
		http.Error(w, "Invalid session", http.StatusUnauthorized)
		return
	}

	csrfToken := r.Header.Get("X-CSRF-Token")
	if csrfToken == "" {
		csrfToken = r.FormValue("csrf_token")
	}
	if !ws.sessionManager.validateCSRFToken(cookie.Value, csrfToken) {
		logError("CSRF token validation failed for session %s", cookie.Value)
		http.Error(w, "Invalid CSRF token", http.StatusForbidden)
		return
	}

	// read config
	secretsData, err := io.ReadAll(r.Body)
	if err != nil {
		genericError(w, "Failed to read request body", err, http.StatusBadRequest)
		return
	}
	defer r.Body.Close()
	if err := validateFileSize(secretsData, 1024*1024); err != nil {
		logError("Secrets file too large: %v", err)
		http.Error(w, "File too large", http.StatusRequestEntityTooLarge)
		return
	}

	// Validate YAML before saving
	var tempSecrets Secrets
	if err := yaml.Unmarshal(secretsData, &tempSecrets); err != nil {
		logError("Invalid YAML in secrets save: %v", err)
		http.Error(w, "Invalid YAML format", http.StatusBadRequest)
		return
	}

	if err := writeFileAtomically(realPath(defaultSecrets), secretsData); err != nil {
		logError("Failed to save config file: %v", err)
		http.Error(w, "Failed to save config file", http.StatusInternalServerError)
		return
	}

	// Reload configuration in memory
	if err := loadConfig(); err != nil {
		logError("Failed to reload configuration after web save: %v", err)
	}

	w.WriteHeader(http.StatusOK)
	w.Write([]byte("Secrets saved successfully"))
}

// handleLogs serves the current logs
func (ws *webServer) handleLogs(w http.ResponseWriter, r *http.Request) {
	logFilePath := realPath(logPath)

	// Read current log file
	currentLogs := ""
	if data, err := os.ReadFile(logFilePath); err == nil {
		currentLogs = string(data)
	}

	// Also try to read the old rotated log file if it exists
	oldLogPath := logFilePath + ".old"
	if data, err := os.ReadFile(oldLogPath); err == nil {
		// Prepend old logs with a separator
		if currentLogs != "" {
			currentLogs = string(data) + "\n--- LOG ROTATION ---\n" + currentLogs
		} else {
			currentLogs = string(data)
		}
	}

	// If no logs found, show a message
	if currentLogs == "" {
		currentLogs = "No logs found or logs are empty."
	}

	w.Header().Set("Content-Type", "text/plain")
	w.Header().Set("X-Content-Type-Options", "nosniff")
	w.Write([]byte(currentLogs))
}

// handleLogsClear clears the application logs
func (ws *webServer) handleLogsClear(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Validate CSRF token
	cookie, err := r.Cookie(sessionCookieName)
	if err != nil {
		http.Error(w, "Invalid session", http.StatusUnauthorized)
		return
	}

	csrfToken := r.Header.Get("X-CSRF-Token")
	if csrfToken == "" {
		csrfToken = r.FormValue("csrf_token")
	}
	if !ws.sessionManager.validateCSRFToken(cookie.Value, csrfToken) {
		logError("CSRF token validation failed for session %s", cookie.Value)
		http.Error(w, "Invalid CSRF token", http.StatusForbidden)
		return
	}

	logFilePath := realPath(logPath)
	oldLogPath := logFilePath + ".old"

	// Clear current log file
	if err := os.Truncate(logFilePath, 0); err != nil {
		logError("Failed to clear log file: %v", err)
		http.Error(w, "Failed to clear logs", http.StatusInternalServerError)
		return
	}

	// Remove old log file if it exists
	if _, err := os.Stat(oldLogPath); err == nil {
		if err := os.Remove(oldLogPath); err != nil {
			logError("Failed to remove old log file: %v", err)
			// Don't return error here, clearing current log is more important
		}
	}

	logInfo("Logs cleared by user from %s", r.RemoteAddr)
	w.WriteHeader(http.StatusOK)
	w.Write([]byte("Logs cleared successfully"))
}

// setupWebServer initializes and starts the web server in a goroutine
func setupWebServer() {
	webServer := newWebServer()

	// Start web server in background
	go func() {
		if err := webServer.start(); err != nil && err != http.ErrServerClosed {
			logError("Web server error: %v", err)
		}
	}()

	logInfo("Configuration web interface available at http://%s:%s", webServerAddr, webServerPort)
}
