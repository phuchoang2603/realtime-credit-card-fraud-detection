package server

import (
	"context"
	"net"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	fraudv1 "github.com/phuchoang2603/realtime-credit-card-fraud-detection/edge/gen/fraud/v1"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
)

type predictionService struct {
	fraudv1.UnimplementedFraudServiceServer
	call func(context.Context, *fraudv1.PredictRequest) (*fraudv1.PredictResponse, error)
}

func (s predictionService) Predict(ctx context.Context, request *fraudv1.PredictRequest) (*fraudv1.PredictResponse, error) {
	return s.call(ctx, request)
}

func TestPredictionAdapter(t *testing.T) {
	for _, scenario := range []string{"success", "invalid-json", "unknown-field", "policy", "unavailable", "deadline", "private-error"} {
		t.Run(scenario, func(t *testing.T) {
			listener, err := net.Listen("tcp", "127.0.0.1:0")
			if err != nil {
				t.Fatal(err)
			}
			backend := grpc.NewServer()
			fraudv1.RegisterFraudServiceServer(backend, predictionService{call: func(ctx context.Context, request *fraudv1.PredictRequest) (*fraudv1.PredictResponse, error) {
				md, _ := metadata.FromIncomingContext(ctx)
				if md.Get("x-request-id")[0] != "public-42" || md.Get("traceparent")[0] != "00-0123456789abcdef0123456789abcdef-0123456789abcdef-01" {
					t.Error("correlation not propagated")
				}
				if _, ok := ctx.Deadline(); !ok {
					t.Error("missing RPC deadline")
				}
				switch scenario {
				case "policy":
					return nil, status.Error(codes.PermissionDenied, "private rule details")
				case "unavailable":
					return nil, status.Error(codes.Unavailable, "private host")
				case "private-error":
					return nil, status.Error(codes.Internal, "private credentials")
				case "deadline":
					<-ctx.Done()
					return nil, status.FromContextError(ctx.Err()).Err()
				}
				if request.TxAmount == nil || request.GetTxAmount() != 0 {
					t.Error("explicit zero presence lost")
				}
				return &fraudv1.PredictResponse{IsFraud: false, FraudProbability: 0.2}, nil
			}})
			go backend.Serve(listener)
			t.Cleanup(backend.Stop)
			s := newTestServer(t)
			conn, err := grpc.NewClient("passthrough:///"+listener.Addr().String(), grpc.WithTransportCredentials(insecure.NewCredentials()))
			if err != nil {
				t.Fatal(err)
			}
			t.Cleanup(func() { _ = conn.Close() })
			s.fraud = fraudv1.NewFraudServiceClient(conn)
			s.config.PredictionTimeout = 100 * time.Millisecond
			body := `{"tx_amount":0}`
			expected := http.StatusOK
			switch scenario {
			case "invalid-json":
				body = `{`
				expected = http.StatusBadRequest
			case "unknown-field":
				body = `{"unknown":1}`
				expected = http.StatusBadRequest
			case "policy":
				expected = http.StatusForbidden
			case "unavailable":
				expected = http.StatusServiceUnavailable
			case "deadline":
				expected = http.StatusGatewayTimeout
			case "private-error":
				expected = http.StatusBadGateway
			}
			request := httptest.NewRequest(http.MethodPost, "/predict", strings.NewReader(body))
			request.Header.Set("X-Request-ID", "public-42")
			request.Header.Set("traceparent", "00-0123456789abcdef0123456789abcdef-0123456789abcdef-01")
			response := httptest.NewRecorder()
			s.Handler().ServeHTTP(response, request)
			if response.Code != expected || strings.Contains(response.Body.String(), "private") {
				t.Fatalf("response %d %s", response.Code, response.Body.String())
			}
			if scenario == "success" && !strings.Contains(response.Body.String(), `"fraud_probability":0.2`) {
				t.Fatal(response.Body.String())
			}
		})
	}
}
