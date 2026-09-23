package server

import (
	"context"
	"io"
	"log/slog"
	"net/http"

	fraudv1 "github.com/phuchoang2603/realtime-credit-card-fraud-detection/edge/gen/fraud/v1"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/encoding/protojson"
)

func (s *Server) predict(w http.ResponseWriter, r *http.Request) {
	body, err := io.ReadAll(http.MaxBytesReader(w, r.Body, 64<<10))
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"detail": "Invalid or oversized request"})
		return
	}
	request := new(fraudv1.PredictRequest)
	if err := protojson.Unmarshal(body, request); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"detail": "Invalid prediction request"})
		return
	}
	ctx, cancel := context.WithTimeout(r.Context(), s.config.PredictionTimeout)
	defer cancel()
	ctx = metadata.AppendToOutgoingContext(ctx, "x-request-id", w.Header().Get("X-Request-ID"))
	for _, name := range []string{"traceparent", "tracestate"} {
		if value := r.Header.Get(name); value != "" {
			ctx = metadata.AppendToOutgoingContext(ctx, name, value)
		}
	}
	result, err := s.fraud.Predict(ctx, request)
	if err != nil {
		code, detail := predictionError(status.Code(err))
		slog.Warn("prediction RPC failed", "request_id", w.Header().Get("X-Request-ID"), "code", status.Code(err).String())
		writeJSON(w, code, map[string]string{"detail": detail})
		return
	}
	// Public response names are owned by the HTTP edge; the internal wire is protobuf.
	writeJSON(w, http.StatusOK, struct {
		IsFraud     bool    `json:"is_fraud"`
		Probability float64 `json:"fraud_probability"`
	}{result.GetIsFraud(), result.GetFraudProbability()})
}

func predictionError(code codes.Code) (int, string) {
	switch code {
	case codes.InvalidArgument:
		return http.StatusBadRequest, "Invalid or missing transaction features"
	case codes.PermissionDenied:
		return http.StatusForbidden, "Transaction rejected by fraud policy"
	case codes.Unavailable:
		return http.StatusServiceUnavailable, "Fraud service unavailable"
	case codes.DeadlineExceeded:
		return http.StatusGatewayTimeout, "Prediction deadline exceeded"
	case codes.ResourceExhausted:
		return http.StatusServiceUnavailable, "Fraud service busy"
	case codes.Canceled:
		return http.StatusRequestTimeout, "Prediction canceled"
	default:
		return http.StatusBadGateway, "Prediction failed"
	}
}
