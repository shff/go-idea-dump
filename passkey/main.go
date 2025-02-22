package main

import (
	"crypto"
	"crypto/ecdsa"
	"crypto/rand"
	"crypto/sha256"
	"crypto/x509"
	"database/sql"
	"encoding/asn1"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"math/big"
	"net/http"
	"strings"

	_ "github.com/mattn/go-sqlite3"
)

var db *sql.DB

func main() {
	// Connect to SQLite
	var err error
	db, err = sql.Open("sqlite3", "./webauthn.db")
	if err != nil {
		log.Fatal(err)
	}

	// Create the users table
	_, err = db.Exec(`CREATE TABLE IF NOT EXISTS users (UserID TEXT PRIMARY KEY, CredentialID TEXT, PublicKey TEXT);`)
	if err != nil {
		log.Fatal(err)
	}
	_, err = db.Exec(`CREATE TABLE IF NOT EXISTS challenges (UserID TEXT, Challenge TEXT);`)
	if err != nil {
		log.Fatal(err)
	}

	http.HandleFunc("/webauthn/register-begin", registerBeginHandler)
	http.HandleFunc("/webauthn/register-finish", registerFinishHandler)
	http.HandleFunc("/webauthn/authenticate-begin", authenticateBeginHandler)
	http.HandleFunc("/webauthn/authenticate-finish", authenticateFinishHandler)
	http.Handle("/", http.FileServer(http.Dir("./static")))

	fmt.Println("Server running on http://localhost:8080")
	http.ListenAndServe(":8080", nil)
}

// Private

type RegisterFormData struct {
	ID       string `json:"id"`
	Type     string `json:"type"`
	Response struct {
		ClientDataJSON    string `json:"clientDataJSON"`
		AttestationObject string `json:"attestationObject"`
		AuthenticatorData string `json:"authenticatorData"`
		PublicKey         string `json:"publicKey"`
	} `json:"response"`
}

func registerBeginHandler(w http.ResponseWriter, r *http.Request) {
	// Extract userID from query params
	userID := r.URL.Query().Get("userID")
	if userID == "" {
		http.Error(w, "userID is required", http.StatusBadRequest)
		return
	}

	// Generate a challenge
	challenge := generateChallenge()

	// Delete existing challenges for this user
	if _, err := db.Exec(`DELETE FROM challenges WHERE UserID = ?;`, userID); err != nil {
		http.Error(w, "Failed to delete existing challenges", http.StatusInternalServerError)
		return
	}

	// Store the challenge
	if _, err := db.Exec(`INSERT INTO challenges (UserID, Challenge) VALUES (?, ?);`, userID, challenge); err != nil {
		http.Error(w, "Failed to store challenge", http.StatusInternalServerError)
		return
	}

	// Send response
	writeJSON(w, map[string]interface{}{
		"challenge":       challenge,
		"rpName":          "myapp",
		"userId":          base64.RawURLEncoding.EncodeToString([]byte(userID)),
		"userName":        userID,
		"userDisplayName": "User " + userID,
	}, http.StatusOK)
}

func registerFinishHandler(w http.ResponseWriter, r *http.Request) {
	// Extract userID from query params
	userID := r.URL.Query().Get("userID")
	if userID == "" {
		http.Error(w, "userID is required", http.StatusBadRequest)
		return
	}

	// Parse the request JSON
	var req RegisterFormData
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request", http.StatusBadRequest)
		return
	}

	// Decode the Base64 ClientDataJSON field
	clientDataJSON, err := base64.RawURLEncoding.DecodeString(req.Response.ClientDataJSON)
	if err != nil {
		http.Error(w, "Failed to decode client data JSON", http.StatusBadRequest)
		return
	}

	// Parse the ClientDataJSON field
	var clientData map[string]interface{}
	if err := json.Unmarshal([]byte(clientDataJSON), &clientData); err != nil {
		http.Error(w, "Failed to parse ClientDataJSON", http.StatusBadRequest)
		return
	}

	// Base64-decode the client challenge and convert to string
	clientChallengeBytes, err := base64.RawURLEncoding.DecodeString(clientData["challenge"].(string))
	if err != nil {
		http.Error(w, "Failed to decode client challenge", http.StatusBadRequest)
		return
	}

	// Fetch the saved challenge
	var challenge string
	if err := db.QueryRow(`SELECT Challenge FROM challenges WHERE UserID = ?;`, userID).Scan(&challenge); err != nil {
		http.Error(w, "Challenge not found", http.StatusNotFound)
		return
	}

	// Validate the challenge
	if challenge != string(clientChallengeBytes) {
		http.Error(w, "Invalid challenge", http.StatusBadRequest)
		return
	}

	// Delete previous credentials for this user
	db.Exec(`DELETE FROM users WHERE UserID = ?;`, userID)

	// Store ID and Public Key in the database
	_, err = db.Exec(`INSERT INTO users (UserID, CredentialID, PublicKey) VALUES (?, ?, ?);`, userID, req.ID, req.Response.PublicKey)
	if err != nil {
		http.Error(w, "Failed to store user", http.StatusInternalServerError)
		return
	}

	writeJSON(w, map[string]string{"message": "Registration successful"}, http.StatusOK)
}

