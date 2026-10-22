package httptransport

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"errors"
	"net/http"
	"net/url"
	"regexp"

	"github.com/phuchoang2603/payment-gateway-fraud-detection/edge/internal/application"
	"github.com/phuchoang2603/payment-gateway-fraud-detection/edge/internal/domain"
)

const (
	checkoutCookieName = "checkout_token"
	maxFormBody        = 8 << 10
	cookieMaxAge       = 3600
)

var sessionIDPattern = regexp.MustCompile(`^[A-Za-z0-9-]{1,64}$`)

func (s *Server) csrfToken(sessionID, idempotencyKey string) string {
	mac := hmac.New(sha256.New, s.opts.CookieKey)
	mac.Write([]byte("csrf\x00" + sessionID + "\x00" + idempotencyKey))
	return base64.RawURLEncoding.EncodeToString(mac.Sum(nil))
}

func (s *Server) validCSRF(sessionID, idempotencyKey, supplied string) bool {
	return hmac.Equal([]byte(s.csrfToken(sessionID, idempotencyKey)), []byte(supplied))
}

func (s *Server) sameSite(r *http.Request) bool {
	if r.Header.Get("Sec-Fetch-Site") == "cross-site" {
		return false
	}
	if origin := r.Header.Get("Origin"); origin != "" {
		return origin == s.origin
	}
	if referer := r.Header.Get("Referer"); referer != "" {
		parsed, err := url.Parse(referer)
		return err == nil && parsed.Scheme+"://"+parsed.Host == s.origin
	}
	return false
}

func (s *Server) sessionToken(w http.ResponseWriter, r *http.Request) (sessionID, token string, ok bool) {
	sessionID = r.PathValue("session_id")
	if !sessionIDPattern.MatchString(sessionID) {
		s.renderFailure(w, r, domain.New(domain.ErrNotFound, "malformed session id"))
		return "", "", false
	}
	cookie, err := r.Cookie(checkoutCookieName)
	if err != nil || cookie.Value == "" {
		s.renderFailure(w, r, domain.New(domain.ErrUnauthorized, "missing checkout cookie"))
		return "", "", false
	}
	return sessionID, cookie.Value, true
}

func (s *Server) exchangeToken(w http.ResponseWriter, r *http.Request, sessionID, token string) {
	if _, err := s.opts.Service.Checkout(r.Context(), sessionID, token); err != nil {
		s.renderFailure(w, r, err)
		return
	}
	clean := "/checkout/" + url.PathEscape(sessionID)
	http.SetCookie(w, &http.Cookie{
		Name:     checkoutCookieName,
		Value:    token,
		Path:     clean,
		MaxAge:   cookieMaxAge,
		HttpOnly: true,
		Secure:   s.opts.CookieSecure,
		SameSite: http.SameSiteLaxMode,
	})
	w.Header().Set("Referrer-Policy", "no-referrer")
	w.Header().Set("Cache-Control", "no-store")
	http.Redirect(w, r, clean, http.StatusSeeOther)
}

func (s *Server) checkoutPage(w http.ResponseWriter, r *http.Request) {
	sessionID := r.PathValue("session_id")
	if token := r.URL.Query().Get("token"); token != "" && sessionIDPattern.MatchString(sessionID) {
		s.exchangeToken(w, r, sessionID, token)
		return
	}
	sessionID, token, ok := s.sessionToken(w, r)
	if !ok {
		return
	}
	view, err := s.opts.Service.Checkout(r.Context(), sessionID, token)
	if err != nil {
		s.renderFailure(w, r, err)
		return
	}
	switch view.State {
	case application.StateSucceeded:
		s.redirectToMerchant(w, r, view.Payment)
	case application.StateExpired:
		s.renderFailure(w, r, domain.New(domain.ErrGone, "checkout session is closed"))
	case application.StatePending:
		s.render(w, http.StatusOK, "pending", pageData{SessionID: sessionID})
	default:
		s.render(w, http.StatusOK, "form", s.formData(view))
	}
}

