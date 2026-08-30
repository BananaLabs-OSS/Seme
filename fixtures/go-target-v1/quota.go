// Package quota is an ordinary Go package with an observable logging effect.
package quota

import "log"

// AdmitRequest is the application boundary consumed by Admit.
type AdmitRequest struct {
	Current int64
	Delta   int64
	Limit   int64
}

// AdmitResponse is the application boundary produced by Admit.
type AdmitResponse struct {
	Accepted bool
}

// Admit evaluates the quota policy and records its decision.
func Admit(request AdmitRequest) AdmitResponse {
	accepted := request.Current+request.Delta <= request.Limit
	log.Printf("quota.accepted=%t", accepted)
	return AdmitResponse{Accepted: accepted}
}
