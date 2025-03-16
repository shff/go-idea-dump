package main

import (
	"bytes"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"time"
)

// SignAWSRequest signs an AWS request for Lambda, S3, SES, or any other service
func SignAWSRequest(method, service, region, endpoint, path string, query map[string]string, payload []byte, headers map[string]string, accessKey, secretKey string) (*http.Request, error) {
	// Time setup
	t := time.Now().UTC()
	amzDate := t.Format("20060102T150405Z")
	date := t.Format("20060102")

	// Default headers
	if headers == nil {
		headers = make(map[string]string)
	}
	headers["x-amz-date"] = amzDate
	if service == "s3" && len(payload) == 0 {
		headers["x-amz-content-sha256"] = "UNSIGNED-PAYLOAD"
	} else {
		headers["x-amz-content-sha256"] = sha256Hex(payload)
	}
	headers["host"] = endpoint

	// Build canonical request
	canonicalURI := path
	canonicalQueryString := buildCanonicalQueryString(query)
	canonicalHeaders, signedHeaders := buildCanonicalHeaders(headers)
	payloadHash := headers["x-amz-content-sha256"]
	canonicalRequest := fmt.Sprintf("%s\n%s\n%s\n%s\n%s\n%s",
		method, canonicalURI, canonicalQueryString, canonicalHeaders, signedHeaders, payloadHash,
	)

	// Create string to sign
	algorithm := "AWS4-HMAC-SHA256"
	credentialScope := fmt.Sprintf("%s/%s/%s/aws4_request", date, region, service)
	stringToSign := fmt.Sprintf("%s\n%s\n%s\n%s",
		algorithm, amzDate, credentialScope, sha256Hex([]byte(canonicalRequest)),
	)

	// Calculate signature
	signingKey := getSignatureKey(secretKey, date, region, service)
	signature := hmacHex(signingKey, stringToSign)

	// Add Authorization header
	authorizationHeader := fmt.Sprintf("%s Credential=%s/%s, SignedHeaders=%s, Signature=%s",
		algorithm, accessKey, credentialScope, signedHeaders, signature,
	)
	headers["Authorization"] = authorizationHeader

	// Construct URL
	reqURL := fmt.Sprintf("https://%s%s", endpoint, canonicalURI)
	if canonicalQueryString != "" {
		reqURL += "?" + canonicalQueryString
	}

	// Create HTTP request
	req, err := http.NewRequest(method, reqURL, bytes.NewReader(payload))
	if err != nil {
		return nil, err
	}

	// Add headers to request
	for key, value := range headers {
		req.Header.Set(key, value)
	}

	return req, nil
}

// Helper: Build canonical query string
func buildCanonicalQueryString(query map[string]string) string {
	params := url.Values{}
	for key, value := range query {
		params.Add(key, value)
	}
	return params.Encode()
}

// Helper: Build canonical headers
func buildCanonicalHeaders(headers map[string]string) (string, string) {
	var canonicalHeaders, signedHeaders strings.Builder
	for key, value := range headers {
		key = strings.ToLower(key)
		canonicalHeaders.WriteString(fmt.Sprintf("%s:%s\n", key, value))
		signedHeaders.WriteString(key + ";")
	}
	return canonicalHeaders.String(), strings.TrimSuffix(signedHeaders.String(), ";")
}

// Helper: SHA256 hash
func sha256Hex(data []byte) string {
	hash := sha256.Sum256(data)
	return hex.EncodeToString(hash[:])
}

// Helper: HMAC with SHA256
func hmacSHA256(key []byte, data string) []byte {
	h := hmac.New(sha256.New, key)
	h.Write([]byte(data))
	return h.Sum(nil)
}

// Helper: HMAC as hex string
func hmacHex(key []byte, data string) string {
	return hex.EncodeToString(hmacSHA256(key, data))
}

// Helper: Get signature key
func getSignatureKey(key, date, region, service string) []byte {
	kDate := hmacSHA256([]byte("AWS4"+key), date)
	kRegion := hmacSHA256(kDate, region)
	kService := hmacSHA256(kRegion, service)
	return hmacSHA256(kService, "aws4_request")
}

// s3Req, err := SignAWSRequest(
// 	"GET",
// 	"s3",
// 	"us-east-1",
// 	"my-bucket.s3.amazonaws.com",
// 	"/my-object-key",
// 	map[string]string{"X-Amz-Expires": "3600"},
// 	nil,
// 	map[string]string{},
// 	"AKIA...YOUR_ACCESS_KEY",
// 	"YOUR_SECRET_KEY",
// )
// if err != nil {
// 	log.Fatalf("Error signing S3 request: %v", err)
// }
// fmt.Println("Presigned S3 URL:", s3Req.URL.String())

// lambdaPayload := []byte(`{"key":"value"}`)
// lambdaReq, err := SignAWSRequest(
// 	"POST",
// 	"lambda",
// 	"us-east-1",
// 	"lambda.us-east-1.amazonaws.com",
// 	"/2015-03-31/functions/my-function/invocations",
// 	nil,
// 	lambdaPayload,
// 	map[string]string{"Content-Type": "application/json"},
// 	"AKIA...YOUR_ACCESS_KEY",
// 	"YOUR_SECRET_KEY",
// )
// if err != nil {
// 	log.Fatalf("Error signing Lambda request: %v", err)
// }
// fmt.Println("Signed Lambda Request URL:", lambdaReq.URL.String())

// sesPayload := []byte(
// 	"Action=SendEmail&Source=verified-sender@example.com&Destination.ToAddresses.member.1=recipient@example.com&Message.Subject.Data=Hello&Message.Body.Text.Data=Hello+from+SES",
// )
// sesReq, err := SignAWSRequest(
// 	"POST",
// 	"ses",
// 	"us-east-1",
// 	"email.us-east-1.amazonaws.com",
// 	"/",
// 	nil,
// 	sesPayload,
// 	map[string]string{"Content-Type": "application/x-www-form-urlencoded"},
// 	"AKIA...YOUR_ACCESS_KEY",
// 	"YOUR_SECRET_KEY",
// )
// if err != nil {
// 	log.Fatalf("Error signing SES request: %v", err)
// }
// fmt.Println("Signed SES Request URL:", sesReq.URL.String())
