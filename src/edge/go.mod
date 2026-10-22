module github.com/phuchoang2603/payment-gateway-fraud-detection/edge

go 1.26.0

require (
	golang.org/x/text v0.43.0
	google.golang.org/grpc v1.83.0
	google.golang.org/protobuf v1.36.12
)

require (
	golang.org/x/net v0.61.0 // indirect
	golang.org/x/sys v0.49.0 // indirect
	google.golang.org/genproto/googleapis/rpc v0.0.0-20261005182115-fad411399dd8 // indirect
)

require github.com/phuchoang2603/payment-gateway-fraud-detection/shared v0.0.0

replace github.com/phuchoang2603/payment-gateway-fraud-detection/shared => ../../shared
