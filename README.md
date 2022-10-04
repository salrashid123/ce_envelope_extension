
## Cloud Events envelope encryption extension
     (for Pubsub and HTTP)


Sample [Cloud Events](https://cloudevents.io/) extension that defines how to decrypt _payloads_ encrypted using Shared AES keys, GCP KMS and TINK Envelope Encryption.

The Cloud [Events Extensions](https://github.com/cloudevents/spec/blob/v1.0/documented-extensions.md) really just describes metadata to include into headers of any given message and cannot by itself do any processing of the enclosed payload data.

This particular extension first encrypts the `Data` area of a PubSub or HTTP message using either a provided symmetric key, Google Cloud KMS-backed DEK or  GCP KMS backed TINK Symmetric key.  Once encrypted, it places the key reference value and the encrypted Data Encryption Key (DEK) as the metadata value for transmission.

The receiver of the event will decode the extension and then use the encrypted key or reference to the shared one to finally decode the message.

Since the extension framework is just metadata and not an 'interceptor' for pre/post processing messages, the decryption has to be done in code manually...

There are two protocol implementations/samples here:  http and GCP pubsub clients.

This sample sets up a pubsub topic with a subscriber as well as a symmetric GCP KMS key to use for encryption.

On startup, the clients publisher will generate 10 messages but only rotate the derived AES DEK key three times.  This is done to avoid repeated calls to kms to rewrap the DEK (meaning, you are using the DEK a couple of times before its rotated out). The DEK and TINK keys are embedded into the cloud event payload.

The receiver will decrypt the DEK and cache it locally.  If multiple messages are encrypted with the same DEK, the cached version (if found) will be used so as to not make a new KMS api call via KMS Client or TINK or RAW:


KeyTypes

  * `KMS`: use KMS to wrap the raw AES key
  * `TINK`: wrap a TINK key with KMS
  * `SHARED`: use a shared AES key

A sample event log is shown below showing for PubSub and HTTP


## QuickStart

As a quick sample, we the following runs a simple http cloudevents client server locally using this extension with a `SHARED` key

What the following shows is a http client and server for eventarc with this extension.

The client encrypts the data for each message with a `SHARED` key.  The extension and hash of the shared key is shown as well
as the encrypted data.

The server will load the same shared key and extension. Then decrypt and display the data 

```log
cd http/
go run main.go \
  --mode server \
   --keyType=SHARED \
   --dek="gUkXp2s5v8y/B?E(H+KbPeShVmYq3t6w" \
   -v 20 -alsologtostderr

I0306 12:44:59.040265   94882 main.go:154]   EventID: b959f405-9ea5-4c7a-b25e-d26ca3770e35
I0306 12:44:59.040434   94882 main.go:155]   EventType: com.github.salrashid123.ce_envelope_extension 
I0306 12:44:59.040439   94882 main.go:156]   Event Context: Context Attributes,
  specversion: 1.0
  type: com.github.salrashid123.ce_envelope_extension
  source: github.com/salrashid123/tink_samples
  id: b959f405-9ea5-4c7a-b25e-d26ca3770e35
  time: 2022-03-06T17:44:59.039594428Z
  datacontenttype: text/plain
Extensions,
  envelopeencryption: {"key_uri":"","dek":"69fefc1758c357c97544a2891c93d336118a5375f325aa6f46ccb5dfa454c8fc","type":2}

I0306 12:44:59.040488   94882 main.go:183] Initialize new key 
I0306 12:44:59.040505   94882 main.go:193]      DEK sha256 value [bafae7a0566f4680e471ec9bfa66579781f946ec4b5aed6d240b05c5fca55cd4]
I0306 12:44:59.040512   94882 main.go:195] HTTP Message WXZ5Z0lpeHJYbmtGMVlLUVh1azBMVUtOQ1FzVTZ3dFVtQmE0R3Fid3RRNWo=
I0306 12:44:59.040519   94882 main.go:208]   Event Data: foo 1
```

Notice the `Extensions -> envelopencryption` section.  That is the metadata that gets sent with each data. The `encryptedKeyset` is the KMS wrapped key that uses the Key Encryption Key (KEK) defined by the `key_uri`.  The inner, encrypted KEK is used to encrypt the `Data` section.

On the subscriber side, the  KEK gets cached locally to avoid repeated lookups.  If the subscriber does not find the KEK in cache,it will use TINK to do envelope decryption on the Data and then save the KEK.  If the keys are rotated by the publisher, no problem,it will detect that the KEK is not in cache and continue.


To run the http client:

```log
 go run main.go \
   --mode client \
   --keyType=SHARED \
   --dek="gUkXp2s5v8y/B?E(H+KbPeShVmYq3t6w" \
   -v 20 -alsologtostderr

12:44:59.039323   94982 main.go:105]      DEK sha256 [bafae7a0566f4680e471ec9bfa66579781f946ec4b5aed6d240b05c5fca55cd4]
I0306 12:44:59.039582   94982 main.go:120]      Encrypting data: [foo 1]
```

---

>> Note about KMS operations: the fact we're using TINK Encrypted KMS backed keys, each operation invokes the KMS api.  If you would rather just use KMS to wrap a secret, pass the reference to KMS and wrap the key manually and rotate as described here below and shows separately with pubsub in : 

- [Message Payload Encryption in Google Cloud Pub/Sub (Part 4: Envelope Encryption with Google Key Management System and PubSub)](https://github.com/salrashid123/gcp_pubsub_message_encryption/tree/master/4_kms_dek)



for more info, see

- [Tink Samples](https://github.com/salrashid123/tink_samples)
- [Message Payload Encryption in Google Cloud Pub/Sub (Part 4: Envelope Encryption with Google Key Management System and PubSub)](https://github.com/salrashid123/gcp_pubsub_message_encryption/tree/master/4_kms_dek)


### Setup

First setup KMS, PubSub

```bash
# in each window
export PROJECT_ID=`gcloud config get-value core/project`
export PROJECT_NUMBER=`gcloud projects describe $PROJECT_ID --format='value(projectNumber)'`


gcloud pubsub topics create crypt-topic

gcloud pubsub subscriptions create crypt-subscribe --topic=crypt-topic

gcloud kms keyrings create pubsub-kr --location us-central1

gcloud kms keys  create key1 --keyring pubsub-kr --location us-central1 --purpose encryption 
```

Then just run the client and server

The following will show the key rotation frequency and the hash value of the DEK used. Thats just shown so that you know which key is being used for encryption.

There are three supported encryption mechanism:  using `Shared` key, `KMS` wrapped `TINK` key and plain `KMS envelope encryption`

## HTTP

The following http cloud events sample will run a client/server in each of the three encryption modes

### SHARED


```bash
cd http/
 go run main.go \
  --mode server \
   --keyType=SHARED \
   --dek="gUkXp2s5v8y/B?E(H+KbPeShVmYq3t6w" \
   -v 20 -alsologtostderr

 go run main.go \
   --mode client \
   --serverAddress http://localhost:8080/ \
   --keyType=SHARED \
   --dek="gUkXp2s5v8y/B?E(H+KbPeShVmYq3t6w" \
   -v 20 -alsologtostderr
```

### KMS

```bash
cd http
go run main.go \
  --mode server \
   --projectID $PROJECT_ID \
   --keyType=KMS \
   --keyUri=projects/$PROJECT_ID/locations/us-central1/keyRings/pubsub-kr/cryptoKeys/key1 \
   -v 20 -alsologtostderr

go run main.go \
   --mode client \
   --serverAddress http://localhost:8080/ \
   --keyType=KMS \
   --keyUri=projects/$PROJECT_ID/locations/us-central1/keyRings/pubsub-kr/cryptoKeys/key1 \
   -v 20 -alsologtostderr
```

The encrypted envelope as displayed on the server would look like the following where the `DEK` is encrypted with the KMS key

```log
I0306 12:47:34.765568   95349 main.go:154]   EventID: 0c132bd6-1b59-4764-9742-e848fd3fde34
I0306 12:47:34.766245   95349 main.go:155]   EventType: com.github.salrashid123.ce_envelope_extension 
I0306 12:47:34.766269   95349 main.go:156]   Event Context: Context Attributes,
  specversion: 1.0
  type: com.github.salrashid123.ce_envelope_extension
  source: github.com/salrashid123/tink_samples
  id: 0c132bd6-1b59-4764-9742-e848fd3fde34
  time: 2022-03-06T17:47:34.763541417Z
  datacontenttype: text/plain
Extensions,
  envelopeencryption: {"key_uri":"projects/mineral-minutia-820/locations/us-central1/keyRings/pubsub-kr/cryptoKeys/key1","dek":"CiQAVjnoXfjK/E+jlTiv5iSzhKk9Rr6kV1uivkrLLydU7aRI7UcSSQBxDtBC/6nx0JwU7T/75EDVLH7xa/UbKhEovCKGUhW8GxlaTlUjo/m6GPrlvgPsKDumARwiBrtTV/QsbZAwk27gKvJFEykMGnE=","type":3}

I0306 12:47:34.766416   95349 main.go:183] Initialize new key 
I0306 12:47:35.094562   95349 main.go:193]      DEK sha256 value [d09df6abac80525345f20a7e0ec57949f1375d480b32922eb0c372250e33e696]
I0306 12:47:35.094598   95349 main.go:195] HTTP Message d2VHdXFRSUdST1VycE8zbHAyY3REeXdFVUd6OWpleWw0eVVicWFwTExvY3M=
I0306 12:47:35.094614   95349 main.go:208]   Event Data: foo 1
```



### TINK KMS

```bash
cd http/
go run main.go \
  --mode server \
   --projectID $PROJECT_ID \
   --keyType=TINK \
   --keyUri=gcp-kms://projects/$PROJECT_ID/locations/us-central1/keyRings/pubsub-kr/cryptoKeys/key1 \
   -v 20 -alsologtostderr

go run main.go \
   --mode client \
   --serverAddress http://localhost:8080/ \
   --keyType=TINK \
   --keyUri=gcp-kms://projects/$PROJECT_ID/locations/us-central1/keyRings/pubsub-kr/cryptoKeys/key1 \
   -v 20 -alsologtostderr   
```

The encrypted envelope as displayed on the server would look like the following where the `DEK` is encrypted with the as  TINK Wrapped  KMS key

```log
I0306 12:47:50.721414   95510 main.go:154]   EventID: 14c712f6-cd18-4895-b053-ff476916e8ed
I0306 12:47:50.722129   95510 main.go:155]   EventType: com.github.salrashid123.ce_envelope_extension 
I0306 12:47:50.722147   95510 main.go:156]   Event Context: Context Attributes,
  specversion: 1.0
  type: com.github.salrashid123.ce_envelope_extension
  source: github.com/salrashid123/tink_samples
  id: 14c712f6-cd18-4895-b053-ff476916e8ed
  time: 2022-03-06T17:47:50.719923384Z
  datacontenttype: text/plain
Extensions,
  envelopeencryption: {"key_uri":"gcp-kms://projects/mineral-minutia-820/locations/us-central1/keyRings/pubsub-kr/cryptoKeys/key1","dek":"{\"encryptedKeyset\":\"CiQAVjnoXdAMT/A7gAK1wvLyEH5Uai/famgXMf0WdGWBpdODLlASkwEAcQ7QQuNxQkbL86aoibxMpOjKfdhfpT7dM+TvS9DhoMPrLEUtzymDYsuknUciAM+jC0Lp00yZsckvEFL/zFgi1tpvv93SluvYg8bqSY46T13OdKrxsdmhiT1hBQ7U/RjQpuUQZvBViqFWG+gIQ7MdHjs6epm/RVsHiPxYDoaxtKVDn6/5Rts2KkYaPfEoZ5KLIx4=\",\"keysetInfo\":{\"primaryKeyId\":94079289,\"keyInfo\":[{\"typeUrl\":\"type.googleapis.com/google.crypto.tink.AesGcmKey\",\"status\":\"ENABLED\",\"keyId\":94079289,\"outputPrefixType\":\"TINK\"}]}}","type":0}

I0306 12:47:50.722298   95510 main.go:183] Initialize new key 
I0306 12:47:51.125973   95510 main.go:193]      DEK sha256 value [27c52cc80455cceed0b35219332b491bbe501805c605e5e703b82af684cc6e3a]
I0306 12:47:51.126066   95510 main.go:195] HTTP Message QVFXYmlUa21tMXpwRmtkWlFLUFUzcjMrWnJEbFQrVmIvS250TTBKMmlxMWNTK1NOQm4wPQ==
I0306 12:47:51.126109   95510 main.go:208]   Event Data: foo 1

```

## Pubsub

The following pubsub cloudevents sample will run a client/server in each of the three encryption modes

### Shared

```bash
cd pubsub/
go run main.go \
  --mode subscribe \
  --projectID $PROJECT_ID \
  --topicID crypt-topic \
  --subID crypt-subscribe \
   --keyType=SHARED \
   --dek="gUkXp2s5v8y/B?E(H+KbPeShVmYq3t6w" \
  -v 20 -alsologtostderr

go run main.go \
  --mode publish \
  --projectID $PROJECT_ID \
  --topicID crypt-topic \
   --keyType=SHARED \
   --dek="gUkXp2s5v8y/B?E(H+KbPeShVmYq3t6w" \
  -v 20 -alsologtostderr
```

### KMS

```bash
cd pubsub/
go run main.go \
  --mode subscribe \
  --projectID $PROJECT_ID \
  --topicID crypt-topic \
  --subID crypt-subscribe \
  --keyType=KMS \
  -v 10 -alsologtostderr

go run main.go \
  --mode publish \
  --projectID $PROJECT_ID \
  --topicID crypt-topic \
  --keyType=KMS \
  --keyUri=projects/$PROJECT_ID/locations/us-central1/keyRings/pubsub-kr/cryptoKeys/key1  \
  -v 10 -alsologtostderr
```
### TINK KMS

```bash
go run main.go \
  --mode subscribe \
  --projectID $PROJECT_ID \
  --topicID crypt-topic \
  --subID crypt-subscribe \
  --keyType=TINK \
  --keyUri=gcp-kms://projects/$PROJECT_ID/locations/us-central1/keyRings/pubsub-kr/cryptoKeys/key1 \
  -v 10 -alsologtostderr

go run main.go   --mode publish  \
   --projectID $PROJECT_ID  \
   --topicID crypt-topic \
   --keyType=TINK \
   --keyUri=gcp-kms://projects/$PROJECT_ID/locations/us-central1/keyRings/pubsub-kr/cryptoKeys/key1 \
   -v 10 -alsologtostderr
```
