package api

import (
	"github.com/labbsr0x/goh/gohserver"
	"github.com/labbsr0x/goh/gohtypes"
	"github.com/labbsr0x/whisper/db"
	"github.com/labbsr0x/whisper/hydra"
	"github.com/labbsr0x/whisper/mail"
	"github.com/labbsr0x/whisper/web/ui"
	"github.com/sirupsen/logrus"
	"net/http"
	"net/url"

	"github.com/labbsr0x/whisper/web/api/types"
	"github.com/labbsr0x/whisper/web/config"
)

// LoginAPI defines the available user apis
type LoginAPI interface {
	LoginGETHandler(route string) http.Handler
	LoginPOSTHandler() http.Handler
}

// DefaultLoginAPI holds the default implementation of the User API interface
type DefaultLoginAPI struct {
	*config.WebBuilder
	UserCredentialsDAO db.UserCredentialsDAO
}

// InitFromWebBuilder initializes a default login api instance
func (dapi *DefaultLoginAPI) InitFromWebBuilder(w *config.WebBuilder) *DefaultLoginAPI {
	dapi.WebBuilder = w
	dapi.UserCredentialsDAO = new(db.DefaultUserCredentialsDAO).Init(w.SecretKey, w.BaseUIPath, w.PublicURL, w.Outbox, w.DB)
	return dapi
}

// LoginPOSTHandler post form handler for logging in users
func (dapi *DefaultLoginAPI) LoginPOSTHandler() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		loginRequest := new(types.RequestLoginPayload).InitFromRequest(r)
		logrus.Debugf("Login request payload '%v'", loginRequest)

		userCredential := dapi.UserCredentialsDAO.CheckCredentials(loginRequest.Username, loginRequest.Password)

		if !userCredential.EmailValidated {
			dapi.Outbox <- mail.GetEmailConfirmationMail(dapi.BaseUIPath, dapi.SecretKey, dapi.PublicURL, userCredential.Username, userCredential.Email, loginRequest.Challenge)
			gohtypes.Panic("This account email is not authenticated, an email was sent to you confirm your email", http.StatusUnauthorized)
		}

		info := dapi.HydraHelper.AcceptLoginRequest(
			loginRequest.Challenge,
			hydra.AcceptLoginRequestPayload{ACR: "0", Remember: loginRequest.Remember, RememberFor: 3600, Subject: loginRequest.Username},
		)
		logrus.Debugf("Accept login request info: %v", info)
		if info != nil {
			gohserver.WriteJSONResponse(map[string]interface{}{
				"redirect_to": info["redirect_to"],
			}, http.StatusOK, w)
			return
		}
	})
}

// LoginGETHandler prompts the browser to the login UI or redirects it to hydra
func (dapi *DefaultLoginAPI) LoginGETHandler(route string) http.Handler {
	return http.StripPrefix(route, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		challenge, err := url.QueryUnescape(r.URL.Query().Get("login_challenge"))
		if err == nil {
			info := dapi.HydraHelper.GetLoginRequestInfo(challenge)
			logrus.Debugf("Login Request Info: %v", info)
			if info["skip"].(bool) {
				subject := info["subject"].(string)
				info = dapi.HydraHelper.AcceptLoginRequest(
					challenge,
					hydra.AcceptLoginRequestPayload{Subject: subject},
				)
				if info != nil {
					logrus.Debugf("Login request skipped for subject '%v'", subject)
					http.Redirect(w, r, info["redirect_to"].(string), http.StatusFound)
				}
			} else {
				page := types.LoginPage{Challenge: challenge}
				ui.WritePage(w, dapi.BaseUIPath, ui.Login, &page)
			}
			return
		}
		panic(gohtypes.Error{Code: http.StatusBadRequest, Err: err, Message: "Unable to parse the login_challenge"})
	}))
}
