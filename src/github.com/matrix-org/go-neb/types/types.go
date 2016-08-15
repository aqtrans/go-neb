package types

import (
	"encoding/json"
	"errors"
	"github.com/matrix-org/go-neb/matrix"
	"github.com/matrix-org/go-neb/plugin"
	"net/http"
	"net/url"
	"strings"
)

// A ClientConfig is the configuration for a matrix client for a bot to use.
type ClientConfig struct {
	UserID        string // The matrix UserId to connect with.
	HomeserverURL string // A URL with the host and port of the matrix server. E.g. https://matrix.org:8448
	AccessToken   string // The matrix access token to authenticate the requests with.
}

// Check that the client has the correct fields.
func (c *ClientConfig) Check() error {
	if c.UserID == "" || c.HomeserverURL == "" || c.AccessToken == "" {
		return errors.New(`Must supply a "UserID", a "HomeserverURL", and an "AccessToken"`)
	}
	if _, err := url.Parse(c.HomeserverURL); err != nil {
		return err
	}
	return nil
}

// A Service is the configuration for a bot service.
type Service interface {
	ServiceUserID() string
	ServiceID() string
	ServiceType() string
	RoomIDs() []string
	Plugin(roomID string) plugin.Plugin
	OnReceiveWebhook(w http.ResponseWriter, req *http.Request, cli *matrix.Client)
	Register() error
	PostRegister(oldService Service)
}

var baseURL = ""

// BaseURL sets the base URL of NEB to the url given. This URL must be accessible from the
// public internet.
func BaseURL(u string) error {
	if u == "" {
		return errors.New("BASE_URL not found")
	}
	if !strings.HasPrefix(u, "http://") && !strings.HasPrefix(u, "https://") {
		return errors.New("BASE_URL must start with http[s]://")
	}
	if !strings.HasSuffix(u, "/") {
		u = u + "/"
	}
	baseURL = u
	return nil
}

var servicesByType = map[string]func(string, string) Service{}

// RegisterService registers a factory for creating Service instances.
func RegisterService(factory func(string, string) Service) {
	servicesByType[factory("", "").ServiceType()] = factory
}

// CreateService creates a Service of the given type and serviceID.
// Returns an error if the Service couldn't be created.
func CreateService(serviceID, serviceType string, serviceJSON []byte) (Service, error) {
	f := servicesByType[serviceType]
	if f == nil {
		return nil, errors.New("Unknown service type: " + serviceType)
	}
	webhookEndpointURL := baseURL + "services/hooks/" + serviceID
	service := f(serviceID, webhookEndpointURL)
	if err := json.Unmarshal(serviceJSON, service); err != nil {
		return nil, err
	}
	return service, nil
}

// AuthRealm represents a place where a user can authenticate themselves.
// This may static (like github.com) or a specific domain (like matrix.org/jira)
type AuthRealm interface {
	ID() string
	Type() string
	Init() error
	Register() error
	OnReceiveRedirect(w http.ResponseWriter, req *http.Request)
	AuthSession(id, userID, realmID string) AuthSession
	RequestAuthSession(userID string, config json.RawMessage) interface{}
}

var realmsByType = map[string]func(string, string) AuthRealm{}

// RegisterAuthRealm registers a factory for creating AuthRealm instances.
func RegisterAuthRealm(factory func(string, string) AuthRealm) {
	realmsByType[factory("", "").Type()] = factory
}

// CreateAuthRealm creates an AuthRealm of the given type and realm ID.
// Returns an error if the realm couldn't be created or the JSON cannot be unmarshalled.
func CreateAuthRealm(realmID, realmType string, realmJSON []byte) (AuthRealm, error) {
	f := realmsByType[realmType]
	if f == nil {
		return nil, errors.New("Unknown realm type: " + realmType)
	}
	redirectURL := baseURL + "realms/redirects/" + realmID
	r := f(realmID, redirectURL)
	if err := json.Unmarshal(realmJSON, r); err != nil {
		return nil, err
	}
	if err := r.Init(); err != nil {
		return nil, err
	}
	return r, nil
}

// AuthSession represents a single authentication session between a user and
// an auth realm.
type AuthSession interface {
	ID() string
	UserID() string
	RealmID() string
	Authenticated() bool
	Info() interface{}
}
