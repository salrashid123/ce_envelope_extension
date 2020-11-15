
## Cloud Events envelope encryption extension
     (for Pubsub)


Sample [Cloud Events](https://cloudevents.io/) extension that describes encrypted payloads using TINK Envelope Encryption.

Cloud [Events Extensions](https://github.com/cloudevents/spec/blob/v1.0/documented-extensions.md) basically just describes metadata to include into headers of any given message.

This particular extension first encrypts the `Data` area of a PubSub message using either a Google Cloud KMS-backed DEK or  GCP KMS backed TINK Symmetric key.  Once encrypted, it places the key reference value and the encrypted Data Encryption Key (DEK) as the metadata value for transmission.

The subscriber of the event will decode the extension and then use the encrypted key to finally decode the message.

Since the extension framework is just metadata and not an 'interceptor' for pre/post processing messages, the decryption has to be done in code manually...

This sample sets up a pubsub topic with a subscriber as well as a symmetric GCP KMS key to use for encryption.

On startup, the pubsub publisher will generate 10 messages but only rotate the derived AES DEK key three times.  This is done to avoid repeated calls to kms to rewrap the DEK (meaning, you are using the DEK a couple of times before its rotated out). The DEK and TINK ke gets embedded into the cloud event payload

A sample event log is shown below showing 

- GCP KMS DEK

```log
$ go run main.go --mode publish --projectID $PROJECT_ID -topicID crypt-topic --keyType=KMS --keyUri=projects/mineral-minutia-820/locations/us-central1/keyRings/pubsub-kr/cryptoKeys/key1  -v 20 -alsologtostderr
I1115 12:21:00.960344  185353 main.go:171] Generating New Key
I1115 12:21:01.509476  185353 main.go:186]      DEK sha256 [df6e0a76a06fbdcebaef7f4b130b914445dd70aad00135ef9409167a89ec6489]
I1115 12:21:01.509712  185353 main.go:201]      Encrypting data: [foo 1]
I1115 12:21:01.509856  185353 main.go:214] Validation: valid
Context Attributes,
  specversion: 1.0
  type: com.google.cloud.pubsub.topic.publish
  source: github.com/salrashid123/tink_samples
  id: 6086176d-f680-459d-a477-3c0bbe7265d8
  datacontenttype: application/json
Extensions,
  envelopeencryption: {"key_uri":"projects/mineral-minutia-820/locations/us-central1/keyRings/pubsub-kr/cryptoKeys/key1","dek":"CiQAVjnoXRMQGyrM/+9iU1Hlpq9D5WxuJcwMyRxRxM6Uw3quag0SSQBxDtBCb7TH6YPykqTt+YOGf2ScTk/HjdkxRrvptqJDcYY3Eax/tirKT04T9pJVpkS4f6sZiQQfII7c/yqFnEVFxXgWuWA2jv4=","type":3}
Data,
  {
    "data": "uHE3U2goGOG2Yc+fhYdFchpHgkdMzL4qIRjyyLU6zZIl"
  }
```

- TINK KMS DEK

```log
$  go run main.go --mode publish --projectID $PROJECT_ID -topicID crypt-topic --keyType=TINK --keyUri=gcp-kms://projects/$PROJECT_ID/locations/us-central1/keyRings/pubsub-kr/cryptoKeys/key1 -v 20 -alsologtostderr
I1115 12:19:37.741541  185133 main.go:171] Generating New Key
I1115 12:19:38.226877  185133 main.go:186]      DEK sha256 [d4f6dbeeb9bfbe8121c7de2ce4ab42e4905bb7ea64cd07e430fbf6cc9d5c5108]
I1115 12:19:38.227058  185133 main.go:201]      Encrypting data: [foo 1]
I1115 12:19:38.300952  185133 main.go:214] Validation: valid
Context Attributes,
  specversion: 1.0
  type: com.google.cloud.pubsub.topic.publish
  source: github.com/salrashid123/tink_samples
  id: 93c98348-f7c7-4801-8fa1-6880cdfbfd3f
  datacontenttype: application/json
Extensions,
  envelopeencryption: {"key_uri":"gcp-kms://projects/mineral-minutia-820/locations/us-central1/keyRings/pubsub-kr/cryptoKeys/key1","dek":"{\"encryptedKeyset\":\"AAAAcwokAFY56F2mZshEbSJ9cxAfGm63IHijrXAGGK0QqlT3MToBrX74EksAcQ7QQrb1mE8wSuNCuK4K4fWbi7ehBhBVTfT1jW2ll8NX5DJfL0nqcowsiRCWqz4dnzF03L/78RhZrJts9DpY9trUIIg5W7glD/p/95bhNzQ3Qmr09N3YGR2P3BfZNfERXAj5mHciGbI3f6oSMeZKlrEtXxCDjVY6esh6SXVkIuecpLq1KVap8wp/JKEsFpAk7Xce2xqevAJE+Y5PLe6X85xa5KDSwBjwvQsY8cEdrYfw69U7tjt2JtE6QyZF4wd9YwMY8Nb5RVS0KJTKkAYaIBRTsZTqd8AJoRodwM8DSmjsyg610A0n8XLgPDsw3C2jSg6S2CybO5SJU3vi3RJkeuyKDneqkGOZsPByA/PsfVqvvf068u7DdWGhRNZaqqTH8YR7OB0YWg0zt39FylzLn7LEJFJfsgB/i2UdrOXccTeCSglnYe2JmS3SpffmNxae0gP2bR0N7kQ=\",\"keysetInfo\":{\"primaryKeyId\":26353762,\"keyInfo\":[{\"typeUrl\":\"type.googleapis.com/google.crypto.tink.KmsEnvelopeAeadKey\",\"status\":\"ENABLED\",\"keyId\":26353762,\"outputPrefixType\":\"TINK\"}]}}","type":0}
Data,
  {
    "data": "AQGSIGIAAABzCiQAVjnoXbQTBExFBA5bJYezssyOKYZtmHT2UhVgOEWS5NFSTPkSSwBxDtBCK9ZXkzvyu/34o42lBwcOwAuJVnHh2yO2KaRkYexHRxS9aK7sahQHZOXXq9fKCYLWn3Qp7KzhwHq2hFWNKuYkP0ucuxXmj1fpvn81OxFa2JqH9pstrVRgLHfWZ4jo48/FPHRfShMmsA=="
  }
```

Notice the `Extensions -> envelopencryption` section.  That is the metadata that gets sent with each data. The `encryptedKeyset` is the KMS wrapped key that uses the Key Encryption Key (KEK) defined by the `key_uri`.  The inner, encrypted KEK is used to encrypt the `Data` section.

On the subscriber side, the  KEK gets cached locally to avoid repeated lookups.  If the subscriber does not find the KEK in cache,it will use TINK to do envelope decryption on the Data and then save the KEK.  If the keys are rotated by the publisher, no problem,it will detect that the KEK is not in cache and continue.


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

There are two supported mechanism:  using `KMS` wrapped `TINK` key and plain `KMS envelope encryption`

#### TINK

- `Publisher`

```log
$ go run main.go \
  --mode publish \
  --projectID $PROJECT_ID \
  --topicID crypt-topic \
  --keyType=TINK \
  --keyUri=gcp-kms://projects/$PROJECT_ID/locations/us-central1/keyRings/pubsub-kr/cryptoKeys/key1 \
  -v 10 -alsologtostderr

I1115 12:23:59.331586  185930 main.go:171] Generating New Key
I1115 12:23:59.942903  185930 main.go:186]      DEK sha256 [6a774899529600b07b553d8e658f85f79cb5406e2a265cd52086ea872210fa2b]
I1115 12:23:59.943108  185930 main.go:201]      Encrypting data: [foo 1]
I1115 12:24:01.499816  185930 main.go:181] Using Existing Key <<<<<<<<<<<<<<<<<<<<<
I1115 12:24:01.499919  185930 main.go:186]      DEK sha256 [6a774899529600b07b553d8e658f85f79cb5406e2a265cd52086ea872210fa2b]
I1115 12:24:01.500056  185930 main.go:201]      Encrypting data: [foo 2]
I1115 12:24:02.702313  185930 main.go:181] Using Existing Key
I1115 12:24:02.702429  185930 main.go:186]      DEK sha256 [6a774899529600b07b553d8e658f85f79cb5406e2a265cd52086ea872210fa2b]
I1115 12:24:02.702558  185930 main.go:201]      Encrypting data: [foo 3]
I1115 12:24:03.897841  185930 main.go:171] Generating New Key
I1115 12:24:04.040219  185930 main.go:186]      DEK sha256 [c8e608b2e29b3162e709660958a69b9755cff7e346b0f1fe4bc03e5014685a7c]
I1115 12:24:04.040285  185930 main.go:201]      Encrypting data: [foo 4]
I1115 12:24:05.186298  185930 main.go:181] Using Existing Key
I1115 12:24:05.186409  185930 main.go:186]      DEK sha256 [c8e608b2e29b3162e709660958a69b9755cff7e346b0f1fe4bc03e5014685a7c]
I1115 12:24:05.186547  185930 main.go:201]      Encrypting data: [foo 5]
I1115 12:24:06.531747  185930 main.go:181] Using Existing Key
I1115 12:24:06.531855  185930 main.go:186]      DEK sha256 [c8e608b2e29b3162e709660958a69b9755cff7e346b0f1fe4bc03e5014685a7c]
I1115 12:24:06.531973  185930 main.go:201]      Encrypting data: [foo 6]
I1115 12:24:07.662745  185930 main.go:181] Using Existing Key
I1115 12:24:07.662851  185930 main.go:186]      DEK sha256 [c8e608b2e29b3162e709660958a69b9755cff7e346b0f1fe4bc03e5014685a7c]
I1115 12:24:07.662969  185930 main.go:201]      Encrypting data: [foo 7]
I1115 12:24:08.788511  185930 main.go:171] Generating New Key
I1115 12:24:08.953111  185930 main.go:186]      DEK sha256 [b7bbf1dddd9c77d0199749ca4a263d82b4ae26f276a77812f7b7ad11013f8d5b]
I1115 12:24:08.953240  185930 main.go:201]      Encrypting data: [foo 8]
I1115 12:24:10.103003  185930 main.go:181] Using Existing Key
I1115 12:24:10.103124  185930 main.go:186]      DEK sha256 [b7bbf1dddd9c77d0199749ca4a263d82b4ae26f276a77812f7b7ad11013f8d5b]
I1115 12:24:10.103237  185930 main.go:201]      Encrypting data: [foo 9]
```

- `Subscriber`

```log
$ go run main.go \
  --mode subscribe \
  --projectID $PROJECT_ID \
  --topicID crypt-topic \
  --subID crypt-subscribe \
  --keyType=TINK \
  --keyUri=gcp-kms://projects/$PROJECT_ID/locations/us-central1/keyRings/pubsub-kr/cryptoKeys/key1 \
  -v 10 -alsologtostderr

I1115 12:24:00.554627  185830 main.go:86] Initialize new key 
I1115 12:24:00.879457  185830 main.go:96]      DEK sha256 value [6a774899529600b07b553d8e658f85f79cb5406e2a265cd52086ea872210fa2b]
I1115 12:24:00.950335  185830 main.go:117] Decrypted Pubsub Message data [foo 1]
I1115 12:24:01.736147  185830 main.go:83] Using Existing key  <<<<<<<<<<<<<<<<<<<<<
I1115 12:24:01.736230  185830 main.go:96]      DEK sha256 value [6a774899529600b07b553d8e658f85f79cb5406e2a265cd52086ea872210fa2b]
I1115 12:24:01.803515  185830 main.go:117] Decrypted Pubsub Message data [foo 2]
I1115 12:24:02.935565  185830 main.go:83] Using Existing key 
I1115 12:24:02.935635  185830 main.go:96]      DEK sha256 value [6a774899529600b07b553d8e658f85f79cb5406e2a265cd52086ea872210fa2b]
I1115 12:24:03.006534  185830 main.go:117] Decrypted Pubsub Message data [foo 3]
I1115 12:24:04.222793  185830 main.go:86] Initialize new key 
I1115 12:24:04.373924  185830 main.go:96]      DEK sha256 value [c8e608b2e29b3162e709660958a69b9755cff7e346b0f1fe4bc03e5014685a7c]
I1115 12:24:04.498616  185830 main.go:117] Decrypted Pubsub Message data [foo 4]
I1115 12:24:06.492514  185830 main.go:83] Using Existing key 
I1115 12:24:06.492589  185830 main.go:96]      DEK sha256 value [c8e608b2e29b3162e709660958a69b9755cff7e346b0f1fe4bc03e5014685a7c]
I1115 12:24:06.557298  185830 main.go:117] Decrypted Pubsub Message data [foo 5]
I1115 12:24:06.698512  185830 main.go:83] Using Existing key 
I1115 12:24:06.698586  185830 main.go:96]      DEK sha256 value [c8e608b2e29b3162e709660958a69b9755cff7e346b0f1fe4bc03e5014685a7c]
I1115 12:24:06.779188  185830 main.go:117] Decrypted Pubsub Message data [foo 6]
I1115 12:24:07.821487  185830 main.go:83] Using Existing key 
I1115 12:24:07.821557  185830 main.go:96]      DEK sha256 value [c8e608b2e29b3162e709660958a69b9755cff7e346b0f1fe4bc03e5014685a7c]
I1115 12:24:07.898012  185830 main.go:117] Decrypted Pubsub Message data [foo 7]
I1115 12:24:09.119041  185830 main.go:86] Initialize new key 
I1115 12:24:09.273569  185830 main.go:96]      DEK sha256 value [b7bbf1dddd9c77d0199749ca4a263d82b4ae26f276a77812f7b7ad11013f8d5b]
I1115 12:24:09.396549  185830 main.go:117] Decrypted Pubsub Message data [foo 8]
I1115 12:24:10.260603  185830 main.go:83] Using Existing key 
I1115 12:24:10.260673  185830 main.go:96]      DEK sha256 value [b7bbf1dddd9c77d0199749ca4a263d82b4ae26f276a77812f7b7ad11013f8d5b]
I1115 12:24:10.365790  185830 main.go:117] Decrypted Pubsub Message data [foo 9]
```

## KMS

- `Publisher`

```log
$ go run main.go \
  --mode publish \
  --projectID $PROJECT_ID \
  --topicID crypt-topic \
  --keyType=KMS \
  --keyUri=projects/mineral-minutia-820/locations/us-central1/keyRings/pubsub-kr/cryptoKeys/key1  \
  -v 10 -alsologtostderr

I1115 13:26:24.253332  191295 main.go:171] Generating New Key
I1115 13:26:24.814591  191295 main.go:186]      DEK sha256 [312393a572f0e5207e76528f159cde2cd417b5e00c61f628dedb74de6c78e6ca]
I1115 13:26:24.814805  191295 main.go:201]      Encrypting data: [foo 1]
I1115 13:26:26.335174  191295 main.go:181] Using Existing Key
I1115 13:26:26.335272  191295 main.go:186]      DEK sha256 [312393a572f0e5207e76528f159cde2cd417b5e00c61f628dedb74de6c78e6ca]
I1115 13:26:26.335385  191295 main.go:201]      Encrypting data: [foo 2]
I1115 13:26:27.546467  191295 main.go:181] Using Existing Key
I1115 13:26:27.546569  191295 main.go:186]      DEK sha256 [312393a572f0e5207e76528f159cde2cd417b5e00c61f628dedb74de6c78e6ca]
I1115 13:26:27.546697  191295 main.go:201]      Encrypting data: [foo 3]
I1115 13:26:28.795868  191295 main.go:171] Generating New Key
I1115 13:26:29.348579  191295 main.go:186]      DEK sha256 [6d18b70d0605a2936cf344e47b2ea5dde0d1d960236450eba7f2ea5853a9690a]
I1115 13:26:29.348716  191295 main.go:201]      Encrypting data: [foo 4]
I1115 13:26:30.577106  191295 main.go:181] Using Existing Key
I1115 13:26:30.577130  191295 main.go:186]      DEK sha256 [6d18b70d0605a2936cf344e47b2ea5dde0d1d960236450eba7f2ea5853a9690a]
I1115 13:26:30.577200  191295 main.go:201]      Encrypting data: [foo 5]
I1115 13:26:31.631940  191295 main.go:181] Using Existing Key
I1115 13:26:31.632010  191295 main.go:186]      DEK sha256 [6d18b70d0605a2936cf344e47b2ea5dde0d1d960236450eba7f2ea5853a9690a]
I1115 13:26:31.632126  191295 main.go:201]      Encrypting data: [foo 6]
I1115 13:26:32.686279  191295 main.go:181] Using Existing Key
I1115 13:26:32.686376  191295 main.go:186]      DEK sha256 [6d18b70d0605a2936cf344e47b2ea5dde0d1d960236450eba7f2ea5853a9690a]
I1115 13:26:32.686498  191295 main.go:201]      Encrypting data: [foo 7]
I1115 13:26:33.842644  191295 main.go:171] Generating New Key
I1115 13:26:34.177667  191295 main.go:186]      DEK sha256 [23aea3947cba4c9b751989321977e600133b3c7f328dc8db31f083a20979a283]
I1115 13:26:34.177826  191295 main.go:201]      Encrypting data: [foo 8]
I1115 13:26:35.233451  191295 main.go:181] Using Existing Key
I1115 13:26:35.233522  191295 main.go:186]      DEK sha256 [23aea3947cba4c9b751989321977e600133b3c7f328dc8db31f083a20979a283]
I1115 13:26:35.233653  191295 main.go:201]      Encrypting data: [foo 9]
```

- `Subscriber`

```log
$ go run main.go \
  --mode subscribe \
  --projectID $PROJECT_ID \
  --topicID crypt-topic \
  --subID crypt-subscribe \
  --keyType=KMS \
  --keyUri=projects/mineral-minutia-820/locations/us-central1/keyRings/pubsub-kr/cryptoKeys/key1 \
  -v 10 -alsologtostderr

I1115 13:26:26.347088  191210 main.go:86] Initialize new key 
I1115 13:26:26.649374  191210 main.go:96]      DEK sha256 value [312393a572f0e5207e76528f159cde2cd417b5e00c61f628dedb74de6c78e6ca]
I1115 13:26:26.649552  191210 main.go:117] Decrypted Pubsub Message data [foo 1]
I1115 13:26:27.576954  191210 main.go:83] Using Existing key 
I1115 13:26:27.577085  191210 main.go:96]      DEK sha256 value [312393a572f0e5207e76528f159cde2cd417b5e00c61f628dedb74de6c78e6ca]
I1115 13:26:27.577172  191210 main.go:117] Decrypted Pubsub Message data [foo 2]
I1115 13:26:28.724293  191210 main.go:83] Using Existing key 
I1115 13:26:28.724364  191210 main.go:96]      DEK sha256 value [312393a572f0e5207e76528f159cde2cd417b5e00c61f628dedb74de6c78e6ca]
I1115 13:26:28.724431  191210 main.go:117] Decrypted Pubsub Message data [foo 3]
I1115 13:26:30.560598  191210 main.go:86] Initialize new key 
I1115 13:26:30.669026  191210 main.go:86] Initialize new key 
I1115 13:26:30.915173  191210 main.go:96]      DEK sha256 value [6d18b70d0605a2936cf344e47b2ea5dde0d1d960236450eba7f2ea5853a9690a]
I1115 13:26:30.915276  191210 main.go:117] Decrypted Pubsub Message data [foo 4]
I1115 13:26:31.026387  191210 main.go:96]      DEK sha256 value [6d18b70d0605a2936cf344e47b2ea5dde0d1d960236450eba7f2ea5853a9690a]
I1115 13:26:31.026496  191210 main.go:117] Decrypted Pubsub Message data [foo 5]
I1115 13:26:31.720859  191210 main.go:83] Using Existing key 
I1115 13:26:31.720978  191210 main.go:96]      DEK sha256 value [6d18b70d0605a2936cf344e47b2ea5dde0d1d960236450eba7f2ea5853a9690a]
I1115 13:26:31.721056  191210 main.go:117] Decrypted Pubsub Message data [foo 6]
I1115 13:26:33.058069  191210 main.go:83] Using Existing key 
I1115 13:26:33.058138  191210 main.go:96]      DEK sha256 value [6d18b70d0605a2936cf344e47b2ea5dde0d1d960236450eba7f2ea5853a9690a]
I1115 13:26:33.058201  191210 main.go:117] Decrypted Pubsub Message data [foo 7]
I1115 13:26:34.266465  191210 main.go:86] Initialize new key 
I1115 13:26:34.600962  191210 main.go:96]      DEK sha256 value [23aea3947cba4c9b751989321977e600133b3c7f328dc8db31f083a20979a283]
I1115 13:26:34.601065  191210 main.go:117] Decrypted Pubsub Message data [foo 8]
I1115 13:26:35.322257  191210 main.go:83] Using Existing key 
I1115 13:26:35.322332  191210 main.go:96]      DEK sha256 value [23aea3947cba4c9b751989321977e600133b3c7f328dc8db31f083a20979a283]
I1115 13:26:35.322398  191210 main.go:117] Decrypted Pubsub Message data [foo 9]
```

Unlike using KMS-TINK, using plain KMS DEK will only call KMS api if the key is rotated.

![images/kms_log.png](images/kms_log.png)