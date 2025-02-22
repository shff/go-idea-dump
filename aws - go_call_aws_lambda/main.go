package main

import (
	"bytes"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"net/http"
	"time"
)

// AWS Credentials and Lambda Config
const (
	AccessKey    = "your-access-key"
	SecretKey    = "your-secret-key"
	Region       = "us-east-1"
	FunctionName = "your-lambda-function-name"
)

func sign(key []byte, message string) []byte {
	mac := hmac.New(sha256.New, key)
	mac.Write([]byte(message))
	return mac.Sum(nil)
}

func getSignatureKey(secret, date, region, service string) []byte {
	kDate := sign([]byte("AWS4"+secret), date)
	kRegion := sign(kDate, region)
	kService := sign(kRegion, service)
	kSigning := sign(kService, "aws4_request")
	return kSigning
}

func invokeLambda(payload []byte) ([]byte, error) {
	// Create the request
	url := fmt.Sprintf("https://lambda.%s.amazonaws.com/2015-03-31/functions/%s/invocations", Region, FunctionName)
	req, err := http.NewRequest("POST", url, bytes.NewReader(payload))
	if err != nil {
		return nil, err
	}

	// Add headers
	t := time.Now().UTC()
	amzDate := t.Format("20060102T150405Z")
	dateStamp := t.Format("20060102")
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Amz-Date", amzDate)

	// Create canonical request
	canonicalHeaders := fmt.Sprintf("content-type:application/json\nx-amz-date:%s\n", amzDate)
	signedHeaders := "content-type;x-amz-date"
	payloadHash := fmt.Sprintf("%x", sha256.Sum256(payload))
	canonicalRequest := fmt.Sprintf("POST\n/2015-03-31/functions/%s/invocations\n\n%s\n%s\n%s", FunctionName, canonicalHeaders, signedHeaders, payloadHash)

	// Create string to sign
	credentialScope := fmt.Sprintf("%s/%s/lambda/aws4_request", dateStamp, Region)
	stringToSign := fmt.Sprintf("AWS4-HMAC-SHA256\n%s\n%s\n%x", amzDate, credentialScope, sha256.Sum256([]byte(canonicalRequest)))

	// Calculate signature
	signingKey := getSignatureKey(SecretKey, dateStamp, Region, "lambda")
	signature := hex.EncodeToString(sign(signingKey, stringToSign))

	// Add authorization header
	authorizationHeader := fmt.Sprintf(
		"AWS4-HMAC-SHA256 Credential=%s/%s, SignedHeaders=%s, Signature=%s",
		AccessKey, credentialScope, signedHeaders, signature,
	)
	req.Header.Set("Authorization", authorizationHeader)

	// Send the request
	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	// Read and return response
	return io.ReadAll(resp.Body)
}

func main() {
	payload := []byte(`{"key": "value"}`) // Replace with your payload
	response, err := invokeLambda(payload)
	if err != nil {
		panic(err)
	}
	fmt.Println("Lambda Response:", string(response))
}
