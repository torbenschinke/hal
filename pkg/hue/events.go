package hue

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/r3labs/sse/v2"
	"gopkg.in/cenkalti/backoff.v1"
	"log"
	"net/url"
	"sync"
	"time"
)

type EventStream struct {
	bridge   *Bridge
	cancel   func()
	handlers map[int]func(Event)
	lastHnd  int
	mutex    sync.Mutex
}

func newEventStream(bridge *Bridge) *EventStream {
	ctx, cancel := context.WithCancel(context.Background())
	e := &EventStream{bridge: bridge, cancel: cancel, handlers: map[int]func(Event){}}
	base := fmt.Sprintf("https://%s:%d", bridge.Addresses[0], bridge.Port)
	uri, err := url.JoinPath(base, "eventstream/clip/v2")
	if err != nil {
		panic(fmt.Errorf("cannot happen: %w", err))
	}

	httpClient := newClient(bridge.ID)
	client := sse.NewClient(uri)
	client.Connection = httpClient
	client.Headers["hue-application-key"] = bridge.auth.username
	client.ReconnectStrategy = backoff.NewConstantBackOff(time.Second * 10)

	go func() {
		if err := client.SubscribeWithContext(ctx, "", e.onEvent); err != nil {
			log.Println(err)
		}
	}()

	return e
}

func (e *EventStream) Close() error {
	e.cancel()
	return nil
}

func (e *EventStream) onEvent(msg *sse.Event) {
	//fmt.Println(string(msg.ID), string(msg.Event), string(msg.Data), string(msg.Retry), string(msg.Comment))
	e.mutex.Lock()
	e.mutex.Unlock()

	if len(msg.Data) != 0 {
		var events []Event
		if err := json.Unmarshal(msg.Data, &events); err != nil {
			log.Println(err)
			return
		}

		for _, f := range e.handlers {
			for _, event := range events {
				f(event)
			}
		}
	}
}

func (e *EventStream) Register(f func(event Event)) int {
	e.mutex.Lock()
	e.mutex.Unlock()

	e.lastHnd++
	e.handlers[e.lastHnd] = f

	return e.lastHnd
}

type Event struct {
	ID           string            `json:"id"`           // e.g. 55fa0608-926b-440d-9e5a-081d8f27445b
	Type         string            `json:"type"`         // e.g. update
	CreationTime time.Time         `json:"creationtime"` // e.g. 2022-09-04T10:42:44Z
	Data         []json.RawMessage `json:"data"`         // depends on the resource type e.g. LightGet see https://developers.meethue.com/develop/hue-api-v2/api-reference/#resource_light_get
}

func (e Event) String() string {
	t := ""
	for _, d := range e.Data {
		t += string(d)
	}

	return t
}
