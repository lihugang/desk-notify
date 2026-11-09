package auth

import (
	"context"
	"net"
	"net/http"
	"strings"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/lihugang/desk-notify/core/pkg/config"
	"golang.org/x/crypto/bcrypt"
)

type contextKey string

const (
	tunnelContextKey contextKey = "desk_notify_from_tunnel"
)

// WithTunnelContext marks the request context as originating from the reverse proxy / tunnel.
func WithTunnelContext(ctx context.Context) context.Context {
	return context.WithValue(ctx, tunnelContextKey, true)
}

// IsTunnelRequest checks if the request was received via the internal tunnel listener.
func IsTunnelRequest(r *http.Request) bool {
	if r == nil {
		return false
	}
	val, ok := r.Context().Value(tunnelContextKey).(bool)
	return ok && val
}

var (
	privateIPBlocks []*net.IPNet
	jwtSecretKey    = []byte("desk-notify-default-jwt-secret-key-32")
	adminSecretKey  = []byte("desk-notify-admin-auth-secret-key-32")
)

func init() {
	for _, cidr := range []string{
		"127.0.0.0/8",    // IPv4 loopback
		"10.0.0.0/8",     // RFC1918
		"172.16.0.0/12",  // RFC1918
		"192.168.0.0/16", // RFC1918
		"::1/128",        // IPv6 loopback
		"fc00::/7",       // IPv6 unique local addr
		"fe80::/10",      // IPv6 link-local
	} {
		_, block, err := net.ParseCIDR(cidr)
		if err == nil {
			privateIPBlocks = append(privateIPBlocks, block)
		}
	}
}

func isPrivateIP(ip net.IP) bool {
	if ip == nil {
		return false
	}
	for _, block := range privateIPBlocks {
		if block.Contains(ip) {
			return true
		}
	}
	return false
}

// isLoopbackHost checks whether host (or host:port) resolves to loopback (localhost, 127.0.0.1, ::1).
func isLoopbackHost(host string) bool {
	h, _, err := net.SplitHostPort(host)
	if err != nil {
		h = host
	}
	h = strings.Trim(strings.TrimSpace(h), "[]")
	if strings.EqualFold(h, "localhost") || strings.EqualFold(h, "example.com") {
		return true
	}
	ip := net.ParseIP(h)
	return ip != nil && ip.IsLoopback()
}

// isPrivateHost checks whether host (or host:port) is loopback, LAN private IP, or .local mDNS.
func isPrivateHost(host string) bool {
	h, _, err := net.SplitHostPort(host)
	if err != nil {
		h = host
	}
	h = strings.Trim(strings.TrimSpace(h), "[]")
	if isLoopbackHost(h) {
		return true
	}
	if strings.HasSuffix(strings.ToLower(h), ".local") {
		return true
	}
	ip := net.ParseIP(h)
	if ip != nil && isPrivateIP(ip) {
		return true
	}
	return false
}

// IsPublicRequest checks whether the incoming request arrived via frp / public network.
func IsPublicRequest(r *http.Request) bool {
	if r == nil {
		return false
	}

	// 1. If marked as incoming from internal tunnel listener, it is public
	if IsTunnelRequest(r) {
		return true
	}

	// 2. If X-Forwarded-For is present with a public client IP, it came through a reverse proxy (frp)
	xff := r.Header.Get("X-Forwarded-For")
	if xff != "" {
		parts := strings.Split(xff, ",")
		for _, part := range parts {
			clientIP := strings.TrimSpace(part)
			ip := net.ParseIP(clientIP)
			if ip != nil && !isPrivateIP(ip) {
				return true
			}
		}
	}

	// 3. Check X-Forwarded-Host: if present and points to a public host
	xfh := r.Header.Get("X-Forwarded-Host")
	if xfh != "" && !isPrivateHost(xfh) {
		return true
	}

	// 4. Check RemoteAddr: if RemoteAddr is not a private IP, it came directly from public internet
	remoteHost, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		remoteHost = r.RemoteAddr
	}
	remoteIP := net.ParseIP(remoteHost)
	if remoteIP != nil && !isPrivateIP(remoteIP) {
		return true
	}

	// 5. Check r.Host: when traffic is forwarded by FRP (TCP proxy) to loopback,
	// RemoteAddr is 127.0.0.1, but r.Host carries the public domain or public IP (e.g. "kr-nc-bgp-1.ofalias.net:39980").
	// If r.Host is not a private or loopback host, it is arriving via a public domain/IP.
	if r.Host != "" && !isPrivateHost(r.Host) {
		return true
	}

	return false
}

