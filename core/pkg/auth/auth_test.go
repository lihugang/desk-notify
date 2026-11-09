package auth

import (
	"net/http/httptest"
	"testing"

	"github.com/lihugang/desk-notify/core/pkg/config"
)

func TestBcryptPassword(t *testing.T) {
	password := "123456"
	hash, err := HashPassword(password)
	if err != nil {
		t.Fatalf("HashPassword failed: %v", err)
	}
	if hash == "" || hash == password {
		t.Fatalf("HashPassword should return salted bcrypt hash, got %s", hash)
	}

	if !CheckPassword(hash, password) {
		t.Fatalf("CheckPassword should return true for correct password")
	}

	if CheckPassword(hash, "wrongpassword") {
		t.Fatalf("CheckPassword should return false for incorrect password")
	}
}

func TestAdminToken(t *testing.T) {
	token, err := GenerateAdminToken()
	if err != nil {
		t.Fatalf("GenerateAdminToken failed: %v", err)
	}

	if !ValidateAdminToken(token) {
		t.Fatalf("ValidateAdminToken failed for valid token")
	}

	if ValidateAdminToken("invalid.token.here") {
		t.Fatalf("ValidateAdminToken should reject invalid token")
	}

	// Normal user token should NOT validate as admin
	userToken, err := GenerateToken("pwd")
	if err != nil {
		t.Fatalf("GenerateToken failed: %v", err)
	}
	if ValidateAdminToken(userToken) {
		t.Fatalf("User token should not be valid as admin token")
	}
}

func TestIsLoopbackRequest(t *testing.T) {
	// 1. Plain 127.0.0.1
	req1 := httptest.NewRequest("GET", "http://localhost:8920/api/history", nil)
	req1.RemoteAddr = "127.0.0.1:54321"
	if !IsLoopbackRequest(req1) {
		t.Fatalf("expected req1 (127.0.0.1) to be recognized as loopback")
	}

	// 2. IPv6 loopback
	req2 := httptest.NewRequest("GET", "http://localhost:8920/api/history", nil)
	req2.RemoteAddr = "[::1]:54321"
	if !IsLoopbackRequest(req2) {
		t.Fatalf("expected req2 ([::1]) to be recognized as loopback")
	}

	// 3. FRP forwarded request with public XFF
	req3 := httptest.NewRequest("GET", "http://localhost:8920/api/history", nil)
	req3.RemoteAddr = "127.0.0.1:54321"
	req3.Header.Set("X-Forwarded-For", "203.0.113.195")
	if IsLoopbackRequest(req3) {
		t.Fatalf("expected req3 with public X-Forwarded-For not to be loopback")
	}

	// 4. LAN request
	req4 := httptest.NewRequest("GET", "http://192.168.1.100:8920/api/history", nil)
	req4.RemoteAddr = "192.168.1.50:54321"
	if IsLoopbackRequest(req4) {
		t.Fatalf("expected req4 (192.168.1.50) not to be loopback")
	}

	// 5. FRP request via public domain (Host header without XFF, RemoteAddr=127.0.0.1)
	reqFRP := httptest.NewRequest("GET", "http://kr-nc-bgp-1.ofalias.net:39980/api/status", nil)
	reqFRP.RemoteAddr = "127.0.0.1:54321"
	if IsLoopbackRequest(reqFRP) {
		t.Fatalf("expected reqFRP (kr-nc-bgp-1.ofalias.net) not to be recognized as loopback")
	}
	if !IsPublicRequest(reqFRP) {
		t.Fatalf("expected reqFRP to be recognized as public request")
	}

	// 6. Request marked with tunnel context (even if Host header is spoofed to localhost)
	reqSpoofed := httptest.NewRequest("GET", "http://localhost:8920/api/status", nil)
	reqSpoofed.RemoteAddr = "127.0.0.1:54321"
	reqSpoofed = reqSpoofed.WithContext(WithTunnelContext(reqSpoofed.Context()))
	if IsLoopbackRequest(reqSpoofed) {
		t.Fatalf("expected reqSpoofed with tunnel context not to be recognized as loopback")
	}
	if !IsPublicRequest(reqSpoofed) {
		t.Fatalf("expected reqSpoofed with tunnel context to be recognized as public request")
	}
}

