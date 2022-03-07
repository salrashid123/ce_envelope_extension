package main

import (
	"context"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"flag"
	"fmt"
	"net/http"

	"github.com/golang/glog"
	"github.com/google/uuid"

	cloudevents "github.com/cloudevents/sdk-go/v2"
	"github.com/cloudevents/sdk-go/v2/types"
	extensions "github.com/salrashid123/ce_envelope_extension"
	//cehttp "github.com/cloudevents/sdk-go/v2/protocol/http"
)

var (
	projectID        = flag.String("projectID", "", "ProjectID for topic and subscriber")
	keyType          = flag.String("keyType", "SHARED", "Key Type used (KMS|TINK|SHARED)")
	aad              = flag.String("aad", "foo", "AAD to use")
	dek              = flag.String("dek", "gUkXp2s5v8y/B?E(H+KbPeShVmYq3t6w", "Raw key  used (SHARED)")
	keyUri           = flag.String("keyUri", "gcp-kms://projects/mineral-minutia-820/locations/us-central1/keyRings/pubsub-kr/cryptoKeys/key1", "Tink KMS Key URL pointing to the GCP or AWS KMS KEK Key to use")
	mode             = flag.String("mode", "server", "(required for  mode=client|server")
	serverAddress    = flag.String("serverAddress", "http://localhost:8080", "(required for mode=server)")
	serverListenPort = flag.Int("serverListenPort", 8080, "(required for mode=server)")
	keys             map[string]extensions.EncryptionExtension
)

const (
	encryptedEventType = "com.github.salrashid123.ce_envelope_extension"
)

// TODO: use middleware to decrypt the data...
func log(h http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		glog.V(30).Infof("Before")
		h.ServeHTTP(w, r) // call original
		glog.V(30).Infof("After")
	})
}

func main() {

	flag.Parse()

	keys = make(map[string]extensions.EncryptionExtension)
	if *mode == "client" {

		protocol, err := cloudevents.NewHTTP(cloudevents.WithTarget(*serverAddress))
		if err != nil {
			glog.Fatalf("Error %v, ", err)
		}

		c, err := cloudevents.NewClient(protocol, cloudevents.WithTimeNow(), cloudevents.WithUUIDs())
		if err != nil {
			glog.Fatalf("failed to create server, %v", err)
		}

		if err != nil {
			glog.Fatalf("failed to create client, %v", err)
		}

		var tt extensions.EncType

		if *keyType == "TINK" {
			tt = extensions.TINK
		} else if *keyType == "KMS" {
			tt = extensions.KMS
		} else if *keyType == "SHARED" {
			tt = extensions.SHARED
		}

		var eet *extensions.EncryptionExtension
		msg := "foo"
		for i := 1; i < 10; i++ {
			if tt == extensions.SHARED {
				eet, err = extensions.NewEncryptionExtension(&extensions.EncryptionExtension{
					DEK:  *dek,
					Type: tt,
				})
				if err != nil {
					glog.Fatalf("failed to create Extension %s", err.Error())
				}
			} else {
				if eet == nil || (i%4 == 0) {
					glog.V(10).Infof("Generating New Key")
					eet, err = extensions.NewEncryptionExtension(&extensions.EncryptionExtension{
						KeyUri: *keyUri,
						Type:   tt,
					})
					if err != nil {
						glog.Fatalf("failed to create Extension %s", err.Error())
					}

				} else {
					glog.V(10).Infof("Using Existing Key")
				}
			}
			h := sha256.New()
			h.Write([]byte(eet.DEK))
			glog.V(10).Infof("     DEK sha256 [%s]", fmt.Sprintf("%x", h.Sum(nil)))

			event := cloudevents.NewEvent()
			event.SetID(uuid.New().String())
			event.SetType(encryptedEventType)
			event.SetSource("github.com/salrashid123/tink_samples")

			out, err := json.Marshal(eet.GetType())
			if err != nil {
				glog.Fatalf("Failed to JSON Marshall EncryptionExtension Type, %s", err.Error())
			}

			event.SetExtension(extensions.EncryptionExtensionName, string(out))

			uu := fmt.Sprintf("%v %d", msg, i)
			glog.V(10).Infof("     Encrypting data: [%s]", uu)
			ret, err := eet.Encrypt([]byte(uu), []byte(*aad))
			if err != nil {
				glog.Fatalf("failed to set data, %s", err.Error())
			}

			event.SetData(cloudevents.TextPlain, base64.StdEncoding.EncodeToString(ret))

			if result := c.Send(context.Background(), event); !cloudevents.IsACK(result) {
				glog.Fatalf("failed to send, %v", result)
			}

		}

	}
	if *mode == "server" {
		protocol, err := cloudevents.NewHTTP(cloudevents.WithPort(*serverListenPort), cloudevents.WithMiddleware(log))
		if err != nil {
			glog.Fatalf("Error %v, ", err)
		}

		c, err := cloudevents.NewClient(protocol, cloudevents.WithTimeNow(), cloudevents.WithUUIDs())
		if err != nil {
			glog.Fatalf("failed to create server, %v", err)
		}

		err = c.StartReceiver(context.Background(), Receive)
		if err != nil {
			glog.Fatalf("failed to StartReceiver, %v", err)
		}
	}
}

func Receive(ctx context.Context, event cloudevents.Event) error {
	glog.V(10).Infof("  EventID: %s\n", event.ID())
	glog.V(10).Infof("  EventType: %s \n", event.Type())
	glog.V(10).Infof("  Event Context: %+v\n", event.Context)

	switch event.Type() {
	case encryptedEventType:
		ic, err := types.ToString(event.Extensions()[extensions.EncryptionExtensionName])
		if err != nil {
			glog.Errorf("Extension Error %v", err)
			return err
		}

		eetconf := &extensions.EncryptionExtension{}
		err = json.Unmarshal([]byte(ic), eetconf)
		if err != nil {
			glog.Errorf("Extension Error %v", err)
			return err
		}

		if *keyType == "SHARED" {
			eetconf.DEK = *dek
		}

		var eet *extensions.EncryptionExtension

		if val, ok := keys[eetconf.DEK]; ok {
			glog.V(10).Infof("Using Existing key ")
			eet = &val
		} else {
			glog.V(10).Infof("Initialize new key ")
			eet, err = extensions.NewEncryptionExtension(eetconf)
			if err != nil {
				glog.Errorf("Extension Error %v", err)
				return err
			}
			keys[eetconf.DEK] = *eet
		}
		h := sha256.New()
		h.Write([]byte(eet.DEK))
		glog.V(10).Infof("     DEK sha256 value [%s]", fmt.Sprintf("%x", h.Sum(nil)))

		glog.V(20).Infof("HTTP Message %s\n", base64.StdEncoding.EncodeToString(event.Data()))

		dec, err := base64.StdEncoding.DecodeString(string(event.Data()))
		if err != nil {
			glog.Errorf("Error Decrypting HTTP Data %v", err)
			return err
		}

		s, err := eet.Decrypt(dec, []byte(*aad))
		if err != nil {
			glog.Errorf("Error Decrypting HTTP Data %v", err)
			return err
		}
		glog.V(10).Infof("  Event Data: %s\n", string(s))
	}
	return nil
}
