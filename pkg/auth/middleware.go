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
	"github.com/spf13/viper"
	"sourcecraft.dev/organization-shipmonitor/ship-cloud-auth/pkg/requests"
)

const (
	AuthorizationHeader = "Authorization"

	sessionKey    = "ship-auth-session"
	middlewareKey = "ship-auth-middleware"
)

var (
	ErrArmenUsedBearer = errors.New(
		"someone (Armen) included 'Bearer ' in token header",
	)
	ErrNoAuthHeader = fmt.Errorf(
		"header %q not specified",
		AuthorizationHeader,
	)
	ErrUnsupportedSigninMethod = errors.New("unsupported signing method")

	ErrNoSessionInCtx = errors.New(
		"no session in context, probably not authenticated",
	)
	ErrUnexpectedSessionType = errors.New(
		"session in context is of unexpected type",
	)
	ErrNoMiddlewareInCtx        = errors.New("no middleware in context")
	ErrUnexpectedMiddlewareType = errors.New(
		"middleware in context is of unexpected type",
	)
)

// DefaultMiddleware returns a new Middleware with the default Redis
// configuration from viper.
//
// It uses the following viper configuration keys:
//
//   - security-key string
//   - spicedb.address string
//   - spicedb.api-key string
//
// For quickest setup use:
//
// DefaultMiddleware(viper.Sub("auth"))
// .
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
			return nil, fmt.Errorf(
				"method %s: %w",
				token.Method.Alg(),
				ErrUnsupportedSigninMethod,
			)
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

type Middleware struct {
	log     *log.Logger
	spice   *authzed.Client
	keyFunc jwt.Keyfunc
}

func newMiddleware(config *MiddlewareConfig) (*Middleware, error) {
	systemCerts, err := grpcutil.WithSystemCerts(grpcutil.VerifyCA)
	if err != nil {
		return nil, fmt.Errorf("unable to load system CA certificates: %w", err)
	}

	spiceClient, err := authzed.NewClient(
		config.SpiceDB.Address,
		systemCerts,
		grpcutil.WithBearerToken(config.SpiceDB.APIKey),
	)
	if err != nil {
		return nil, fmt.Errorf("failed connect to SpiceDB: %w", err)
	}

	return &Middleware{
		log.WithPrefix("auth_middleware"),
		spiceClient,
		config.SecurityKeyFunc,
	}, nil
}

func MustNewMiddleware(config *MiddlewareConfig) *Middleware {
	middleware, err := newMiddleware(config)
	if err != nil {
		panic(fmt.Errorf("failed create auth middleware: %w", err))
	}

	return middleware
}

func GetMiddleware(ctx *gin.Context) *Middleware {
	middleware, ok := ctx.Get(middlewareKey)
	if !ok {
		abortRequest(ctx, ErrNoMiddlewareInCtx)
	}

	m, ok := middleware.(*Middleware)
	if !ok {
		abortRequest(ctx, ErrUnexpectedMiddlewareType)
	}

	return m
}

func (m *Middleware) WithMiddleware(ctx *gin.Context) {
	m.addToContext(ctx)
}

func (m *Middleware) WithAuthenticationRequired(ctx *gin.Context) {
	m.addToContext(ctx)

	header := ctx.GetHeader(AuthorizationHeader)

	if header == "" {
		err := fmt.Errorf("bad token specified: %w", ErrNoAuthHeader)
		m.log.Error("No authorization header", "error", err)
		ctx.AbortWithStatusJSON(
			http.StatusUnauthorized,
			requests.ResponseErr(err),
		)

		return
	}

	if strings.Contains(header, "Bearer") {
		err := fmt.Errorf("bad token specified: %w", ErrArmenUsedBearer)
		m.log.Error("Bad token specified", "error", err)
		ctx.AbortWithStatusJSON(
			http.StatusUnauthorized,
			requests.ResponseArmenErr(
				err,
				"Армен, пиши авторизацию не через ИИ",
			),
		)

		return
	}

	claims, err := m.ParseToken(header)
	if err != nil {
		m.log.Error("Failed parse JWT", "error", err)
		ctx.AbortWithStatusJSON(
			http.StatusUnauthorized,
			requests.ResponseErr(fmt.Errorf("bad credentials: %w", err)),
		)

		return
	}

	session := &Session{
		UserID:  claims.UserID,
		Email:   claims.Email,
		spiceDB: m.spice,
		c:       ctx,
	}

	ctx.Set(sessionKey, session)
}

func (m *Middleware) addToContext(ctx *gin.Context) {
	ctx.Set(middlewareKey, m)
}

func GetSession(ctx *gin.Context) *Session {
	session, ok := ctx.Get(sessionKey)
	if !ok {
		abortRequest(ctx, ErrNoSessionInCtx)
	}

	s, ok := session.(*Session)
	if !ok {
		abortRequest(ctx, ErrUnexpectedSessionType)
	}

	return s
}

func abortRequest(ctx *gin.Context, err error) {
	ctx.AbortWithStatusJSON(http.StatusUnauthorized, requests.ResponseErr(err))
	_ = ctx.Error(err)

	panic(fmt.Sprintf("aborting request: %s", err))
}
