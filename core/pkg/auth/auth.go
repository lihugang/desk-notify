package auth

import (
	"net"
	"net/http"
	"strings"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/lihugang/desk-notify/core/pkg/config"
)

var (
	privateIPBlocks []*net.IPNet
	jwtSecretKey    = []byte("desk-notify-default-jwt-secret-key-32")
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

// IsPublicRequest checks whether the incoming request arrived via frp / public network.
func IsPublicRequest(r *http.Request) bool {
	// 1. If X-Forwarded-For is present with a public client IP, it came through a reverse proxy (frp)
	xff := r.Header.Get("X-Forwarded-For")
	if xff != "" {
		parts := strings.Split(xff, ",")
		clientIP := strings.TrimSpace(parts[0])
		ip := net.ParseIP(clientIP)
		if ip != nil && !isPrivateIP(ip) {
			return true
		}
	}

	// 2. Check RemoteAddr
	host, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		host = r.RemoteAddr
	}
	remoteIP := net.ParseIP(host)
	if remoteIP != nil && !isPrivateIP(remoteIP) {
		return true
	}

	return false
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

func CheckAuth(r *http.Request, cfg config.AppConfig) bool {
	isPublic := IsPublicRequest(r)

	// If LAN and no password required, allow
	if !isPublic && !cfg.LanAuthRequired {
		return true
	}

	// If public, check if password is not set
	if cfg.PublicPassword == "" {
		return true
	}

	// Check Cookie
	cookie, err := r.Cookie("dn_token")
	if err == nil && ValidateToken(cookie.Value) {
		return true
	}

	// Check Authorization header: Bearer <token>
	authHeader := r.Header.Get("Authorization")
	if strings.HasPrefix(authHeader, "Bearer ") {
		token := strings.TrimPrefix(authHeader, "Bearer ")
		if ValidateToken(token) {
			return true
		}
	}

	return false
}
