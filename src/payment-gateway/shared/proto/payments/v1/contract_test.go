package paymentsv1_test

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	paymentsv1 "github.com/phuchoang2603/payment-gateway-fraud-detection/shared/proto/payments/v1"
	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/proto"
)

func TestContractJSONRoundTrip(t *testing.T) {
	files, err := filepath.Glob("../../../../../../contracts/payments/v1/testdata/*.json")
	if err != nil || len(files) == 0 {
		t.Fatalf("contract fixtures: %v", err)
	}
	for _, file := range files {
		t.Run(filepath.Base(file), func(t *testing.T) {
			raw, err := os.ReadFile(file)
			if err != nil {
				t.Fatal(err)
			}
			var history struct {
				Events []json.RawMessage `json:"events"`
			}
			if err := json.Unmarshal(raw, &history); err != nil {
				t.Fatal(err)
			}
			if len(history.Events) == 0 {
				t.Fatal("empty history")
			}
			for _, raw := range history.Events {
				event := &paymentsv1.PaymentEvent{}
				if err := protojson.Unmarshal(raw, event); err != nil {
					t.Fatal(err)
				}
				encoded, err := protojson.Marshal(event)
				if err != nil {
					t.Fatal(err)
				}
				decoded := &paymentsv1.PaymentEvent{}
				if err := protojson.Unmarshal(encoded, decoded); err != nil {
					t.Fatal(err)
				}
				if event.GetPayload() == nil || !proto.Equal(event, decoded) {
					t.Fatal("contract JSON lost event fields")
				}
			}
		})
	}
}
