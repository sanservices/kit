//Package bundle initilizes all the external and internal services
package bundle

import (
	"net/http"

	"github.com/sanservices/kit/config"
	"github.com/sanservices/kit/log"
	"go.uber.org/zap"
)

//All represent all services, that contain internal and external services
type All struct {
	Services Services
}

//Services represent services schema
type Services struct {
	Log Log
}

//Log ...
type Log struct {
	Logger *zap.SugaredLogger
}

//Initialize settings all services
func (a *All) Initialize(cfg *config.General) error {
	//initilizing logging in Go.
	log.Initialize(&cfg.Info)
	a.Services.Log.Logger = log.Logger()

	return nil
}

//Middleware ...
type Middleware struct {
	UserAgent string
	Next      http.RoundTripper
}

func (m Middleware) RoundTrip(req *http.Request) (res *http.Response, e error) {
	req.Header.Set("User-Agent", m.UserAgent)
	return m.Next.RoundTrip(req)
}
