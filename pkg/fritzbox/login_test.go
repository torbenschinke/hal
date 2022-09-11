package fritzbox

import (
	"fmt"
	"os"
	"testing"
)

func TestCalc(t *testing.T) {
	resp := calcResponse("1234567z", "äbc")
	fmt.Println(resp)

	resp = calcResponse("4640f673", "test123")
	fmt.Println(resp)
}

func TestLogin(t *testing.T) {
	service := NewService()

	id, err := service.SessionID(os.Getenv("FB_USER"), os.Getenv("FB_PWD"))
	if err != nil {
		t.Fatal(err)
	}

	fmt.Println(id)

	service.SID = id

	devices, err := service.WLANDevices()
	if err != nil {
		t.Fatal(err)
	}

	for _, device := range devices {
		if device.Type == "active" {
			fmt.Println(device)
		}
	}

}
