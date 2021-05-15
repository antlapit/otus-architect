package toolbox

import (
	"crypto/rsa"
	jwt "github.com/appleboy/gin-jwt/v2"
	jwtgo "github.com/dgrijalva/jwt-go"
	"github.com/gin-gonic/gin"
	"github.com/lestrrat-go/jwx/jwk"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"strconv"
	"time"
)

type AuthConfig struct {
	Realm          string
	PublicKeyFile  string
	PrivateKeyFile string
	Timeout        time.Duration
	MaxRefresh     time.Duration
	Authenticator  func(c *gin.Context) (interface{}, error)
	privKey        *rsa.PrivateKey
	pubKey         *rsa.PublicKey
	pubJWK         jwk.Key
	privJWK        jwk.Key
	tokenIssuer    string
	tokenAudience  string
}

type AuthData struct {
	Id       int64
	UserName string
	Role     string
}

const (
	IdentityKey string = "auth_id"
	UserNameKey string = "auth_username"
	RoleKey     string = "auth_role"
)

const (
	RoleAdmin string = "ADMIN"
	RoleUser  string = "USER"
)

func LoadAuthConfig() *AuthConfig {
	authTimeoutEnv, exists := os.LookupEnv("AUTH_TIMEOUT")
	var authTimeout time.Duration
	if exists {
		authTimeout, _ = time.ParseDuration(authTimeoutEnv)
	} else {
		authTimeout = time.Hour
	}

	maxRefreshEnv, exists := os.LookupEnv("AUTH_MAX_REFRESH")
	var maxRefresh time.Duration
	if exists {
		maxRefresh, _ = time.ParseDuration(maxRefreshEnv)
	} else {
		maxRefresh = time.Hour
	}
	config := &AuthConfig{
		Realm:          os.Getenv("AUTH_REALM"),
		PublicKeyFile:  os.Getenv("AUTH_PUBLIC_KEY_FILE"),
		PrivateKeyFile: os.Getenv("AUTH_PRIVATE_KEY_FILE"),
		Timeout:        authTimeout,
		MaxRefresh:     maxRefresh,
		Authenticator:  nil,
		tokenIssuer:    os.Getenv("AUTH_ISSUER"),
		tokenAudience:  os.Getenv("AUTH_AUDIENCE"),
	}
	if len(config.PublicKeyFile) > 0 || len(config.PrivateKeyFile) > 0 {
		config.readKeys()
	}
	return config
}

func (config *AuthConfig) readKeys() error {
	err := config.privateKey()
	if err != nil {
		return err
	}
	err = config.publicKey()
	if err != nil {
		return err
	}

	err = config.privateJwk()
	if err != nil {
		return err
	}
	err = config.publicJwk()
	if err != nil {
		return err
	}
	return nil
}

func (config *AuthConfig) privateKey() error {
	keyData, err := ioutil.ReadFile(config.PrivateKeyFile)
	if err != nil {
		return jwt.ErrNoPrivKeyFile
	}
	key, err := jwtgo.ParseRSAPrivateKeyFromPEM(keyData)
	if err != nil {
		return jwt.ErrInvalidPrivKey
	}
	config.privKey = key
	return nil
}

func (config *AuthConfig) publicKey() error {
	keyData, err := ioutil.ReadFile(config.PublicKeyFile)
	if err != nil {
		return jwt.ErrNoPubKeyFile
	}
	key, err := jwtgo.ParseRSAPublicKeyFromPEM(keyData)
	if err != nil {
		return jwt.ErrInvalidPubKey
	}
	config.pubKey = key
	return nil
}

func InitAuthMiddleware(config *AuthConfig) *jwt.GinJWTMiddleware {
	authMiddleware, err := jwt.New(&jwt.GinJWTMiddleware{
		Realm:            config.Realm,
		SigningAlgorithm: "RS256",
		Timeout:          config.Timeout,
		MaxRefresh:       config.MaxRefresh,
		IdentityKey:      IdentityKey,
		IdentityHandler: func(c *gin.Context) interface{} {
			claims := jwt.ExtractClaims(c)
			return &AuthData{
				Id:       int64(claims[IdentityKey].(float64)),
				UserName: claims[UserNameKey].(string),
				Role:     claims[RoleKey].(string),
			}
		},
		PayloadFunc: func(data interface{}) jwt.MapClaims {
			if v, ok := data.(*AuthData); ok {
				return jwt.MapClaims{
					IdentityKey: v.Id,
					UserNameKey: v.UserName,
					RoleKey:     v.Role,
					"sub":       strconv.FormatInt(v.Id, 10),
					"aud":       config.tokenAudience,
					"iss":       config.tokenIssuer,
				}
			}
			return jwt.MapClaims{}
		},
		Authenticator: config.Authenticator,
		TokenLookup:   "header: Authorization",
		TokenHeadName: "Bearer",
		TimeFunc:      time.Now,
		PrivKeyFile:   config.PrivateKeyFile,
		PubKeyFile:    config.PublicKeyFile,
	})

	if err != nil {
		log.Fatal("JWT Error:" + err.Error())
	}

	errInit := authMiddleware.MiddlewareInit()

	if errInit != nil {
		log.Fatal("authMiddleware.MiddlewareInit() Error:" + errInit.Error())
	}
	return authMiddleware
}

