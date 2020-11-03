module main

go 1.14

require (

	cloud.google.com/go/pubsub v1.6.1 // indirect
	github.com/cloudevents/sdk-go/protocol/pubsub/v2 v2.2.0 // indirect
	github.com/cloudevents/sdk-go/v2 v2.2.0 // indirect
	github.com/gogo/protobuf v1.3.1 // indirect
	github.com/google/tink/go v1.5.0 // indirect
	github.com/googleapis/google-cloudevents-go v0.0.0-20200710170715-c543fb3cb993 // indirect
	github.com/salrashid123/ce_envelope_extension v0.0.0

)

replace (

	github.com/salrashid123/ce_envelope_extension => ./ce_envelope_extension

)
