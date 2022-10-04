package main

import (
	"context"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"time"

	"github.com/golang/glog"

	cepubsub "github.com/cloudevents/sdk-go/protocol/pubsub/v2"

	pscontext "github.com/cloudevents/sdk-go/protocol/pubsub/v2/context"
	cloudevents "github.com/cloudevents/sdk-go/v2"
	"github.com/cloudevents/sdk-go/v2/event"
	"github.com/cloudevents/sdk-go/v2/types"
	"github.com/google/uuid"
	extensions "github.com/salrashid123/ce_envelope_extension/handler"
	p "google.golang.org/api/pubsub/v1"
)

var (
	projectID = flag.String("projectID", "", "ProjectID for topic and subscriber")
	topicID   = flag.String("topicID", "", "Topic run-events")
	keyType   = flag.String("keyType", "SHARED", "Key Type used (KMS|TINK|SHARED)")
	aad       = flag.String("aad", "foo", "AAD to use")
	dek       = flag.String("dek", "gUkXp2s5v8y/B?E(H+KbPeShVmYq3t6w", "Raw key  used (SHARED)")
	subID     = flag.String("subID", "", "Subscription cloud-events-auditlog | cloud-events-pubsub")
	keyUri    = flag.String("keyUri", "gcp-kms://projects/mineral-minutia-820/locations/us-central1/keyRings/pubsub-kr/cryptoKeys/key1", "Tink KMS Key URL pointing to the GCP or AWS KMS KEK Key to use")
	mode      = flag.String("mode", "subscribe", "(required for mode=mode) mode=subscribe|publish")

	keys map[string]extensions.EncryptionExtension
)

const (
	encryptedEventType = "com.github.salrashid123.ce_envelope_extension"
)

func main() {

	flag.Parse()

	keys = make(map[string]extensions.EncryptionExtension)
	if *mode == "subscribe" {
		err := pullMsgs(*projectID, *topicID, *subID)
		if err != nil {
			panic(err)
		}
	}
	if *mode == "publish" {
		err := sendMsg("foo", *projectID, *topicID)
		if err != nil {
			panic(err)
		}
	}
}

func receive(ctx context.Context, event event.Event) error {
	glog.V(20).Infof("Event Context: %+v\n", event.Context)
	glog.V(20).Infof("Protocol Context: %+v\n", pscontext.ProtocolContextFrom(ctx))
	glog.V(20).Infof("EventID %s\n", event.ID())

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

		h := sha256.New()
		h.Write([]byte(eetconf.DEK))
		dekHash := hex.EncodeToString(h.Sum(nil))
		glog.V(10).Infof("     DEK sha256 value [%s]", dekHash)

		if val, ok := keys[dekHash]; ok {
			glog.V(10).Infof("Using Existing key ")
			eet = &val
		} else {
			glog.V(10).Infof("Initialize new key ")
			eet, err = extensions.NewEncryptionExtension(eetconf)
			if err != nil {
				glog.Errorf("Extension Error %v", err)
				return err
			}
			keys[dekHash] = *eet
		}

		pubsubData := &p.PubsubMessage{}
		if err := event.DataAs(pubsubData); err != nil {
			glog.Errorf("Error unmarshalling loud Event as PububMessage %v", err)
			return err
		}

		glog.V(20).Infof("Pubsub Message %s\n", pubsubData.Data)

		dec, err := base64.StdEncoding.DecodeString(pubsubData.Data)
		if err != nil {
			glog.Errorf("Error Decoding pubsub Data %v", err)
			return err
		}
		s, err := eet.Decrypt(dec, []byte(*aad))
		if err != nil {
			glog.Errorf("Error Decrypting Pubsub Data %v", err)
			return err
		}

		glog.V(10).Infof("Decrypted Pubsub Message data [%s]\n", string(s))

	default:
		return errors.New("could not parse Cloud Event TYpe")
	}
	return nil
}

func pullMsgs(projectId, topicID, subID string) error {
	t, err := cepubsub.New(context.Background(),
		cepubsub.WithProjectID(projectId),
		cepubsub.WithTopicID(topicID),
		cepubsub.WithSubscriptionID(subID))

	if err != nil {
		return err
	}
	c, err := cloudevents.NewClient(t)
	if err != nil {
		return err
	}

	glog.V(10).Infof("Created client, listening...")
	ctx := context.Background()
	if err := c.StartReceiver(ctx, receive); err != nil {
		return err
	}
	return nil
}

func sendMsg(msg string, projectID, topicID string) error {
	t, err := cepubsub.New(context.Background(),
		cepubsub.WithProjectID(projectID),
		cepubsub.WithTopicID(topicID))
	if err != nil {
		glog.Fatalf("failed to create pubsub transport, %s", err.Error())
	}
	c, err := cloudevents.NewClient(t, cloudevents.WithTimeNow(), cloudevents.WithUUIDs())
	if err != nil {
		glog.Fatalf("failed to create client, %s", err.Error())
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
					glog.Fatalf("failed to set data, %s", err.Error())
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

		err = event.SetData("application/json", &p.PubsubMessage{
			Data: base64.StdEncoding.EncodeToString(ret),
		})
		if err != nil {
			glog.Fatalf("failed to set data, %s", err.Error())
		}

		glog.V(20).Infof("%v\n", event)
		if result := c.Send(context.Background(), event); cloudevents.IsUndelivered(result) {
			glog.Fatalf("failed to send: %v\n", result.Error())
		} else {
			glog.V(20).Infof("sent, accepted: %t\n", cloudevents.IsACK(result))
		}
		time.Sleep(time.Duration(1 * time.Second))
	}
	return nil
}
