// Package quota is an ordinary Go package with an observable logging effect.
package quota

import "log"

// AdmitRequest is the application boundary consumed by Admit.
type AdmitRequest struct {
	Current  int64
	Delta    int64
	Limit    int64
	Subject  string
	Evidence []byte
}

// AdmitResponse is the application boundary produced by Admit.
type AdmitResponse struct {
	Accepted bool
	Subject  string
	Evidence []byte
}

// AdmitError is the ordinary Go error returned for invalid requests.
type AdmitError struct{ Message string }

func (e AdmitError) Error() string { return e.Message }

// WithinLimit is kept separate so Seme must preserve and lower an ordinary
// Go-to-Go call rather than recognizing one monolithic handler expression.
func WithinLimit(current, delta, limit int64) bool {
	return current+delta <= limit
}

// Admit evaluates the quota policy and records its decision.
func Admit(request AdmitRequest) (AdmitResponse, error) {
	if request.Subject == "" {
		return AdmitResponse{}, AdmitError{Message: "subject required"}
	}
	accepted := WithinLimit(request.Current, request.Delta, request.Limit)
	log.Printf("quota.accepted=%t", accepted)
	return AdmitResponse{
		Evidence: request.Evidence,
		Accepted: accepted,
		Subject:  request.Subject,
	}, nil
}