func LoginHandler(config *AuthConfig, mw *jwt.GinJWTMiddleware) func(context *gin.Context) {
	return func(c *gin.Context) {
		if mw.Authenticator == nil {
			unauthorized(mw, c, http.StatusInternalServerError, mw.HTTPStatusMessageFunc(jwt.ErrMissingAuthenticatorFunc, c))
			return
		}

		data, err := mw.Authenticator(c)

		if err != nil {
			unauthorized(mw, c, http.StatusUnauthorized, mw.HTTPStatusMessageFunc(err, c))
			return
		}

		// Create the token
		token := newCustomToken(config, jwtgo.GetSigningMethod(mw.SigningAlgorithm))
		claims := token.Claims.(jwtgo.MapClaims)

		if mw.PayloadFunc != nil {
			for key, value := range mw.PayloadFunc(data) {
				claims[key] = value
			}
		}

		expire := mw.TimeFunc().Add(mw.Timeout)
		claims["exp"] = expire.Unix()
		claims["iat"] = mw.TimeFunc().Unix()
		claims["orig_iat"] = mw.TimeFunc().Unix()
		tokenString, err := token.SignedString(config.privKey)

		if err != nil {
			unauthorized(mw, c, http.StatusUnauthorized, mw.HTTPStatusMessageFunc(jwt.ErrFailedTokenCreation, c))
			return
		}

		// set cookie
		if mw.SendCookie {
			expireCookie := mw.TimeFunc().Add(mw.CookieMaxAge)
			maxage := int(expireCookie.Unix() - mw.TimeFunc().Unix())

			if mw.CookieSameSite != 0 {
				c.SetSameSite(mw.CookieSameSite)
			}

			c.SetCookie(
				mw.CookieName,
				tokenString,
				maxage,
				"/",
				mw.CookieDomain,
				mw.SecureCookie,
				mw.CookieHTTPOnly,
			)
		}

		mw.LoginResponse(c, http.StatusOK, tokenString, expire)
	}
}

func newCustomToken(config *AuthConfig, method jwtgo.SigningMethod) *jwtgo.Token {
	kid, _ := config.pubJWK.Get("kid")
	return &jwtgo.Token{
		Header: map[string]interface{}{
			"typ": "JWT",
			"alg": method.Alg(),
			"kid": kid,
		},
		Claims: jwtgo.MapClaims{},
		Method: method,
	}
}

func unauthorized(mw *jwt.GinJWTMiddleware, c *gin.Context, code int, message string) {
	c.Header("WWW-Authenticate", "JWT realm="+mw.Realm)
	if !mw.DisabledAbort {
		c.Abort()
	}

	mw.Unauthorized(c, code, message)
}

func (config *AuthConfig) publicJwk() error {
	var err error
	config.pubJWK, err = jwk.New(config.pubKey)

	err = jwk.AssignKeyID(config.pubJWK)
	if err != nil {
		log.Printf("failed to assign kid: %s", err)
		return err
	}

	config.pubJWK.Set("alg", "RS256")
	return err
}

func (config *AuthConfig) privateJwk() error {
	var err error
	config.privJWK, err = jwk.New(config.privKey)

	err = jwk.AssignKeyID(config.privJWK)
	if err != nil {
		log.Printf("failed to assign kid: %s", err)
		return err
	}
	return err
}

func createPrivateJWK(privateKey *rsa.PrivateKey) (jwk.Key, error) {
	set, err := jwk.New(privateKey)
	if err != nil {
		log.Printf("failed to convert to JWK: %s", err)
		return nil, err
	}

	err = jwk.AssignKeyID(set)
	if err != nil {
		log.Printf("failed to assign kid: %s", err)
		return nil, err
	}
	return set, nil
}
