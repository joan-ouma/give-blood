package service

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

func TestPasswordHashing(t *testing.T) {
	password := "securepassword123"

	hash, err := HashPassword(password)
	if err != nil {
		t.Fatalf("failed to hash password: %v", err)
	}

	if len(hash) == 0 {
		t.Error("hash should not be empty")
	}

	if !CheckPasswordHash(password, hash) {
		t.Error("hash check should succeed for correct password")
	}

	if CheckPasswordHash("wrongpassword", hash) {
		t.Error("hash check should fail for incorrect password")
	}
}

func TestTokenService(t *testing.T) {
	secret := "mytestsupersecretkey"
	svc := NewTokenService(secret)

	userID := "user123"
	role := "donor"

	t.Run("Generate and Parse Access Token", func(t *testing.T) {
		tokenStr, err := svc.GenerateAccessToken(userID, role)
		if err != nil {
			t.Fatalf("failed to generate access token: %v", err)
		}

		claims, err := svc.ParseToken(tokenStr)
		if err != nil {
			t.Fatalf("failed to parse valid token: %v", err)
		}

		if claims["sub"] != userID {
			t.Errorf("expected sub %s, got %v", userID, claims["sub"])
		}

		if claims["role"] != role {
			t.Errorf("expected role %s, got %v", role, claims["role"])
		}
	})

	t.Run("Generate and Parse Refresh Token", func(t *testing.T) {
		tokenStr, err := svc.GenerateRefreshToken(userID, role)
		if err != nil {
			t.Fatalf("failed to generate refresh token: %v", err)
		}

		claims, err := svc.ParseToken(tokenStr)
		if err != nil {
			t.Fatalf("failed to parse valid refresh token: %v", err)
		}

		if claims["sub"] != userID {
			t.Errorf("expected sub %s, got %v", userID, claims["sub"])
		}
	})

	t.Run("Invalid Token Parsing", func(t *testing.T) {
		_, err := svc.ParseToken("invalid.token.string")
		if err == nil {
			t.Error("parsing invalid token string should fail")
		}
	})

	t.Run("Signature Mismatch", func(t *testing.T) {
		otherSvc := NewTokenService("differentsigningsecret")
		tokenStr, _ := otherSvc.GenerateAccessToken(userID, role)

		_, err := svc.ParseToken(tokenStr)
		if err == nil {
			t.Error("parsing token signed with a different key should fail")
		}
	})

	t.Run("Expired Token", func(t *testing.T) {
		expiredClaims := jwt.MapClaims{
			"sub":  userID,
			"role": role,
			"exp":  time.Now().Add(-1 * time.Minute).Unix(),
			"iat":  time.Now().Add(-10 * time.Minute).Unix(),
		}
		token := jwt.NewWithClaims(jwt.SigningMethodHS256, expiredClaims)
		tokenStr, _ := token.SignedString([]byte(secret))

		_, err := svc.ParseToken(tokenStr)
		if err == nil {
			t.Error("parsing expired token should fail")
		}
	})
}

func TestAuthMiddleware(t *testing.T) {
	secret := "mytestsupersecretkey"
	tokenService := NewTokenService(secret)
	middleware := Middleware(tokenService)

	// Handler that asserts auth values are in context
	testHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		userID, role, err := GetUserFromContext(r.Context())
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(userID + ":" + role))
	})

	handlerToTest := middleware(testHandler)

	t.Run("Missing Authorization Header", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/api/protected", nil)
		w := httptest.NewRecorder()

		handlerToTest.ServeHTTP(w, req)

		if w.Code != http.StatusUnauthorized {
			t.Errorf("expected status 401, got %d", w.Code)
		}
	})

	t.Run("Malformed Authorization Header", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/api/protected", nil)
		req.Header.Set("Authorization", "BearerInvalidFormat")
		w := httptest.NewRecorder()

		handlerToTest.ServeHTTP(w, req)

		if w.Code != http.StatusUnauthorized {
			t.Errorf("expected status 401, got %d", w.Code)
		}
	})

	t.Run("Valid Token", func(t *testing.T) {
		tokenStr, _ := tokenService.GenerateAccessToken("user999", "agency")

		req := httptest.NewRequest("GET", "/api/protected", nil)
		req.Header.Set("Authorization", "Bearer "+tokenStr)
		w := httptest.NewRecorder()

		handlerToTest.ServeHTTP(w, req)

		if w.Code != http.StatusOK {
			t.Errorf("expected status 200, got %d", w.Code)
		}

		body := w.Body.String()
		expectedBody := "user999:agency"
		if body != expectedBody {
			t.Errorf("expected body %q, got %q", expectedBody, body)
		}
	})
}

func TestGetUserFromContext(t *testing.T) {
	ctx := context.Background()

	_, _, err := GetUserFromContext(ctx)
	if err == nil {
		t.Error("expected error retrieving user from empty context")
	}

	ctxWithUserID := context.WithValue(ctx, UserIDKey, "uid123")
	_, _, err = GetUserFromContext(ctxWithUserID)
	if err == nil {
		t.Error("expected error when role is missing in context")
	}

	ctxWithBoth := context.WithValue(ctxWithUserID, RoleKey, "donor")
	uid, role, err := GetUserFromContext(ctxWithBoth)
	if err != nil {
		t.Fatalf("unexpected error getting user: %v", err)
	}

	if uid != "uid123" || role != "donor" {
		t.Errorf("unexpected context values retrieved: %s, %s", uid, role)
	}
}