// IsLoopbackRequest checks whether the incoming request originated from the local host (127.0.0.1, ::1, localhost)
// and was not reverse-proxied from the public internet (frp).
func IsLoopbackRequest(r *http.Request) bool {
	if r == nil {
		return false
	}
	if IsTunnelRequest(r) || IsPublicRequest(r) {
		return false
	}

	// Host header MUST be a loopback host (localhost, 127.0.0.1, ::1).
	// If the request was addressed to a public domain/IP or LAN IP, it is not a loopback request.
	if r.Host != "" && !isLoopbackHost(r.Host) {
		return false
	}

	xff := r.Header.Get("X-Forwarded-For")
	if xff != "" {
		parts := strings.Split(xff, ",")
		for _, part := range parts {
			clientIP := strings.TrimSpace(part)
			ip := net.ParseIP(clientIP)
			if ip == nil || !ip.IsLoopback() {
				return false
			}
		}
	}

	xfh := r.Header.Get("X-Forwarded-Host")
	if xfh != "" && !isLoopbackHost(xfh) {
		return false
	}

	host, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		host = r.RemoteAddr
	}
	if host == "localhost" {
		return true
	}
	remoteIP := net.ParseIP(host)
	return remoteIP != nil && remoteIP.IsLoopback()
}

// HashPassword hashes a plain-text password using bcrypt with default salt cost.
func HashPassword(password string) (string, error) {
	bytes, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	return string(bytes), err
}

// CheckPassword compares a bcrypt-hashed password with its possible plaintext equivalent.
func CheckPassword(hashedPassword, password string) bool {
	if hashedPassword == "" {
		return false
	}
	err := bcrypt.CompareHashAndPassword([]byte(hashedPassword), []byte(password))
	return err == nil
}

func GenerateToken(password string) (string, error) {
	claims := jwt.MapClaims{
		"authorized": true,
		"exp":        time.Now().Add(7 * 24 * time.Hour).Unix(), // 7 days valid
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString(jwtSecretKey)
}

func ValidateToken(tokenStr string) bool {
	if tokenStr == "" {
		return false
	}
	token, err := jwt.Parse(tokenStr, func(t *jwt.Token) (interface{}, error) {
		if _, ok := t.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, jwt.ErrSignatureInvalid
		}
		return jwtSecretKey, nil
	})
	if err != nil || !token.Valid {
		return false
	}
	return true
}

func GenerateAdminToken() (string, error) {
	claims := jwt.MapClaims{
		"admin": true,
		"exp":   time.Now().Add(24 * time.Hour).Unix(),
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString(adminSecretKey)
}

func ValidateAdminToken(tokenStr string) bool {
	if tokenStr == "" {
		return false
	}
	token, err := jwt.Parse(tokenStr, func(t *jwt.Token) (interface{}, error) {
		if _, ok := t.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, jwt.ErrSignatureInvalid
		}
		return adminSecretKey, nil
	})
	if err != nil || !token.Valid {
		return false
	}
	claims, ok := token.Claims.(jwt.MapClaims)
	if !ok {
		return false
	}
	isAdmin, ok := claims["admin"].(bool)
	return ok && isAdmin
}

func CheckAdminAuth(r *http.Request) bool {
	cookie, err := r.Cookie("dn_admin_token")
	if err == nil && ValidateAdminToken(cookie.Value) {
		return true
	}

	authHeader := r.Header.Get("Authorization")
	if strings.HasPrefix(authHeader, "Bearer ") {
		token := strings.TrimPrefix(authHeader, "Bearer ")
		if ValidateAdminToken(token) || ValidateToken(token) {
			return true
		}
	}

	if headerToken := r.Header.Get("X-Admin-Token"); headerToken != "" {
		if ValidateAdminToken(headerToken) {
			return true
		}
	}

	// Also check dn_token cookie (teacher already authenticated via unified password)
	if tokenCookie, err := r.Cookie("dn_token"); err == nil && ValidateToken(tokenCookie.Value) {
		return true
	}

	return false
}

func CheckAuth(r *http.Request, cfg config.AppConfig) bool {
	// 1. Local loopback requests (desktop app, local webview) are always trusted
	if IsLoopbackRequest(r) {
		return true
	}

	isPublic := IsPublicRequest(r)
	if !isPublic && cfg.FRP.Enabled && cfg.FRP.ServerAddr != "" && r != nil && r.Host != "" {
		host, _, err := net.SplitHostPort(r.Host)
		if err != nil {
			host = r.Host
		}
		if strings.EqualFold(host, cfg.FRP.ServerAddr) {
			isPublic = true
		}
	}

	// If LAN and no password required, allow
	if !isPublic && !cfg.LanAuthRequired {
		return true
	}

	// If public, check if password is not set
	pwdHash := cfg.GetEffectivePasswordHash()
	if pwdHash == "" && cfg.PublicPassword == "" {
		return true
	}

	// Check Cookie (user token or admin token)
	cookie, err := r.Cookie("dn_token")
	if err == nil && ValidateToken(cookie.Value) {
		return true
	}
	if adminCookie, err := r.Cookie("dn_admin_token"); err == nil && ValidateAdminToken(adminCookie.Value) {
		return true
	}

	// Check Authorization header: Bearer <token>
	authHeader := r.Header.Get("Authorization")
	if strings.HasPrefix(authHeader, "Bearer ") {
		token := strings.TrimPrefix(authHeader, "Bearer ")
		if ValidateToken(token) || ValidateAdminToken(token) {
			return true
		}
	}

	if headerToken := r.Header.Get("X-Admin-Token"); headerToken != "" {
		if ValidateAdminToken(headerToken) {
			return true
		}
	}

	return false
}
