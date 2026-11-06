package main

import (
	"encoding/json"
	"fmt"
	"os"
)

type Response struct {
	OK    bool   `json:"ok"`
	Error string `json:"error,omitempty"`
	Data  map[string]interface{}
}

func outputJSON(data map[string]interface{}) {
	if data == nil {
		data = make(map[string]interface{})
	}
	if _, exists := data["ok"]; !exists {
		data["ok"] = true
	}
	bytes, err := json.MarshalIndent(data, "", "  ")
	if err != nil {
		fmt.Printf("{\"ok\": false, \"error\": \"Failed to marshal JSON: %s\"}\n", err.Error())
		return
	}
	fmt.Println(string(bytes))
}

func outputError(msg string) {
	resp := map[string]interface{}{
		"ok":    false,
		"error": msg,
	}
	bytes, _ := json.MarshalIndent(resp, "", "  ")
	fmt.Println(string(bytes))
	os.Exit(1)
}

func outputErrorWithFallback(msg, fallback, hint string) {
	resp := map[string]interface{}{
		"ok":       false,
		"error":    msg,
		"fallback": fallback,
		"hint":     hint,
	}
	bytes, _ := json.MarshalIndent(resp, "", "  ")
	fmt.Println(string(bytes))
	os.Exit(0)
}
