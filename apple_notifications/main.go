package main

import (
	"bytes"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"net/http"
)

const apnsURL = "https://api.sandbox.push.apple.com/3/device/" // Use production URL for live

type Aps struct {
	Alert string `json:"alert"`
}

type Payload struct {
	Aps Aps `json:"aps"`
}

func sendAPNS(deviceToken, authToken, bundleID string) error {
	payload := Payload{}
	payload.Aps.Alert = "Hello from Go!"

	data, err := json.Marshal(payload)
	if err != nil {
		return err
	}

	req, err := http.NewRequest("POST", apnsURL+deviceToken, bytes.NewBuffer(data))
	if err != nil {
		return err
	}
	req.Header.Set("apns-topic", bundleID)
	req.Header.Set("Authorization", "bearer "+authToken)
	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{
		Transport: &http.Transport{
			TLSClientConfig: &tls.Config{},
		},
	}

	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("APNs error: %s", resp.Status)
	}

	return nil
}

func main() {
	deviceToken := "YOUR_DEVICE_TOKEN"
	authToken := "YOUR_AUTH_TOKEN"
	bundleID := "com.example.app"
	err := sendAPNS(deviceToken, authToken, bundleID)
	if err != nil {
		fmt.Println("Error:", err)
	} else {
		fmt.Println("Notification sent!")
	}
}