type LoginFormData struct {
	ID       string `json:"id"`
	Type     string `json:"type"`
	Response struct {
		ClientDataJSON    string `json:"clientDataJSON"`
		AuthenticatorData string `json:"authenticatorData"`
		Signature         string `json:"signature"`
		UserHandle        string `json:"userHandle"`
	} `json:"response"`
}

func authenticateBeginHandler(w http.ResponseWriter, r *http.Request) {
	// Extract userID from query params
	userID := r.URL.Query().Get("userID")
	if userID == "" {
		http.Error(w, "userID is required", http.StatusBadRequest)
		return
	}

	// Generate challenge
	challenge := generateChallenge()

	// Delete existing challenges for this user
	db.Exec(`DELETE FROM challenges WHERE UserID = ?;`, userID)

	// Store challenge
	_, err := db.Exec(`INSERT INTO challenges (UserID, Challenge) VALUES (?, ?);`, userID, challenge)
	if err != nil {
		http.Error(w, "Failed to store challenge", http.StatusInternalServerError)
		return
	}

	// Send response containing challenge and Credential ID
	writeJSON(w, map[string]interface{}{
		"challenge": challenge,
		"rpName":    "myapp",
	}, http.StatusOK)
}

func authenticateFinishHandler(w http.ResponseWriter, r *http.Request) {
	// Extract userID from query params
	userID := r.URL.Query().Get("userID")
	if userID == "" {
		http.Error(w, "userID is required", http.StatusBadRequest)
		return
	}

	// Parse the request JSON
	var req LoginFormData
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request", http.StatusBadRequest)
		return
	}

	// Challenge Verification

	// Decode the Base64 ClientDataJSON field
	clientDataJSON, err := base64.RawURLEncoding.DecodeString(req.Response.ClientDataJSON)
	if err != nil {
		http.Error(w, "Failed to decode client data JSON", http.StatusBadRequest)
		return
	}

	// Parse the ClientDataJSON field
	var clientData map[string]interface{}
	if err := json.Unmarshal([]byte(clientDataJSON), &clientData); err != nil {
		http.Error(w, "Failed to parse ClientDataJSON", http.StatusBadRequest)
		return
	}

	// Base64-decode the client challenge and convert to string
	clientChallengeBytes, err := base64.RawURLEncoding.DecodeString(clientData["challenge"].(string))
	if err != nil {
		http.Error(w, "Failed to decode client challenge", http.StatusBadRequest)
		return
	}

	// Fetch the saved challenge
	var challenge string
	if err := db.QueryRow(`SELECT Challenge FROM challenges WHERE UserID = ?;`, userID).Scan(&challenge); err != nil {
		http.Error(w, "Challenge not found", http.StatusNotFound)
		return
	}

	// Validate the challenge
	if challenge != string(clientChallengeBytes) {
		http.Error(w, "Invalid challenge", http.StatusBadRequest)
		return
	}

	// Signature Verification

	finalUserID := req.Response.UserHandle

	// Fetch the user Public Key
	var publicKeyBase64 string
	if err := db.QueryRow(`SELECT PublicKey FROM users WHERE UserID = ?;`, finalUserID).Scan(&publicKeyBase64); err != nil {
		http.Error(w, "User not found", http.StatusNotFound)
		return
	}

	// Trim trailing padding from the base64 public key
	publicKeyBase64 = strings.TrimRight(publicKeyBase64, "=")

	// Extract bytes from the base64 public key
	publicKeyBytes, err := base64.RawStdEncoding.DecodeString(publicKeyBase64)
	if err != nil {
		http.Error(w, "Failed to decode public key: "+err.Error(), http.StatusBadRequest)
		return
	}

	// Convert to ECDSA public key (assuming ECDSA for example)
	publicKey, err := x509.ParsePKIXPublicKey(publicKeyBytes)
	if err != nil {
		http.Error(w, "Failed to parse public key", http.StatusBadRequest)
		return
	}

	ecdsaPublicKey, ok := publicKey.(*ecdsa.PublicKey)
	if !ok {
		http.Error(w, "Not an ECDSA public key", http.StatusBadRequest)
		return
	}

	// Decode the base64 authenticator data
	authenticatorData, err := decodeBase64(req.Response.AuthenticatorData)
	if err != nil {
		http.Error(w, "Failed to decode authenticator data", http.StatusBadRequest)
		return
	}

	// Create the data to verify signature
	clientDataHash := sha256.Sum256(clientDataJSON)
	dataToVerify := append(authenticatorData, clientDataHash[:]...)

	// Convert to Std Encoding
	unpaddedSignature := strings.ReplaceAll(req.Response.Signature, "-", "+")
	unpaddedSignature = strings.ReplaceAll(unpaddedSignature, "_", "/")

	// Base64 decode the signature
	signature, err := base64.StdEncoding.DecodeString(unpaddedSignature)
	if err != nil {
		http.Error(w, "Failed to decode signature: "+err.Error()+" - "+unpaddedSignature, http.StatusBadRequest)
		return
	}

	// Get R and S from the Signature
	var sig struct {
		R, S *big.Int
	}
	_, err = asn1.Unmarshal(signature, &sig)
	if err != nil {
		http.Error(w, "Failed to unmarshal signature", http.StatusBadRequest)
		return
	}

	// Verify if the signature matches the public key and hash
	hash := sha256.Sum256(dataToVerify)
	valid := ecdsa.Verify(ecdsaPublicKey, hash[:], sig.R, sig.S)
	if !valid {
		http.Error(w, "Invalid signature", http.StatusBadRequest)
		return
	}

	writeJSON(w, map[string]string{"message": "Authentication successful"}, http.StatusOK)
}

