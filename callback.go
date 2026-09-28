package main

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/sha1"
	"crypto/subtle"
	"encoding/base64"
	"encoding/binary"
	"encoding/xml"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"sort"
	"strconv"
	"strings"
	"time"
)

const callbackAddrDefault = "0.0.0.0:28473"
const callbackPath = "/hook_path"

const callbackTimestampSkew = 5 * time.Minute

type callbackXML struct {
	Encrypt string `xml:"Encrypt"`
}

func startCallbackServer() {
	if env("WECOM_CALLBACK_ENABLED", "") != "true" {
		return
	}

	token := os.Getenv("WECOM_CALLBACK_TOKEN")
	aesKey := os.Getenv("WECOM_CALLBACK_AES_KEY")

	if token == "" || aesKey == "" {
		log.Printf("callback disabled: WECOM_CALLBACK_TOKEN / WECOM_CALLBACK_AES_KEY missing")
		return
	}

	addr := env("WECOM_CALLBACK_ADDR", callbackAddrDefault)

	mux := http.NewServeMux()
	mux.HandleFunc(callbackPath, callbackHandler(token, aesKey))

	server := &http.Server{
		Addr:              addr,
		Handler:           mux,
		ReadHeaderTimeout: 5 * time.Second,
		ReadTimeout:       10 * time.Second,
		WriteTimeout:      10 * time.Second,
		IdleTimeout:       30 * time.Second,
		MaxHeaderBytes:    16 * 1024,
	}

	go func() {
		log.Printf("wecom callback listening on %s%s", addr, callbackPath)

		if err := server.ListenAndServe(); err != nil &&
			err != http.ErrServerClosed {
			log.Printf("wecom callback server stopped: %v", err)
		}
	}()
}

func callbackHandler(token, aesKey string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != callbackPath {
			http.NotFound(w, r)
			return
		}

		if r.Method == http.MethodGet {
			q := r.URL.Query()

			msgSignature := q.Get("msg_signature")
			timestamp := q.Get("timestamp")
			nonce := q.Get("nonce")
			echoStr := q.Get("echostr")

			if msgSignature == "" ||
				timestamp == "" ||
				nonce == "" ||
				echoStr == "" {
				http.Error(w, "missing verification parameters", http.StatusBadRequest)
				return
			}

			if !validCallbackTimestamp(timestamp) {
				http.Error(w, "invalid timestamp", http.StatusForbidden)
				return
			}

			if !validCallbackSignature(
				token,
				timestamp,
				nonce,
				echoStr,
				msgSignature,
			) {
				http.Error(w, "invalid signature", http.StatusForbidden)
				return
			}

			plain, err := decryptCallbackEcho(aesKey, echoStr)
			if err != nil {
				http.Error(w, "decrypt failed", http.StatusForbidden)
				return
			}

			w.Header().Set("Content-Type", "text/plain; charset=utf-8")
			_, _ = w.Write([]byte(plain))
			return
		}

		if r.Method != http.MethodPost {
			w.Header().Set("Allow", "GET, POST")
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}

		if r.ContentLength > 128*1024 {
			http.Error(w, "request body too large", http.StatusRequestEntityTooLarge)
			return
		}

		r.Body = http.MaxBytesReader(w, r.Body, 128*1024)
		defer r.Body.Close()

		body, err := io.ReadAll(r.Body)
		if err != nil {
			http.Error(w, "read body failed", http.StatusBadRequest)
			return
		}

		var envelope callbackXML

		if err := xml.Unmarshal(body, &envelope); err != nil ||
			envelope.Encrypt == "" {
			http.Error(w, "invalid callback body", http.StatusBadRequest)
			return
		}

		q := r.URL.Query()
		timestamp := q.Get("timestamp")
		nonce := q.Get("nonce")
		msgSignature := q.Get("msg_signature")

		if !validCallbackTimestamp(timestamp) {
			http.Error(w, "invalid timestamp", http.StatusForbidden)
			return
		}

		if !validCallbackSignature(
			token,
			timestamp,
			nonce,
			envelope.Encrypt,
			msgSignature,
		) {
			http.Error(w, "invalid signature", http.StatusForbidden)
			return
		}

		if _, err := decryptCallbackEcho(aesKey, envelope.Encrypt); err != nil {
			http.Error(w, "decrypt failed", http.StatusForbidden)
			return
		}

		w.Header().Set("Content-Type", "text/plain; charset=utf-8")
		_, _ = w.Write([]byte("success"))
	}
}

func validCallbackTimestamp(timestamp string) bool {
	value, err := strconv.ParseInt(strings.TrimSpace(timestamp), 10, 64)
	if err != nil {
		return false
	}

	now := time.Now().Unix()
	diff := now - value
	if diff < 0 {
		diff = -diff
	}

	return diff <= int64(callbackTimestampSkew/time.Second)
}

func validCallbackSignature(
	token,
	timestamp,
	nonce,
	encrypted,
	expected string,
) bool {
	if token == "" ||
		timestamp == "" ||
		nonce == "" ||
		encrypted == "" ||
		expected == "" {
		return false
	}

	items := []string{
		token,
		timestamp,
		nonce,
		encrypted,
	}

	sort.Strings(items)

	h := sha1.Sum([]byte(strings.Join(items, "")))

	actual := fmt.Sprintf("%x", h[:])
	expected = strings.ToLower(strings.TrimSpace(expected))

	if len(actual) != len(expected) {
		return false
	}

	return subtle.ConstantTimeCompare(
		[]byte(actual),
		[]byte(expected),
	) == 1
}

func decryptCallbackEcho(
	encodingAESKey,
	encrypted string,
) (string, error) {
	key, err := base64.StdEncoding.DecodeString(encodingAESKey + "=")
	if err != nil || len(key) != 32 {
		return "", fmt.Errorf("invalid EncodingAESKey")
	}

	ciphertext, err := base64.StdEncoding.DecodeString(encrypted)
	if err != nil ||
		len(ciphertext) == 0 ||
		len(ciphertext)%aes.BlockSize != 0 {
		return "", fmt.Errorf("invalid encrypted payload")
	}

	block, err := aes.NewCipher(key)
	if err != nil {
		return "", err
	}

	plaintext := make([]byte, len(ciphertext))

	cipher.NewCBCDecrypter(
		block,
		key[:aes.BlockSize],
	).CryptBlocks(plaintext, ciphertext)

	plaintext, err = pkcs7Unpad(plaintext, aes.BlockSize)
	if err != nil || len(plaintext) < 20 {
		return "", fmt.Errorf("invalid decrypted payload")
	}

	msgLen := binary.BigEndian.Uint32(plaintext[16:20])
	end := 20 + int(msgLen)
	if end > len(plaintext) {
		return "", fmt.Errorf("invalid message length")
	}

	return string(plaintext[20:end]), nil
}

func pkcs7Unpad(data []byte, blockSize int) ([]byte, error) {
	if len(data) == 0 ||
		len(data)%blockSize != 0 {
		return nil, fmt.Errorf("invalid padding length")
	}

	n := int(data[len(data)-1])
	if n == 0 || n > blockSize || n > len(data) {
		return nil, fmt.Errorf("invalid padding")
	}

	for _, b := range data[len(data)-n:] {
		if int(b) != n {
			return nil, fmt.Errorf("invalid padding")
		}
	}

	return data[:len(data)-n], nil
}
