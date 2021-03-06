package otpRestful

import (
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/emicklei/go-restful"
	defs "github.com/ibm-security-innovation/libsecurity-go/defs"
	"github.com/ibm-security-innovation/libsecurity-go/otp"
	cr "github.com/ibm-security-innovation/libsecurity-go/restful/common-restful"
	"github.com/ibm-security-innovation/libsecurity-go/restful/libsecurity-restful"
)

const (
	otpPrefix = "/otp"
	usersPath = "/users"

	userIDParam         = "user-name"
	userNameComment     = "user name"
	blockedStateToken   = "blocked-state"
	blockedStateParam   = "blocked-state"
	verifyHotpTypeParam = "verify-hotp"
	verifyTotpTypeParam = "verify-totp"

	originToken = "Origin"

	trueStr  = "true"
	falseStr = "false"

	blockedStr    = "blocked"
	notblockedStr = "not blocked"
)

var (
	servicePath         string // = cr.ServicePathPrefix + otpPrefix
	checkSecretStrength = true // Allow only strength passwords
)

// OtpRestful : OtpRestful structure
type OtpRestful struct {
	st *libsecurityRestful.LibsecurityRestful
}

type userState struct {
	Blocked bool
}

func init() {
	initCommandToPath()
}

// NewOtpRestful : return a pointer to the OtpRestful structure
func NewOtpRestful() *OtpRestful {
	return &OtpRestful{}
}

// SetData : initialize the OtpRestful structure
func (u *OtpRestful) SetData(stR *libsecurityRestful.LibsecurityRestful) {
	u.st = stR
}

func (u OtpRestful) getURLPath(request *restful.Request, name string) cr.URL {
	//	return cr.URL{URL: fmt.Sprintf("%v%v/%v", request.Request.Header.Get(originToken), servicePath, name)}
	return cr.URL{URL: fmt.Sprintf("%v/%v", servicePath, name)}
}

func (u OtpRestful) setError(response *restful.Response, httpStatusCode int, err error) {
	data, _ := json.Marshal(cr.Error{Code: httpStatusCode, Message: fmt.Sprintf("%v", err)})
	response.WriteErrorString(httpStatusCode, string(data))
}

func (u OtpRestful) getOtp(request *restful.Request, response *restful.Response) *otp.UserInfoOtp {
	userName := request.PathParameter(userIDParam)
	data, err := cr.GetPropertyData(userName, defs.OtpPropertyName, u.st.UsersList)
	if err != nil {
		u.setError(response, http.StatusNotFound, err)
		return nil
	}
	return data.(*otp.UserInfoOtp)
}

func (u OtpRestful) restAddOtp(request *restful.Request, response *restful.Response) {
	var secret cr.Secret
	name := request.PathParameter(userIDParam)
	err := request.ReadEntity(&secret)
	if err != nil {
		u.setError(response, http.StatusBadRequest, err)
		return
	}
	data, err := otp.NewSimpleOtpUser([]byte(secret.Secret), checkSecretStrength)
	if err != nil {
		u.setError(response, http.StatusBadRequest, err)
		return
	}
	err = u.st.UsersList.AddPropertyToEntity(name, defs.OtpPropertyName, data)
	if err != nil {
		u.setError(response, http.StatusNotFound, err)
		return
	}
	response.WriteHeaderAndEntity(http.StatusCreated, u.getURLPath(request, name))
}

func (u OtpRestful) restGetOtp(request *restful.Request, response *restful.Response) {
	data := u.getOtp(request, response)
	if data == nil {
		return
	}
	response.WriteHeaderAndEntity(http.StatusOK, data)
}

func (u OtpRestful) restDeleteOtp(request *restful.Request, response *restful.Response) {
	name := request.PathParameter(userIDParam)
	err := u.st.UsersList.RemovePropertyFromEntity(name, defs.OtpPropertyName)
	if err != nil {
		u.setError(response, http.StatusNotFound, err)
	} else {
		response.WriteHeader(http.StatusNoContent)
	}
}

func (u OtpRestful) restIsOtpBlocked(request *restful.Request, response *restful.Response) {
	var state userState

	data := u.getOtp(request, response)
	if data == nil {
		return
	}
	ok, err := data.IsOtpUserBlocked()
	state.Blocked = ok
	if err != nil {
		u.setError(response, http.StatusNotFound, err)
		return
	}
	response.WriteHeaderAndEntity(http.StatusOK, state)
}

func (u OtpRestful) restSetOtpBlockedState(request *restful.Request, response *restful.Response) {
	var blockedState userState

	name := request.PathParameter(userIDParam)
	err := request.ReadEntity(&blockedState)
	if err != nil {
		u.setError(response, http.StatusBadRequest, err)
		return
	}
	data := u.getOtp(request, response)
	if data == nil {
		return
	}
	err = data.SetOtpUserBlockedState(blockedState.Blocked)
	if err != nil {
		u.setError(response, http.StatusNotFound, err)
		return
	}
	response.WriteHeaderAndEntity(http.StatusOK, u.getURLPath(request, name))
}

func (u OtpRestful) restVerifyOtpHotpUserCode(request *restful.Request, response *restful.Response) {
	u.verifyUserOtp(request, response, otp.HotpType)
}

func (u OtpRestful) restVerifyOtpTotpUserCode(request *restful.Request, response *restful.Response) {
	u.verifyUserOtp(request, response, otp.TotpType)
}

func (u OtpRestful) verifyUserOtp(request *restful.Request, response *restful.Response, otpType otp.TypeOfOtp) {
	var secret cr.Secret

	err := request.ReadEntity(&secret)
	if err != nil {
		u.setError(response, http.StatusBadRequest, err)
		return
	}
	data := u.getOtp(request, response)
	if data == nil {
		return
	}
	ok, err := data.VerifyOtpUserCode(secret.Secret, otpType)
	res := cr.Match{Match: ok, Message: cr.NoMessageStr}
	if ok == false && err != nil {
		res.Message = fmt.Sprintf("%v", err)
	}
	response.WriteHeaderAndEntity(http.StatusOK, res)
}
