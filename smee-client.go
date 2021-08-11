package main

import (
	"bytes"
	"crypto/hmac"
	"crypto/sha1"
	"encoding/hex"
	"errors"
	"io/ioutil"
	"net/http"
	"strings"

	"github.com/buger/jsonparser"
	log "github.com/sirupsen/logrus"
	"github.com/urfave/cli/v2"
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

	evCh := make(chan *Event)
	go Notify(c.String("url"), evCh)

	for ev := range evCh {
		if len(ev.Data) <= 2 {
			continue
		}

		body, _, _, err := jsonparser.Get(ev.Data, "body")
		if err != nil {
			log.Info("Error: no body found")
			continue
		}

		if c.String("secret") != "" {
			signature, _, _, err := jsonparser.Get(ev.Data, "x-hub-signature")
			if err != nil {
				log.Info("Error: no signature found")
				continue
			}
			if string(signature[:5]) != "sha1=" {
				log.Warnf("Skipping checking. signature is not SHA1: %s\n", signature)
				continue
			} else {
				if !ValidMAC([]byte(body), hex2bytes(string(signature[5:])), []byte(c.String("secret"))) {
					log.Error("Error: Invalid HMAC\n")
					continue
				}
			}
		}
		log.Debugf("Received %s", string(ev.Data))

		req, err := http.NewRequest("POST", c.String("target"), bytes.NewBuffer(body))
		jsonparser.ObjectEach(ev.Data, func(key []byte, value []byte, dataType jsonparser.ValueType, offset int) error {
			if strings.HasPrefix(string(key), "x-") || strings.ToLower(string(key)) == "content-type" {
				req.Header.Set(string(key), string(value))
			}
			return nil
		})

		client := &http.Client{}
		resp, err := client.Do(req)
		if err != nil {
			log.Errorf("Error when call target: %s", err.Error())
			continue
		}
		defer resp.Body.Close()

		log.Debugf("response Status: %s", resp.Status)
		log.Debugf("response Headers: %s", resp.Header)
		rspbody, _ := ioutil.ReadAll(resp.Body)
		log.Debugf("response Body: %s", string(rspbody))

		log.Info("Successfully proxied webhook to target")
	}

	return nil
}
