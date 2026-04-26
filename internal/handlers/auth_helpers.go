package handlers

import (
	"crypto"
	"crypto/rsa"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"errors"
	"math/big"
	"net/http"
	"nrs-authentication/internal/config"
	"strings"
	"sync"
	"time"

	"github.com/aws/aws-sdk-go-v2/service/cognitoidentityprovider/types"
	"github.com/gin-gonic/gin"
)

var inviteAllowedRoles = map[string]struct{}{
	"SUPER_ADMIN":    {},
	"HOSPITAL_ADMIN": {},
	"SERVICE_ADMIN":  {},
}

type tokenClaims struct {
	Email         string      `json:"email"`
	Username      string      `json:"cognito:username"`
	CognitoGroups interface{} `json:"cognito:groups"`
	Issuer        string      `json:"iss"`
	ExpiresAt     int64       `json:"exp"`
	NotBefore     int64       `json:"nbf"`
	TokenUse      string      `json:"token_use"`
	ClientID      string      `json:"client_id"`
	Audience      string      `json:"aud"`
}

type jwksDocument struct {
	Keys []jwkKey `json:"keys"`
}

type jwkKey struct {
	Kid string `json:"kid"`
	Kty string `json:"kty"`
	Alg string `json:"alg"`
	N   string `json:"n"`
	E   string `json:"e"`
	Use string `json:"use"`
}

var (
	jwksCacheMu      sync.Mutex
	jwksCachedKeys   map[string]*rsa.PublicKey
	jwksCacheExpires time.Time
)

func requireInviteAdmin(c *gin.Context, cfg *config.Config) (tokenClaims, bool) {
	claims, err := validateAndExtractClaims(c.GetHeader("Authorization"), cfg)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid authorization token"})
		return tokenClaims{}, false
	}

	for _, role := range groupsFromClaim(claims.CognitoGroups) {
		if _, ok := inviteAllowedRoles[role]; ok {
			return claims, true
		}
	}

	c.JSON(http.StatusForbidden, gin.H{"error": "insufficient role"})
	return tokenClaims{}, false
}

func validateAndExtractClaims(authorization string, cfg *config.Config) (tokenClaims, error) {
	token := strings.TrimSpace(authorization)
	if token == "" {
		return tokenClaims{}, errors.New("missing authorization header")
	}

	if strings.HasPrefix(strings.ToLower(token), "bearer ") {
		token = strings.TrimSpace(token[7:])
	}

	parts := strings.Split(token, ".")
	if len(parts) != 3 {
		return tokenClaims{}, errors.New("invalid token format")
	}

	header, err := decodeJWTPart(parts[0])
	if err != nil {
		return tokenClaims{}, err
	}

	var tokenHeader struct {
		Alg string `json:"alg"`
		Kid string `json:"kid"`
		Typ string `json:"typ"`
	}
	if err := json.Unmarshal(header, &tokenHeader); err != nil {
		return tokenClaims{}, err
	}

	if tokenHeader.Alg != "RS256" {
		return tokenClaims{}, errors.New("unsupported signing algorithm")
	}

	payload, err := decodeJWTPart(parts[1])
	if err != nil {
		return tokenClaims{}, err
	}

	var claims tokenClaims
	if err := json.Unmarshal(payload, &claims); err != nil {
		return tokenClaims{}, err
	}

	if err := verifyJWTSignature(token, tokenHeader.Kid, cfg); err != nil {
		return tokenClaims{}, err
	}

	if err := validateClaims(claims, cfg); err != nil {
		return tokenClaims{}, err
	}

	return claims, nil
}

func decodeJWTPart(part string) ([]byte, error) {
	return base64.RawURLEncoding.DecodeString(part)
}

func verifyJWTSignature(token, kid string, cfg *config.Config) error {
	if cfg == nil || strings.TrimSpace(cfg.CognitoJWKSURL) == "" {
		return errors.New("jwks url is not configured")
	}

	parts := strings.Split(token, ".")
	signingInput := parts[0] + "." + parts[1]
	signature, err := decodeJWTPart(parts[2])
	if err != nil {
		return err
	}

	key, err := getJWKSKey(kid, cfg.CognitoJWKSURL)
	if err != nil {
		return err
	}

	hashed := sha256.Sum256([]byte(signingInput))
	return rsa.VerifyPKCS1v15(key, crypto.SHA256, hashed[:], signature)
}

