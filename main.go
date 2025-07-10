package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
)

// SOAPEnvelope is a minimal structure for wrapping parameters into a SOAP envelope.
func SOAPEnvelope(action string, payload map[string]interface{}) string {
	body, _ := json.Marshal(payload) // naive conversion; adapt as necessary
	return fmt.Sprintf(`<?xml version="1.0"?>
<soap:Envelope xmlns:soap="http://schemas.xmlsoap.org/soap/envelope/">
  <soap:Body>
    <%s>%s</%s>
  </soap:Body>
</soap:Envelope>`, action, body, action)
}

// forwardHandler accepts JSON and forwards it to a SOAP endpoint.
func forwardHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost && r.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var payload map[string]interface{}
	if r.Body != nil {
		defer r.Body.Close()
		if err := json.NewDecoder(r.Body).Decode(&payload); err != nil && err != io.EOF {
			http.Error(w, "invalid json", http.StatusBadRequest)
			return
		}
	}

	action := r.URL.Query().Get("action")
	if action == "" {
		http.Error(w, "missing action", http.StatusBadRequest)
		return
	}

	soapXML := SOAPEnvelope(action, payload)

	ykURL := os.Getenv("YK_ENDPOINT")
	if ykURL == "" {
		http.Error(w, "yurtici endpoint not configured", http.StatusInternalServerError)
		return
	}

	resp, err := http.Post(ykURL, "text/xml", bytes.NewBufferString(soapXML))
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadGateway)
		return
	}
	defer resp.Body.Close()

	w.Header().Set("Content-Type", "text/xml")
	w.WriteHeader(resp.StatusCode)
	io.Copy(w, resp.Body)
}

func main() {
	http.HandleFunc("/api/forward", forwardHandler)
	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}
	fmt.Printf("entegra listening on :%s\n", port)
	http.ListenAndServe(":"+port, nil)
}
