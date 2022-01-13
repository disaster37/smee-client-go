package main

import (
	"bufio"
	"bytes"
	"fmt"
	"io"
	"net/http"
	"regexp"
	"strings"

	log "github.com/sirupsen/logrus"
)

//SSE name constants
const (
	eName = "event"
	dName = "data"
)

var (
	//ErrNilChan will be returned by Notify if it is passed a nil channel
	ErrNilChan       = fmt.Errorf("nil channel given")
	ErrLostConnexion = fmt.Errorf("we lost connexion")
)

func liveReq(verb, uri string, body io.Reader) (*http.Request, error) {
	req, err := http.NewRequest(verb, uri, body)
	if err != nil {
		return nil, err
	}

	req.Header.Set("Accept", "text/event-stream")

	return req, nil
}

//Event is a go representation of an http server-sent event
type Event struct {
	URI  string
	Type []byte
	Data []byte
	Err  error
}

//Notify takes the uri of an SSE stream and channel, and will send an Event
//down the channel when recieved, until the stream is closed. It will then
//close the stream. This is blocking, and so you will likely want to call this
//in a new goroutine (via `go Notify(..)`)
func Notify(client *http.Client, uri string, evCh chan<- *Event) {
	if evCh == nil {
		panic(ErrNilChan)
	}

	req, err := liveReq("GET", uri, nil)
	if err != nil {
		evCh <- newErr(err)
		return
	}

	res, err := client.Do(req)
	if err != nil {
		evCh <- newErr(err)
		return
	}

	br := bufio.NewReader(res.Body)
	defer res.Body.Close()

	var currEvent *Event

	for {
		bs, err := br.ReadBytes('\n')
		if err != nil {
			if (err != io.EOF) && (err != io.ErrUnexpectedEOF) {
				evCh <- newErr(err)
			} else {
				evCh <- newErr(ErrLostConnexion)
			}
			return
		}

		if len(bs) < 2 {
			log.Debugf("Read byte: Continue because bs < 2: %s", string(bs))
			continue
		}

		event := strings.TrimSuffix(string(bs), "\n")
		log.Debugf("Read bytes: %s", string(event))

		// extract data or event type
		re := regexp.MustCompile("^(\\w+):\\s+(.*)$")
		match := re.FindStringSubmatch(event)
		if len(match) < 3 {
			log.Debugf("Bad event: `%s`", event)
			continue
		}

		currEvent = newData(uri, nil, nil)
		switch match[1] {
		case eName:
			currEvent.Type = bytes.TrimSpace([]byte(match[2]))
			log.Debugf("Found type: %s", currEvent.Type)
			break
		case dName:
			currEvent.Data = bytes.TrimSpace([]byte(match[2]))
			log.Debugf("Found data: %s", currEvent.Data)
			evCh <- currEvent
			break
		}
	}
}

// newData return data event
func newData(URI string, tType []byte, data []byte) *Event {
	return &Event{
		URI:  URI,
		Type: tType,
		Data: data,
		Err:  nil,
	}
}

// newErr return error event
func newErr(err error) *Event {
	return &Event{
		Err: err,
	}
}