func validateClaims(claims tokenClaims, cfg *config.Config) error {
	now := time.Now().Unix()

	if claims.ExpiresAt != 0 && now >= claims.ExpiresAt {
		return errors.New("token has expired")
	}

	if claims.NotBefore != 0 && now < claims.NotBefore {
		return errors.New("token is not valid yet")
	}

	if cfg != nil && strings.TrimSpace(cfg.CognitoIssuer) != "" && claims.Issuer != cfg.CognitoIssuer {
		return errors.New("invalid token issuer")
	}

	if claims.TokenUse != "" && claims.TokenUse != "access" && claims.TokenUse != "id" {
		return errors.New("invalid token use")
	}

	if cfg != nil && strings.TrimSpace(cfg.CognitoAppClientID) != "" {
		if claims.ClientID != "" && claims.ClientID != cfg.CognitoAppClientID {
			return errors.New("invalid token client")
		}
		if claims.ClientID == "" && claims.Audience != "" && claims.Audience != cfg.CognitoAppClientID {
			return errors.New("invalid token audience")
		}
	}

	return nil
}

func getJWKSKey(kid, jwksURL string) (*rsa.PublicKey, error) {
	jwksCacheMu.Lock()
	defer jwksCacheMu.Unlock()

	if key, ok := jwksCachedKeys[kid]; ok && time.Now().Before(jwksCacheExpires) {
		return key, nil
	}

	resp, err := http.Get(jwksURL)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, errors.New("failed to fetch jwks")
	}

	var document jwksDocument
	if err := json.NewDecoder(resp.Body).Decode(&document); err != nil {
		return nil, err
	}

	keys := make(map[string]*rsa.PublicKey, len(document.Keys))
	for _, key := range document.Keys {
		if key.Kty != "RSA" || key.Kid == "" {
			continue
		}

		publicKey, err := buildRSAPublicKey(key.N, key.E)
		if err != nil {
			return nil, err
		}
		keys[key.Kid] = publicKey
	}

	jwksCachedKeys = keys
	jwksCacheExpires = time.Now().Add(15 * time.Minute)

	key, ok := jwksCachedKeys[kid]
	if !ok {
		return nil, errors.New("matching jwk not found")
	}

	return key, nil
}

func buildRSAPublicKey(modulus, exponent string) (*rsa.PublicKey, error) {
	modulusBytes, err := base64.RawURLEncoding.DecodeString(modulus)
	if err != nil {
		return nil, err
	}

	exponentBytes, err := base64.RawURLEncoding.DecodeString(exponent)
	if err != nil {
		return nil, err
	}

	e := 0
	for _, b := range exponentBytes {
		e = e<<8 + int(b)
	}

	if e == 0 {
		return nil, errors.New("invalid rsa exponent")
	}

	return &rsa.PublicKey{
		N: new(big.Int).SetBytes(modulusBytes),
		E: e,
	}, nil
}

func groupsFromClaim(value interface{}) []string {
	switch v := value.(type) {
	case []interface{}:
		groups := make([]string, 0, len(v))
		for _, item := range v {
			if group, ok := item.(string); ok {
				groups = append(groups, group)
			}
		}
		return groups
	case string:
		if v == "" {
			return nil
		}
		if strings.Contains(v, ",") {
			parts := strings.Split(v, ",")
			groups := make([]string, 0, len(parts))
			for _, part := range parts {
				groups = append(groups, strings.TrimSpace(part))
			}
			return groups
		}
		return []string{strings.TrimSpace(v)}
	default:
		return nil
	}
}

func getEmailFromCognitoAttributes(attributes []types.AttributeType) string {
	for _, attribute := range attributes {
		if attribute.Name == nil || attribute.Value == nil {
			continue
		}

		if strings.EqualFold(strings.TrimSpace(*attribute.Name), "email") {
			return strings.ToLower(strings.TrimSpace(*attribute.Value))
		}
	}

	return ""
}