// Private

var algorithms = map[COSEAlgorithmIdentifier]struct {
	name   string
	hash   crypto.Hash
	sigAlg x509.SignatureAlgorithm
}{
	AlgRS1:   {"SHA1-RSA", crypto.SHA1, x509.SHA1WithRSA},
	AlgRS256: {"SHA256-RSA", crypto.SHA256, x509.SHA256WithRSA},
	AlgRS384: {"SHA384-RSA", crypto.SHA384, x509.SHA384WithRSA},
	AlgRS512: {"SHA512-RSA", crypto.SHA512, x509.SHA512WithRSA},
	AlgPS256: {"SHA256-RSAPSS", crypto.SHA256, x509.SHA256WithRSAPSS},
	AlgPS384: {"SHA384-RSAPSS", crypto.SHA384, x509.SHA384WithRSAPSS},
	AlgPS512: {"SHA512-RSAPSS", crypto.SHA512, x509.SHA512WithRSAPSS},
	AlgES256: {"ECDSA-SHA256", crypto.SHA256, x509.ECDSAWithSHA256},
	AlgES384: {"ECDSA-SHA384", crypto.SHA384, x509.ECDSAWithSHA384},
	AlgES512: {"ECDSA-SHA512", crypto.SHA512, x509.ECDSAWithSHA512},
	AlgEdDSA: {"EdDSA", crypto.SHA512, x509.PureEd25519},
}

type COSEAlgorithmIdentifier int

const (
	AlgES256  COSEAlgorithmIdentifier = -7     // AlgES256 ECDSA with SHA-256.
	AlgEdDSA  COSEAlgorithmIdentifier = -8     // AlgEdDSA EdDSA.
	AlgES384  COSEAlgorithmIdentifier = -35    // AlgES384 ECDSA with SHA-384.
	AlgES512  COSEAlgorithmIdentifier = -36    // AlgES512 ECDSA with SHA-512.
	AlgPS256  COSEAlgorithmIdentifier = -37    // AlgPS256 RSASSA-PSS with SHA-256.
	AlgPS384  COSEAlgorithmIdentifier = -38    // AlgPS384 RSASSA-PSS with SHA-384.
	AlgPS512  COSEAlgorithmIdentifier = -39    // AlgPS512 RSASSA-PSS with SHA-512.
	AlgES256K COSEAlgorithmIdentifier = -47    // AlgES256K is ECDSA using secp256k1 curve and SHA-256.
	AlgRS256  COSEAlgorithmIdentifier = -257   // AlgRS256 RSASSA-PKCS1-v1_5 with SHA-256.
	AlgRS384  COSEAlgorithmIdentifier = -258   // AlgRS384 RSASSA-PKCS1-v1_5 with SHA-384.
	AlgRS512  COSEAlgorithmIdentifier = -259   // AlgRS512 RSASSA-PKCS1-v1_5 with SHA-512.
	AlgRS1    COSEAlgorithmIdentifier = -65535 // AlgRS1 RSASSA-PKCS1-v1_5 with SHA-1.
)

// Helper to send JSON responses
func writeJSON(w http.ResponseWriter, data interface{}, statusCode int) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)
	json.NewEncoder(w).Encode(data)
}

// Helper to decode base64 input
func decodeBase64(input string) ([]byte, error) {
	input = strings.ReplaceAll(input, "-", "+")
	input = strings.ReplaceAll(input, "_", "/")

	// Add missing padding if necessary
	if len(input)%4 != 0 {
		input += strings.Repeat("=", 4-(len(input)%4))
	}

	// Decode the Base64-URL string
	data, err := base64.StdEncoding.DecodeString(input)
	if err != nil {
		return nil, errors.New("invalid Base64-URL input")
	}

	return data, nil
}

func generateChallenge() string {
	challenge := make([]byte, 32)
	_, _ = rand.Read(challenge)
	return base64.RawURLEncoding.EncodeToString(challenge)
}
