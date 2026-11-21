package auth

import (
	"context"
	"crypto/sha256"
	"fmt"
	"net"
	"net/http"
	"strings"
	"time"

	"github.com/W0LIF/LoveTea/backend/internal/platform/security"
	"github.com/google/uuid"
	"github.com/redis/go-redis/v9"
)

type contextKey string

const userIDContextKey contextKey = "user_id"

func UserIDFromContext(ctx context.Context) (uuid.UUID, bool) {
	userID, ok := ctx.Value(userIDContextKey).(uuid.UUID)
	return userID, ok
}

func AuthMiddleware(tokens *security.TokenManager) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			header := r.Header.Get("Authorization")

			if !strings.HasPrefix(header, "Bearer ") {
				writeAuthJSON(w, http.StatusUnauthorized, map[string]string{
					"error": "unauthorized",
				})
				return
			}

			rawToken := strings.TrimSpace(strings.TrimPrefix(header, "Bearer "))
			userID, err := tokens.ParseAccessToken(rawToken)

			if err != nil {
				writeAuthJSON(w, http.StatusUnauthorized, map[string]string{
					"error": "unauthorized",
				})
				return
			}

			ctx := context.WithValue(r.Context(), userIDContextKey, userID)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

func RateLimit(
	client *redis.Client,
	namespace string,
	maxAttempts int64,
	window time.Duration,
) func(http.Handler) http.Handler {
	const script = `
		local current = redis.call("INCR", KEYS[1])
		if current == 1 then
			redis.call("EXPIRE", KEYS[1], ARGV[1])
		end
		return current
	`

	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			remoteAddr, _, err := net.SplitHostPort(r.RemoteAddr)
			if err != nil {
				remoteAddr = r.RemoteAddr
			}
			addressHash := sha256.Sum256([]byte(remoteAddr))
			key := fmt.Sprintf(
				"love-tea:rate-limit:%s:%x",
				namespace,
				addressHash,
			)

			count, err := client.Eval(
				r.Context(),
				script,
				[]string{key},
				int64(window.Seconds()),
			).Int64()
			if err != nil {
				writeAuthJSON(w, http.StatusServiceUnavailable, map[string]string{
					"error": "rate limiting temporarily unavailable",
				})
				return
			}
			if count > maxAttempts {
				w.Header().Set("Retry-After", fmt.Sprint(int64(window.Seconds())))
				writeAuthJSON(w, http.StatusTooManyRequests, map[string]string{
					"error": "too many requests",
				})
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}
