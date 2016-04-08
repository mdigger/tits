package main

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/rpc"
	"os"
	"strings"
	"testing"
	"time"
)

func TestConfig(t *testing.T) {
	service := &Config{
		MongoDB: "mongodb://localhost/testtits",
		Ublox: &Ublox{
			Token: os.Getenv("Ublox"),
			Servers: []string{
				"http://online-live1.services.u-blox.com/GetOnlineData.ashx",
				"http://online-live2.services.u-blox.com/GetOnlineData.ashx",
			},
			Timeout:     time.Minute * 2,
			CacheTime:   time.Minute * 2,
			MaxDistance: 10000.0,
			Pacc:        100000,
		},
		LBS: &LBS{
			Type:  "Google",
			Token: os.Getenv("Google"),
		},
		POI:     &POI{},
		Devices: &Devices{},
		Store: &Store{
			CacheTime: time.Minute * 2,
		},
	}

	// data2, err := json.MarshalIndent(service, "", "    ")
	// if err != nil {
	// 	t.Fatal(err)
	// }
	// fmt.Println(string(data2))

	defer service.Close()
	go func() {
		err := service.Run(":1234")
		if err != nil {
			t.Fatal(err)
		}
	}()
	time.Sleep(time.Second)

	client, err := rpc.DialHTTP("tcp", ":1234")
	if err != nil {
		t.Fatal(err)
	}
	defer client.Close()

	// UBLOX

	ubloxRequest := UbloxRequest{
		Point: NewPoint(38.67451, 55.715084),
		Profile: UbloxProfile{
			Datatype:    []string{"pos", "eph", "aux"},
			Format:      "aid",
			GNSS:        []string{"gps"},
			FilterOnPos: true,
		},
	}
	var ubloxOut []byte
	err = client.Call("Ublox.Get", ubloxRequest, &ubloxOut)
	if err != nil {
		t.Error("UBLOX error:", err)
	} else {
		fmt.Println("UBLOX:", len(ubloxOut))
	}

	// LBS

	type CellTower struct {
		MobileCountryCode uint16 // The mobile country code.
		MobileNetworkCode uint16 // The mobile network code.
		LocationAreaCode  uint16 // The location area code for GSM and WCDMA networks. The tracking area code for LTE networks.
		CellId            uint32 // The cell id or cell identity.
		SignalStrength    int16  // The signal strength for this cell network, either the RSSI or RSCP.
		Age               uint32 // The number of milliseconds since this networks was last detected.
		TimingAdvance     uint8  // The timing advance value for this cell network.
	}

	type WifiAccessPoint struct {
		MacAddress         string // The BSSID of the WiFi network.
		SignalStrength     int16  // The received signal strength (RSSI) in dBm.
		Age                uint32 // The number of milliseconds since this network was last detected.
		Channel            uint8  // The WiFi channel, often 1 - 13 for networks in the 2.4GHz range.
		SignalToNoiseRatio uint16 // The current signal to noise ratio measured in dB.
	}

	type Fallbacks struct {
		LAC bool // If no exact cell match can be found, fall back from exact cell position estimates to more coarse grained cell location area estimates, rather than going directly to an even worse GeoIP based estimate.
		IP  bool // If no position can be estimated based on any of the provided data points, fall back to an estimate based on a GeoIP database based on the senders IP address at the time of the query.
	}

	type LBSRequest struct {
		HomeMobileCountryCode uint16            // The mobile country code stored on the SIM card (100-999).
		HomeMobileNetworkCode uint16            // The mobile network code stored on the SIM card (0-32767).
		RadioType             string            // The mobile radio type. Supported values are lte, gsm, cdma, and wcdma.
		Carrier               string            // The clear text name of the cell carrier / operator.
		ConsiderIp            bool              // Should the clients IP address be used to locate it, defaults to true.
		CellTowers            []CellTower       // Array of cell towers
		WifiAccessPoints      []WifiAccessPoint // Array of wifi access points
		IPAddress             string            // Client IP Address
		Fallbacks             *Fallbacks        // The fallback section is a custom addition to the GLS API.
	}

	lbsRequest := LBSRequest{
		RadioType:             "gsm",
		HomeMobileCountryCode: 250,
		HomeMobileNetworkCode: 1,
		ConsiderIp:            false,
		CellTowers: []CellTower{
			{250, 1, 6101, 4765, -62, 0, 0},
			{250, 1, 6101, 4762, -56, 0, 0},
			{250, 1, 6101, 4763, -60, 0, 0},
			{250, 1, 6101, 4766, -75, 0, 0},
			{250, 1, 818, 13000, -87, 0, 0},
			{250, 1, 818, 13049, -83, 0, 0},
			{250, 1, 6101, 4761, -76, 0, 0},
		},
	}

	var lbsResponse LBSResponse
	err = client.Call("LBS.Get", lbsRequest, &lbsResponse)
	if err != nil {
		t.Error("LBS error:", err)
	} else {
		fmt.Println("LBS:", lbsResponse)
	}

	// POI

	place := Place{
		Group:  "test_group",
		ID:     "test_id",
		Name:   "Test Place",
		Center: NewPoint(38.67451, 55.715084),
		Radius: 456.08,
	}
	var placeID string
	err = client.Call("POI.Save", place, &placeID)
	if err != nil {
		t.Error("Save POI error:", err)
	} else {
		fmt.Println("Save POI:", placeID)
	}

	// time.Sleep(time.Second)

	var places = make([]Place, 0)
	err = client.Call("POI.Get", place.Group, &places)
	if err != nil {
		t.Error("Get POI error:", err)
	} else {
		fmt.Println("Get POI:", places)
	}

	// time.Sleep(time.Second)

	placePoint := PlacePoint{
		Group: place.Group,
		Point: place.Center,
	}
	var placeIDs = make([]string, 0)
	err = client.Call("POI.In", placePoint, &placeIDs)
	if err != nil {
		t.Error("In POI error:", err)
	} else {
		fmt.Println("In POI:", placeIDs)
	}

	// time.Sleep(time.Second)

	placeID2 := PlaceID{
		Group: "test_group",
		ID:    "test_id",
	}
	err = client.Call("POI.Delete", placeID2, &placeID)
	if err != nil {
		t.Error("Delete POI error:", err)
	} else {
		fmt.Println("Delete POI:", placeID)
	}

	// time.Sleep(time.Second)

	// KeyStore

	var key string
	var data = DeviceData{
		Device: "deviceid",
		Data:   []byte(`data`),
	}

	err = client.Call("Devices.Save", data, &key)
	if err != nil {
		t.Error("Save to Devices error:", err)
	} else {
		fmt.Println("Save to Devices:", key)
	}

	err = client.Call("Devices.Get", key, &data)
	if err != nil {
		t.Error("Get Devices error:", err)
	} else {
		fmt.Println("Get Devices:", data)
	}

	// data.Data = nil
	// err = client.Call("Devices.Save", data, &key)
	// if err != nil {
	// 	t.Error("Save to Devices error:", err)
	// } else {
	// 	fmt.Println("Save to Devices:", key)
	// }

	resp, err := http.Post("http://localhost:1234"+service.Store.prefix,
		"text/plain", strings.NewReader("test string"))
	if err != nil {
		t.Error("POST Store error:", err)
	} else {
		fmt.Println("POST Store:", resp.Status)
		loc, err := resp.Location()
		if err != nil {
			t.Error(err)
		}
		resp, err := http.Get(loc.String())
		if err != nil {
			t.Error(err)
		}
		fmt.Println("GET Store:", resp.ContentLength)
	}

	var zone string
	err = client.Call("LocTime.Get", NewPoint(37.589431, 55.766242), &zone)
	if err != nil {
		t.Error("Get LocTime error:", err)
	} else {
		fmt.Println("Get LocTime:", data)
	}
	fmt.Println("LocTime zone:", zone) //time.Now().In(&loc))
	loc, err := time.LoadLocation(zone)
	if err != nil {
		t.Error("Bad LocTime zone:", err)
	}
	fmt.Println("LocTime:", time.Now().In(loc))
}
