module main

go 1.14

require (
	github.com/cloudevents/sdk-go/protocol/pubsub/v2 v2.12.0
	github.com/cloudevents/sdk-go/v2 v2.12.0
	github.com/golang/glog v0.0.0-20160126235308-23def4e6c14b
	github.com/google/tink/go v1.5.0 // indirect
	github.com/google/uuid v1.1.2
	github.com/salrashid123/ce_envelope_extension/handler v0.0.0
	google.golang.org/api v0.84.0

)

replace github.com/salrashid123/ce_envelope_extension/handler => ../handler
