package auth

import (
	"crypto/rand"
	"crypto/subtle"
	"encoding/base64"
	"fmt"
	"net/http"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
)

var (
	jwtSecret     []byte
	jwtSecretOnce sync.Once
)

// Config holds authentication configuration
type Config struct {
	Enabled      bool
	Username     string
	Password     string
	JWTSecret    string
	TokenExpiry  time.Duration
	CookieName   string
	CookieDomain string
	CookieSecure bool
}

// GetDefaultConfig returns default auth configuration
func GetDefaultConfig() *Config {
	return &Config{
		Enabled:      true,
		Username:     os.Getenv("NECTAR_AUTH_USERNAME"),
		Password:     os.Getenv("NECTAR_AUTH_PASSWORD"),
		JWTSecret:    os.Getenv("NECTAR_JWT_SECRET"),
		TokenExpiry:  24 * time.Hour,
		CookieName:   "nectar_token",
		CookieDomain: "",
		CookieSecure: false,
	}
}

// Claims represents JWT claims
type Claims struct {
	Username string `json:"username"`
	jwt.RegisteredClaims
}

// InitJWTSecret initializes the JWT secret
func InitJWTSecret(config *Config) error {
	jwtSecretOnce.Do(func() {
		if config.JWTSecret != "" {
			jwtSecret = []byte(config.JWTSecret)
		} else {
			// Generate a random secret if not provided
			secret := make([]byte, 32)
			if _, err := rand.Read(secret); err != nil {
				panic(fmt.Sprintf("Failed to generate JWT secret: %v", err))
			}
			jwtSecret = secret
			fmt.Printf("Generated JWT secret: %s\n", base64.StdEncoding.EncodeToString(jwtSecret))
			fmt.Println("Set NECTAR_JWT_SECRET environment variable to persist this secret")
		}
	})
	return nil
}

// GenerateToken generates a JWT token for the given username
func GenerateToken(username string, config *Config) (string, error) {
	claims := &Claims{
		Username: username,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(config.TokenExpiry)),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
			NotBefore: jwt.NewNumericDate(time.Now()),
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString(jwtSecret)
}

// ValidateToken validates a JWT token
func ValidateToken(tokenString string) (*Claims, error) {
	claims := &Claims{}
	token, err := jwt.ParseWithClaims(tokenString, claims, func(token *jwt.Token) (interface{}, error) {
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
		}
		return jwtSecret, nil
	})

	if err != nil {
		return nil, err
	}

	if !token.Valid {
		return nil, fmt.Errorf("invalid token")
	}

	return claims, nil
}

// BasicAuth returns a Gin middleware for basic authentication
func BasicAuth(config *Config) gin.HandlerFunc {
	return func(c *gin.Context) {
		if !config.Enabled {
			c.Next()
			return
		}

		// Skip auth for login endpoint
		if c.Request.URL.Path == "/login" || c.Request.URL.Path == "/api/login" {
			c.Next()
			return
		}

		// Check for JWT token in cookie
		cookie, err := c.Cookie(config.CookieName)
		if err == nil && cookie != "" {
			claims, err := ValidateToken(cookie)
			if err == nil {
				c.Set("username", claims.Username)
				c.Next()
				return
			}
		}

		// Check for JWT token in Authorization header
		authHeader := c.GetHeader("Authorization")
		if authHeader != "" {
			parts := strings.Split(authHeader, " ")
			if len(parts) == 2 && parts[0] == "Bearer" {
				claims, err := ValidateToken(parts[1])
				if err == nil {
					c.Set("username", claims.Username)
					c.Next()
					return
				}
			}
		}

		// Fall back to basic auth
		username, password, hasAuth := c.Request.BasicAuth()
		if hasAuth && subtle.ConstantTimeCompare([]byte(username), []byte(config.Username)) == 1 &&
			subtle.ConstantTimeCompare([]byte(password), []byte(config.Password)) == 1 {
			c.Set("username", username)
			c.Next()
			return
		}

		// Require authentication
		c.Header("WWW-Authenticate", `Basic realm="Nectar Dashboard"`)
		c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{
			"error": "Authentication required",
		})
	}
}

// LoginHandler handles login requests
func LoginHandler(config *Config) gin.HandlerFunc {
	return func(c *gin.Context) {
		var loginReq struct {
			Username string `json:"username" binding:"required"`
			Password string `json:"password" binding:"required"`
		}

		if err := c.ShouldBindJSON(&loginReq); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request"})
			return
		}

		// Validate credentials
		if subtle.ConstantTimeCompare([]byte(loginReq.Username), []byte(config.Username)) != 1 ||
			subtle.ConstantTimeCompare([]byte(loginReq.Password), []byte(config.Password)) != 1 {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid credentials"})
			return
		}

		// Generate token
		token, err := GenerateToken(loginReq.Username, config)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to generate token"})
			return
		}

		// Set cookie
		c.SetCookie(
			config.CookieName,
			token,
			int(config.TokenExpiry.Seconds()),
			"/",
			config.CookieDomain,
			config.CookieSecure,
			true, // httpOnly
		)

		c.JSON(http.StatusOK, gin.H{
			"token":    token,
			"username": loginReq.Username,
			"expires":  time.Now().Add(config.TokenExpiry).Unix(),
		})
	}
}

// LogoutHandler handles logout requests
func LogoutHandler(config *Config) gin.HandlerFunc {
	return func(c *gin.Context) {
		// Clear cookie
		c.SetCookie(
			config.CookieName,
			"",
			-1,
			"/",
			config.CookieDomain,
			config.CookieSecure,
			true,
		)

		c.JSON(http.StatusOK, gin.H{"message": "Logged out successfully"})
	}
}