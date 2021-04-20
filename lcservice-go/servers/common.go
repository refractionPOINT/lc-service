package servers

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"io"
	"net/http"

	svc "github.com/refractionPOINT/lc-service/lcservice-go/service"
)

func encodeResponse(resp svc.Response, w http.ResponseWriter) {
	if err := json.NewEncoder(w).Encode(resp); err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
}

func handleResponse(resp svc.Response, w http.ResponseWriter) {
	if resp.Error != "" {
		w.WriteHeader(http.StatusBadRequest)
		encodeResponse(resp, w)
		return
	}

	w.WriteHeader(http.StatusOK)
	encodeResponse(resp, w)
}

func process(service Service, w http.ResponseWriter, r *http.Request) {
	// Read all the incoming body.
	b, err := io.ReadAll(r.Body)
	if err != nil {
		w.WriteHeader(http.StatusNoContent)
		return
	}

	// Check the signature.
	sig := r.Header.Get("lc-svc-sig")
	if !verifyOrigin(b, sig, service.GetSecretKey()) {
		w.WriteHeader(http.StatusUnauthorized)
		return
	}

	// Deserialize content.
	d := map[string]interface{}{}
	if err := json.Unmarshal(b, &d); err != nil {
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	requestTypeValue, ok := d["etype"]
	if !ok {
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	if requestTypeValue == "command" {
		resp := service.ProcessCommand(d)
		handleResponse(resp, w)
		return
	}

	// it's not a command, then it's a request
	resp := service.ProcessRequest(d)
	handleResponse(resp, w)
}

func verifyOrigin(data []byte, sig string, secretKey []byte) bool {
	mac := hmac.New(sha256.New, secretKey)
	if _, err := mac.Write(data); err != nil {
		return false
	}
	jsonCompatSig := []byte(hex.EncodeToString(mac.Sum(nil)))
	return hmac.Equal(jsonCompatSig, []byte(sig))
}
