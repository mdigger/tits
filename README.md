# Сервисы проекта "Track-in-Touch"

Адрес сервера и названия файла с конфигурацией задается в виде параметров при запуске приложения. По умолчанию используется адрес `:7777` и имя файла - `config.json`.


## Конфигурация сервиса

Для определения настроек сервиса используется файл в формате JSON:

	{
	    "MongoDB": "mongodb://localhost/trackintouch",
	    "Ublox": {
	        "Token": "XXXXXXXXXXXXXXX",
	        "Pacc": 100000,
	        "Servers": [
	            "http://online-live1.services.u-blox.com/GetOnlineData.ashx",
	            "http://online-live2.services.u-blox.com/GetOnlineData.ashx"
	        ],
	        "Timeout": 120000000000,
	        "CacheTime": 1800000000000,
	        "MaxDistance": 10000
	    },
	    "LBS": {
	        "Type": "Google",
	        "Token": "XXXXXXXXXXXXXXX"
	    }
	}

- `MongoDB` - содержит строку для подключения к базе данных MongoDB. Данная база используется как внутреннее хранилище данных.
- `Ublox` - описывает настройки доступа к сервису U-Blox:
	- `Token` - токен для доступа к сервису
	- `Pacc` - параметр точности определения данных
	- `Servers` - список URL серверов U-Blox
	- `Timeout` - максимальное время ожидания ответа от сервера в наносекундах (по умолчанию — 2 минуты)
	- `CacheTime` - время хранения ответов сервиса в кеш (по умолчанию — 30 минут). Данный параметр влияет на используемый индекс базы данных, поэтому не может быть в дальнейшем изменен без переиндексации данных.
	- `MaxDistance` - максимальная дистанция в метрах, при которой данные считаются совпадающими (используется при выборке из кеша)
- `LBS` - сервис уточнения координат LBS
	- `Type` - название используемого сервиса (`Google`, `Mozilla`, `Yandex`)
	- `Token` - токен для использования сервиса

Если данные для какого либо сервиса не определены, то он не будет инициализирован и при попытке вызова его методов будет возвращаться ошибка, что сервис не определен.


## Подключение к сервису

Для взаимодействия с сервисом используется стандартный протокол GO RPC поверх TCP-соединения:

	import "net/rpc"

	client, err := rpc.Dial("tcp", ":7777")
	defer client.Close()
	err = client.Call("TrackInTouch.<Method>", in, &out)


## Сервис U-Blox

Возвращает информацию для инициализации гео-локации браслетов. В качестве параметров передаются данные предполагаемых координат и профиля устройства, а в ответ возвращаются бинарные данные для инициализации.

Название метода: `TrackInTouch.GetUblox`.

Входящие данные:

	type UBLOXRequest struct {
		Point [2]float32   // ориентировочные координаты браслета
		Profile struct {   // профиль браслета, передающийся серверу
			Datatype    []string
			Format      string
			GNSS        []string
			FilterOnPos bool
		}
	}

Формат ответа: `[]byte`

**Пример:**

	var in = UBLOXRequest{
		Point: [2]float32{38.67451, 55.715084},
		Profile: struct{
			Datatype:    []string{"pos", "eph", "aux"},
			Format:      "aid",
			GNSS:        []string{"gps"},
			FilterOnPos: true,
		},
	}
	var out []byte
	err = client.Call("TrackInTouch.GetUblox", in, &out)


## LBS

Возвращает уточненные координаты по данным LBS, обращаясь к внешним сервисам. На данный момент поддерживаются сервисы Yandex, Mozilla и Google.

Название метода: `TrackInTouch.GetLBS`.

Входящие данные: 

	type CellTower struct {
		MobileCountryCode uint16  // The mobile country code.
		MobileNetworkCode uint16  // The mobile network code.
		LocationAreaCode  uint16  // The location area code for GSM and WCDMA networks. The tracking area code for LTE networks.
		CellId            uint32  // The cell id or cell identity.
		SignalStrength    int16   // The signal strength for this cell network, either the RSSI or RSCP.
		Age               uint32  // The number of milliseconds since this networks was last detected.
		TimingAdvance     uint8   // The timing advance value for this cell network.
	}

	type WifiAccessPoint struct {
		MacAddress         string  // The BSSID of the WiFi network.
		SignalStrength     int16   // The received signal strength (RSSI) in dBm.
		Age                uint32  // The number of milliseconds since this network was last detected.
		Channel            uint8   // The WiFi channel, often 1 - 13 for networks in the 2.4GHz range.
		SignalToNoiseRatio uint16  // The current signal to noise ratio measured in dB.
	}

	type Fallbacks struct {
		LAC bool // If no exact cell match can be found, fall back from exact cell position estimates to more coarse grained cell location area estimates, rather than going directly to an even worse GeoIP based estimate.
		IP  bool // If no position can be estimated based on any of the provided data points, fall back to an estimate based on a GeoIP database based on the senders IP address at the time of the query.
	}

	type LBSRequest struct {
		HomeMobileCountryCode uint16 // The mobile country code stored on the SIM card (100-999).
		HomeMobileNetworkCode uint16 // The mobile network code stored on the SIM card (0-32767).
		RadioType             string // The mobile radio type. Supported values are lte, gsm, cdma, and wcdma.
		Carrier               string // The clear text name of the cell carrier / operator.
		ConsiderIp            bool   // Should the clients IP address be used to locate it, defaults to true.
		CellTowers            []CellTower // Array of cell towers
		WifiAccessPoints      []WifiAccessPoint // Array of wifi access points
		IPAddress             string // Client IP Address
		Fallbacks             *Fallbacks // The fallback section is a custom addition to the GLS API.
	}


Формат ответа: 

	type LBSResponse struct {
		Point    [2]float32   // координаты точки
		Accuracy float32      // точность вычисления (погрешность)
	}


**Пример:**

	in := LBSRequest{
		CellTowers: []CellTower{
			{250, 2, 7743, 22517, -78, 0, 0},
			{250, 2, 7743, 39696, -81, 0, 0},
			{250, 2, 7743, 22518, -91, 0, 0},
			{250, 2, 7743, 27306, -101, 0, 0},
			{250, 2, 7743, 29909, -103, 0, 0},
			{250, 2, 7743, 22516, -104, 0, 0},
			{250, 2, 7743, 20736, -105, 0, 0},
		},
		WifiAccessPoints: []WifiAccessPoint{
			{"2:18:E4:C8:38:30", -22, 0, 0, 0},
		},
	}
	var out LBSResponse
	err = client.Call("TrackInTouch.GetLBS", in, &out)

