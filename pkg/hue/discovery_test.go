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
		fmt.Printf("%+v\n", devices)
	}
}