func TestCheckAuthFRPAndPublic(t *testing.T) {
	hash, _ := HashPassword("123456")
	cfg := config.AppConfig{
		PasswordHash:    hash,
		LanAuthRequired: false, // LAN free, but public must require password!
		FRP: config.FRPConfig{
			Enabled:    true,
			ServerAddr: "kr-nc-bgp-1.ofalias.net",
			RemotePort: 39980,
		},
	}

	// 1. Unauthenticated FRP request via public domain should FAIL CheckAuth
	reqFRP := httptest.NewRequest("GET", "http://kr-nc-bgp-1.ofalias.net:39980/api/status", nil)
	reqFRP.RemoteAddr = "127.0.0.1:54321"
	if CheckAuth(reqFRP, cfg) {
		t.Fatalf("expected unauthenticated FRP request to fail CheckAuth")
	}

	// 2. Authenticated FRP request with valid token should PASS CheckAuth
	token, _ := GenerateToken("123456")
	reqFRPAuth := httptest.NewRequest("GET", "http://kr-nc-bgp-1.ofalias.net:39980/api/status", nil)
	reqFRPAuth.RemoteAddr = "127.0.0.1:54321"
	reqFRPAuth.Header.Set("Authorization", "Bearer "+token)
	if !CheckAuth(reqFRPAuth, cfg) {
		t.Fatalf("expected authenticated FRP request with token to pass CheckAuth")
	}

	// 3. Spoofed Host via tunnel context should FAIL CheckAuth when unauthenticated
	reqSpoofed := httptest.NewRequest("GET", "http://localhost:8920/api/config", nil)
	reqSpoofed.RemoteAddr = "127.0.0.1:54321"
	reqSpoofed = reqSpoofed.WithContext(WithTunnelContext(reqSpoofed.Context()))
	if CheckAuth(reqSpoofed, cfg) {
		t.Fatalf("expected spoofed Host via tunnel context to fail CheckAuth")
	}

	// 4. Real local loopback request (localhost) should PASS CheckAuth
	reqLocal := httptest.NewRequest("GET", "http://localhost:8920/api/config", nil)
	reqLocal.RemoteAddr = "127.0.0.1:54321"
	if !CheckAuth(reqLocal, cfg) {
		t.Fatalf("expected real loopback request to pass CheckAuth")
	}

	// 5. LAN request when LanAuthRequired=false should PASS CheckAuth
	reqLAN := httptest.NewRequest("GET", "http://192.168.1.100:8920/api/status", nil)
	reqLAN.RemoteAddr = "192.168.1.50:54321"
	if !CheckAuth(reqLAN, cfg) {
		t.Fatalf("expected LAN request to pass CheckAuth when LanAuthRequired=false")
	}
}

func TestCheckAuthLoopbackAndLan(t *testing.T) {
	hash, _ := HashPassword("123456")
	cfg := config.AppConfig{
		PasswordHash:    hash,
		LanAuthRequired: true,
	}

	// 1. Loopback request when LanAuthRequired=true should be allowed (Desktop app)
	reqLocal := httptest.NewRequest("GET", "http://localhost:8920/api/config", nil)
	reqLocal.RemoteAddr = "127.0.0.1:54321"
	if !CheckAuth(reqLocal, cfg) {
		t.Fatalf("expected loopback request to pass CheckAuth even when LanAuthRequired=true")
	}

	// 2. LAN request without token when LanAuthRequired=true should be rejected
	reqLAN := httptest.NewRequest("GET", "http://192.168.1.100:8920/api/config", nil)
	reqLAN.RemoteAddr = "192.168.1.50:54321"
	if CheckAuth(reqLAN, cfg) {
		t.Fatalf("expected unauthenticated LAN request to fail CheckAuth when LanAuthRequired=true")
	}

	// 3. LAN request with valid token when LanAuthRequired=true should pass
	token, _ := GenerateToken("123456")
	reqLANAuth := httptest.NewRequest("GET", "http://192.168.1.100:8920/api/config", nil)
	reqLANAuth.RemoteAddr = "192.168.1.50:54321"
	reqLANAuth.Header.Set("Authorization", "Bearer "+token)
	if !CheckAuth(reqLANAuth, cfg) {
		t.Fatalf("expected authenticated LAN request to pass CheckAuth")
	}

	// 4. LAN request when LanAuthRequired=false should pass without token
	cfgNoLanAuth := config.AppConfig{
		PasswordHash:    hash,
		LanAuthRequired: false,
	}
	if !CheckAuth(reqLAN, cfgNoLanAuth) {
		t.Fatalf("expected LAN request to pass CheckAuth when LanAuthRequired=false")
	}
}
