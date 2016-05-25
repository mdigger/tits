package main

import (
	"fmt"
	"net/rpc"
	"testing"
	"time"
)

func TestConfig(t *testing.T) {
	service := &Config{
		URL:             "https://maxhome-msk.dyndns.org/admin/newapi/",
		Login:           "usrsvc",
		Password:        "9801",
		BraceletProfile: "590890912028694065",
	}
	defer service.Close()

	go func() {
		err := service.Run(":1234")
		if err != nil {
			t.Fatal(err)
		}
	}()
	time.Sleep(time.Second * 3)

	client, err := rpc.DialHTTP("tcp", ":1234")
	if err != nil {
		t.Fatal(err)
	}
	defer client.Close()

	bracelet := User{
		IsUser:    false,
		Login:     "005",
		Password:  "005",
		Ext:       "005",
		CellPhone: "79031111111",
	}

	var id string
	err = client.Call("MXA.Add", bracelet, &id)
	if err != nil {
		t.Error("MXA.Add error:", err)
	} else {
		fmt.Println("MXA.Add:", id)
	}

	bracelet.RecID = id
	bracelet.Login = "006"
	bracelet.Password = "006"
	bracelet.Ext = "006"
	bracelet.CellPhone = "79032222222"

	var ok bool
	err = client.Call("MXA.Update", bracelet, &ok)
	if err != nil {
		t.Error("MXA.Update error:", err)
	} else {
		fmt.Println("MXA.Update:", ok)
	}

	err = client.Call("MXA.Delete", id, &ok)
	if err != nil {
		t.Error("MXA.Delete error:", err)
	} else {
		fmt.Println("MXA.Delete:", ok)
	}
}
