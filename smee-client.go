package main

import (
	"bytes"
	"crypto/hmac"
	"crypto/sha1"
	"crypto/tls"
	"encoding/hex"
	"errors"
	"net/http"
	"strings"
	"time"

	"github.com/buger/jsonparser"
	log "github.com/sirupsen/logrus"
	"github.com/tmaxmax/go-sse"
	cli "github.com/urfave/cli/v2"
)

// ValidMAC reports whether messageMAC is a valid HMAC tag for message.
func ValidMAC(message, messageMAC, key []byte) bool {
	mac := hmac.New(sha1.New, key)
	mac.Write(message)
	expectedMAC := mac.Sum(nil)
	return hmac.Equal(messageMAC, expectedMAC)
}

func hex2bytes(hexstr string) []byte {
	src := []byte(hexstr)
	dst := make([]byte, hex.DecodedLen(len(src)))
	_, err := hex.Decode(dst, src)
	if err != nil {
		log.Fatal(err)
	}
	return dst
}

func startSmee(c *cli.Context) error {

	if c.String("url") == "" {
		return errors.New("--url parameter is required")
	}
	if c.String("target") == "" {
		return errors.New("--target parameter is required")
	}

	clientSmee := &http.Client{
		Timeout: c.Duration("timeout") * time.Second,
	}
	clientBackend := &http.Client{
		Timeout: c.Duration("timeout") * time.Second,
	}

	if c.Bool("self-signed-certificate") {
		clientBackend.Transport = &http.Transport{
			TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
		}
	}

	sseClient := sse.Client{
		MaxRetries:              -1,
		DefaultReconnectionTime: time.Minute * 1,
		HTTPClient:              clientSmee,
	}
	r, err := http.NewRequest(http.MethodGet, c.String("url"), nil)
	if err != nil {
		log.Errorf("Error when create request on URL %s: %s", c.String("url"), err.Error())
		return err
	}
	conn := sseClient.NewConnection(r)

	// Handle messages
	conn.SubscribeMessages(func(ev sse.Event) {
		log.Debugf("Receive message: %s", ev.Data)

		body, _, _, err := jsonparser.Get([]byte(ev.Data), "body")
		if err != nil {
			log.Info("Error: no body found")
			return
		}

		if c.String("secret") != "" {
			signature, _, _, err := jsonparser.Get([]byte(ev.Data), "x-hub-signature")
			if err != nil {
				log.Info("Error: no signature found")
				return
			}
			if string(signature[:5]) != "sha1=" {
				log.Warnf("Skipping checking. signature is not SHA1: %s\n", signature)
				return
			} else {
				if !ValidMAC([]byte(body), hex2bytes(string(signature[5:])), []byte(c.String("secret"))) {
					log.Error("Error: Invalid HMAC\n")
					return
				}
			}
		}

		// Call target to send event
		req, err := http.NewRequest("POST", c.String("target"), bytes.NewBuffer(body))
		if err != nil {
			log.Errorf("Error when create request: %s", err.Error())
			return
		}
		err = jsonparser.ObjectEach([]byte(ev.Data), func(key []byte, value []byte, dataType jsonparser.ValueType, offset int) error {
			if strings.HasPrefix(string(key), "x-") || strings.ToLower(string(key)) == "content-type" {
				req.Header.Set(string(key), string(value))
			}
			return nil
		})
		if err != nil {
			log.Errorf("Error when set header: %s", err.Error())
			return
		}

		_, err = clientBackend.Do(req)
		if err != nil {
			log.Errorf("Error when call target: %s", err.Error())
			return
		}

		log.Infof("Successfully proxied webhook to target: %s", string(body))
	})

	for {
		log.Infof("We proxy '%s' to '%s'", c.String("url"), c.String("target"))
		if err := conn.Connect(); err != nil {
			log.Errorf("Error when access on URL %s: %s", c.String("url"), err.Error())
			time.Sleep(c.Duration("timeout"))
		}
	}

}
