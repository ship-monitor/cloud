package auth

import (
	"errors"
	"fmt"
	"net/http"
	"strings"

	"charm.land/log/v2"
	authzed "github.com/authzed/authzed-go/v1"
	"github.com/authzed/grpcutil"
	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
	"github.com/spf13/viper"
)

var sessionKey = "session-" + uuid.New().String()
var middlewareKey = "middleware-" + uuid.New().String()

const AuthorizationHeader = "Authorization"

var (
	ErrArmenUsedBearer = errors.New("someone (Armen) included 'Bearer ' in token header")
	ErrNoAuthheader    = fmt.Errorf("header %q not specified", AuthorizationHeader)
)

// DefaultMiddleware returns a new Middleware with the default Redis configuration from viper.
//
// It uses the following viper configuration keys:
//
//   - security-key string
//   - spicedb.address string
//   - spicedb.api-key string
//
// For quickest setup use:
//
// ```
// DefaultMiddleware(viper.Sub("auth"))
// ```
func DefaultMiddleware(config *viper.Viper) *Middleware {
	return MustNewMiddleware(&MiddlewareConfig{
		&SpiceDBOptions{
			Address: config.GetString("spicedb.address"),
			APIKey:  config.GetString("spicedb.api-key"),
		},
		tokenKeyFunc(config),
	})
}
func tokenKeyFunc(config *viper.Viper) jwt.Keyfunc {

	return func(token *jwt.Token) (any, error) {
		switch token.Method {
		case jwt.SigningMethodHS256:
			fallthrough
		case jwt.SigningMethodHS384:
			fallthrough
		case jwt.SigningMethodHS512:
			return []byte(config.GetString("security-key")), nil
		default:
			return nil, fmt.Errorf("unsupported signing method: %s", token.Method.Alg())
		}
	}
}

type SpiceDBOptions struct {
	Address string
	APIKey  string
}

type MiddlewareConfig struct {
	SpiceDB         *SpiceDBOptions
	SecurityKeyFunc jwt.Keyfunc
}

func newMiddleware(config *MiddlewareConfig) (*Middleware, error) {
	systemCerts, err := grpcutil.WithSystemCerts(grpcutil.VerifyCA)
	if err != nil {
		return nil, fmt.Errorf("unable to load system CA certificates: %s", err)
	}
	spiceClient, err := authzed.NewClient(config.SpiceDB.Address, systemCerts, grpcutil.WithBearerToken(config.SpiceDB.APIKey))
	if err != nil {
		return nil, fmt.Errorf("failed connect to SpiceDB: %s", err)
	}
	return &Middleware{spiceClient, config.SecurityKeyFunc}, nil
}

func MustNewMiddleware(config *MiddlewareConfig) *Middleware {
	middleware, err := newMiddleware(config)
	if err != nil {
		panic(fmt.Errorf("failed create auth middleware: %s", err))
	}
	return middleware
}

type Middleware struct {
	spice   *authzed.Client
	keyFunc jwt.Keyfunc
}

func GetMiddleware(ctx *gin.Context) *Middleware {
	if middleware, ok := ctx.Get(middlewareKey); ok {
		return middleware.(*Middleware)
	} else {
		panic("there is no middleware")
	}
}

func (m *Middleware) addToContext(ctx *gin.Context) {
	ctx.Set(middlewareKey, m)
}

func (m *Middleware) WithMiddleware(ctx *gin.Context) {
	m.addToContext(ctx)
}

func (m *Middleware) WithAuthenticationRequired(ctx *gin.Context) {
	m.addToContext(ctx)

	header := ctx.GetHeader(AuthorizationHeader)

	if header == "" {
		err := fmt.Errorf("bad token specified: %s", ErrNoAuthheader)
		log.Error("No authorization header", "error", err)
		ctx.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"details": err.Error()})
		return
	}

	if strings.Contains(header, "Bearer") {
		err := fmt.Errorf("bad token specified: %s", ErrArmenUsedBearer)
		log.Error("Армен, заебал, пиши авторизацию сам, а не ИИшкой", "error", ErrArmenUsedBearer)
		ctx.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"details": err.Error(), "armensMessage": "stop it"})
		return
	}

	claims, err := m.ParseToken(header)
	if err != nil {
		log.Error("Failed parse JWT", "error", err)
		ctx.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"details": "bad credentials: " + err.Error()})
		return
	}

	session := &Session{
		UserID:  claims.UserID,
		Email:   claims.Email,
		spiceDB: m.spice,
		ctx:     ctx.Request.Context(),
	}

	ctx.Set(sessionKey, session)

}

func GetSession(ctx *gin.Context) *Session {
	session, ok := ctx.Get(sessionKey)
	if !ok {
		panic(fmt.Errorf("session not found (key %q), probably not authenticated", sessionKey))
	}
	return session.(*Session)
}
