package main

import (
	"bytes"
	"crypto/hmac"
	"crypto/sha1"
	"encoding/hex"
	"errors"
	"fmt"
	"net/http"
	"strings"
	"time"

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

	client := &http.Client{
		Timeout: c.Duration("timeout") * time.Second,
	}

	// Check we can access on URL
	_, err := client.Get(c.String("url"))
	if err != nil {
		return fmt.Errorf("Error when access on URL %s: %s", c.String("url"), err.Error())
	}

	// Check we can access on backend
	// Maybee we need to wait some time before backend start
	timeout := c.Duration("timeout") * time.Second
	wait := time.After(timeout)
	loop := true
	for loop {
		select {
		case <-wait:
			return fmt.Errorf("We can't access on target during %d seconds", timeout)
		default:
			_, err = client.Get(c.String("target"))
			if err != nil {
				log.Warnf("Error when access on target %s: %s", c.String("target"), err.Error())
				time.Sleep(1 * time.Second)
			} else {
				loop = false
			}
		}
	}

	// Disable timeout for live stream
	client.Timeout = 0
	evCh := make(chan *Event)
	go Notify(client, c.String("url"), evCh)

	log.Infof("We proxy '%s' to '%s'", c.String("url"), c.String("target"))

	for ev := range evCh {

		// Handle error event
		if ev.Err != nil {
			switch ev.Err {
			case ErrNilChan:
				log.Errorf("You need to provide chan")
				return ev.Err
			case ErrLostConnexion:
				log.Warnf("We lost connexion on %s, we try to reconnect on it", c.String("url"))
				time.Sleep(1 * time.Millisecond)
				go Notify(client, c.String("url"), evCh)
				break
			default:
				log.Errorf("Error appear: %s", ev.Err.Error())
				time.Sleep(1 * time.Millisecond)
				go Notify(client, c.String("url"), evCh)
			}
		}

		// Handle data event
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

		// Call target to send event
		req, err := http.NewRequest("POST", c.String("target"), bytes.NewBuffer(body))
		jsonparser.ObjectEach(ev.Data, func(key []byte, value []byte, dataType jsonparser.ValueType, offset int) error {
			if strings.HasPrefix(string(key), "x-") || strings.ToLower(string(key)) == "content-type" {
				req.Header.Set(string(key), string(value))
			}
			return nil
		})

		client := &http.Client{}
		_, err = client.Do(req)
		if err != nil {
			log.Errorf("Error when call target: %s", err.Error())
			continue
		}

		log.Infof("Successfully proxied webhook to target: %s", string(body))
	}

	return nil
}
