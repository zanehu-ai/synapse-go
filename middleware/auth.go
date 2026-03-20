package middleware

import (
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/go-redis/redis/v8"
	"github.com/golang-jwt/jwt/v5"
)

// ExtractBearerToken extracts the token from an "Authorization: Bearer <token>" header.
func ExtractBearerToken(c *gin.Context) (string, bool) {
	parts := strings.SplitN(c.GetHeader("Authorization"), " ", 2)
	if len(parts) != 2 || parts[0] != "Bearer" {
		return "", false
	}
	return parts[1], true
}

// ParseJWT validates and parses a JWT token with the given secret, returning the claims.
func ParseJWT(token, secret string) (jwt.MapClaims, error) {
	claims := jwt.MapClaims{}
	parsed, err := jwt.ParseWithClaims(token, claims, func(t *jwt.Token) (interface{}, error) {
		return []byte(secret), nil
	})
	if err != nil || !parsed.Valid {
		return nil, fmt.Errorf("invalid token")
	}
	return claims, nil
}

// LoginRateLimit blocks login attempts after too many failures.
func LoginRateLimit(rdb *redis.Client, maxAttempts int64, window time.Duration) gin.HandlerFunc {
	return func(c *gin.Context) {
		if rdb == nil {
			c.Next()
			return
		}

		ip := c.ClientIP()
		key := fmt.Sprintf("login_fail:%s", ip)

		ctx := c.Request.Context()
		count, _ := rdb.Get(ctx, key).Int64()
		if count >= maxAttempts {
			c.AbortWithStatusJSON(http.StatusTooManyRequests, gin.H{
				"code": 4029, "message": "too many failed login attempts, please try again later", "data": nil,
			})
			return
		}

		c.Next()

		if c.Writer.Status() == http.StatusUnauthorized {
			pipe := rdb.Pipeline()
			pipe.Incr(ctx, key)
			pipe.Expire(ctx, key, window)
			pipe.Exec(ctx)
		} else if c.Writer.Status() == http.StatusOK {
			rdb.Del(ctx, key)
		}
	}
}

// IPRateLimit limits requests per IP address using a sliding window (requests per minute).
func IPRateLimit(rdb *redis.Client, rpm int) gin.HandlerFunc {
	return func(c *gin.Context) {
		if rdb == nil {
			c.Next()
			return
		}

		ip := c.ClientIP()
		key := fmt.Sprintf("ip_rl:%s", ip)
		ctx := c.Request.Context()

		count, _ := rdb.Incr(ctx, key).Result()
		rdb.Expire(ctx, key, 60*time.Second)
		if count > int64(rpm) {
			c.AbortWithStatusJSON(http.StatusTooManyRequests, gin.H{
				"code": 4029, "message": "rate limit exceeded", "data": nil,
			})
			return
		}

		c.Next()
	}
}