func (s *Server) redirectToMerchant(w http.ResponseWriter, r *http.Request, payment application.Payment) {
	if domain.ValidateReturnURL(payment.ReturnURL) != nil {
		s.renderMessage(w, http.StatusOK, "Payment complete", "Your payment succeeded. You can close this page.")
		return
	}
	w.Header().Set("Referrer-Policy", "no-referrer")
	w.Header().Set("Cache-Control", "no-store")
	http.Redirect(w, r, payment.ReturnURL, http.StatusSeeOther)
}

func (s *Server) checkoutPay(w http.ResponseWriter, r *http.Request) {
	if !s.sameSite(r) {
		s.renderMessage(w, http.StatusForbidden, "Request blocked", "This request did not come from the checkout page.")
		return
	}
	sessionID, token, ok := s.sessionToken(w, r)
	if !ok {
		return
	}
	r.Body = http.MaxBytesReader(w, r.Body, maxFormBody)
	if err := r.ParseForm(); err != nil {
		s.renderMessage(w, http.StatusBadRequest, "Invalid request", "The form could not be read.")
		return
	}
	key := r.PostForm.Get("idempotency_key")
	if key == "" || !s.validCSRF(sessionID, key, r.PostForm.Get("csrf_token")) {
		s.renderMessage(w, http.StatusForbidden, "Request blocked", "This form is no longer valid. Reload the checkout page.")
		return
	}
	device := r.PostForm.Get("device_fingerprint")
	if !deviceFingerprintRegex.MatchString(device) {
		device = ""
	}
	request := application.PayRequest{
		SessionID:         sessionID,
		CheckoutToken:     token,
		IdempotencyKey:    key,
		CardNumber:        r.PostForm.Get("card_number"),
		DeviceFingerprint: device,
		UserAgent:         r.UserAgent(),
	}
	r.PostForm.Del("card_number")
	r.Form.Del("card_number")
	if address, found := s.clientAddress(r); found {
		request.ClientIP = address
	}
	result, err := s.opts.Service.Pay(r.Context(), request)
	switch {
	case errors.Is(err, domain.ErrInvalidCard):
		s.rerender(w, r, sessionID, token, http.StatusBadRequest, "Invalid card. Use one of the accepted test cards.")
	case err != nil:
		s.renderFailure(w, r, err)
	default:
		s.renderAttempt(w, r, result)
	}
}

func (s *Server) renderAttempt(w http.ResponseWriter, r *http.Request, result application.PayResult) {
	switch result.Attempt.Status {
	case application.AttemptSucceeded:
		s.redirectToMerchant(w, r, result.Payment)
	case application.AttemptPending:
		s.render(w, http.StatusOK, "pending", pageData{SessionID: result.Payment.SessionID})
	case application.AttemptDeclined:
		data := s.formData(application.CheckoutView{Payment: result.Payment})
		data.Reason = declineText(result.Attempt.Reason)
		s.render(w, http.StatusOK, "form", data)
	default:
		s.renderFailure(w, r, domain.New(domain.ErrInternal, "unknown attempt status"))
	}
}

func (s *Server) rerender(w http.ResponseWriter, r *http.Request, sessionID, token string, status int, problem string) {
	view, err := s.opts.Service.Checkout(r.Context(), sessionID, token)
	if err != nil {
		s.renderFailure(w, r, err)
		return
	}
	data := s.formData(view)
	data.Error = problem
	s.render(w, status, "form", data)
}

type statusJSON struct {
	State       string `json:"state"`
	RedirectURL string `json:"redirect_url,omitempty"`
}

func (s *Server) checkoutStatus(w http.ResponseWriter, r *http.Request) {
	sessionID, token, ok := s.sessionToken(w, r)
	if !ok {
		return
	}
	view, err := s.opts.Service.Checkout(r.Context(), sessionID, token)
	if err != nil {
		s.fail(w, r, err)
		return
	}
	body := statusJSON{State: string(view.State)}
	if view.State == application.StateSucceeded {
		body.RedirectURL = view.Payment.ReturnURL
	}
	writeJSON(w, http.StatusOK, body)
}
