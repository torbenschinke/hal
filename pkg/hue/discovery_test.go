package hue

import (
	"fmt"
	"os"
	"testing"
	"time"
)

func TestDiscover(t *testing.T) {
	bridges, err := Discover(time.Second)
	if err != nil {
		t.Fatal(err)
	}

	for _, bridge := range bridges {
		fmt.Printf("%+v\n", bridge)
		cfg, err := bridge.Config()
		if err != nil {
			t.Fatal(err)
		}
		fmt.Printf("%+v\n", cfg)

		auth := os.Getenv("USERNAME")
		if auth == "" {
			key, err := bridge.GenerateClientKey("hal", "1234")
			if err != nil {
				t.Fatal(err)
			}

			fmt.Printf("%+v\n", key)
		}

		bridge.SetAuthentication(auth)
		devices, err := bridge.Devices()
		if err != nil {
			t.Fatal(err)
		}
		fmt.Printf("%+v\n", devices)

		bridge.EventStream().Register(func(event Event) {
			fmt.Printf("%v %s\n", event.CreationTime, event.String())
		})
	}

	time.Sleep(1 * time.Minute)
}
