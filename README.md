# Сервис Track in Touch

<!-- [![Build Status](https://travis-ci.org/mdigger/tits.svg)](https://travis-ci.org/mdigger/tits)
 -->
Адрес сервера и названия файла с конфигурацией задается в виде параметров при запуске приложения. По умолчанию используется адрес `:7777` и имя файла - `config.json`.


## Конфигурация

Для определения настроек сервиса используется файл в формате JSON:

	{
	    "MongoDB": "mongodb://localhost/testtits",
	    "Ublox": {
	        "Token": "XXXXXXXXXXXXXXXXXXXXX",
	        "Servers": [
	            "http://online-live1.services.u-blox.com/GetOnlineData.ashx",
	            "http://online-live2.services.u-blox.com/GetOnlineData.ashx"
	        ],
	        "Timeout": 120000000000,
	        "CacheTime": 1800000000000,
	        "MaxDistance": 10000,
	        "Pacc": 100000
	    },
	    "LBS": {
	        "Type": "Google",
	        "Token": "XXXXXXXXXXXXXXXXXXXXX"
	    },
	    "POI": {},
	    "Devices": {}
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
- `POI` - не содержит дополнительных настроек и простого указания достаточно для инициализации сервиса
- `Devices` - позволяет сохранять дополнительную информацию об устройстве

Если данные для какого либо сервиса не определены, то он не будет инициализирован и при попытке вызова его методов будет возвращаться ошибка, что сервис не определен.


## Инициализация клиента

Для взаимодействия с сервисом используется стандартный протокол GO RPC поверх TCP-соединения:

	import "net/rpc"
	client, err := rpc.Dial("tcp", ":7777")
	...
	client.Close()


## Сервис U-BLOX

Возвращает информацию для инициализации гео-локации браслетов. В качестве параметров передаются данные предполагаемых координат и профиля устройства, а в ответ возвращаются бинарные данные для инициализации.


### Запрос данных для инициализации

Название метода: `Ublox.Get`.

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
	err = client.Call("Ublox.Get", in, &out)


## Сервис LBS

Возвращает уточненные координаты по данным LBS, обращаясь к внешним сервисам. На данный момент поддерживаются сервисы Yandex, Mozilla и Google.


### Конвертация данных LBS в реальные координаты

Название метода: `LBS.Get`.

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
	err = client.Call("LBS.Get", in, &out)


##  Сервис работы с определением мест (PoI)

Места определяются в виде окружности, задавая координаты географической точки и радиуса в метрах.

### Сохранение и изменение описания PoI

Название метода: `POI.Save`.

Входящие данные: 

	type Place struct {
		Group  string     // уникальный идентификатор группы
		ID     string     // уникальный идентификатор места
		Name   string     // отображаемое имя
		Center [2]float32 // точка цента окружности
		Radius float32    // радиус окружности в метрах
	}

Формат ответа: `*string` — уникальный идентификатор места

Если уникальный идентификатор места не задан, то он будет присвоен автоматически. В противном случае, описание места будет сохранено с данных идентификатором.

**Пример**

	place := Place{
		Group:  "test_group",
		ID:     "",
		Name:   "Test Place",
		Center: [2]float32{38.67451, 55.715084},
		Radius: 456.08,
	}
	var placeID string
	err = client.Call("POI.Save", place, &placeID)


### Удаление описания PoI

Для удаления описания места необходимо указать идентификатор группы и идентификатор места.

Название метода: `POI.Delete`.

Входящие данные: 

	type PlaceID struct {
		Group  string     // уникальный идентификатор группы
		ID     string     // уникальный идентификатор места
	}

Формат ответа: `*string` — уникальный идентификатор места

**Пример**

	placeID2 := PlaceID{
		Group: "test_group",
		ID:    "test_id",
	}
	err = client.Call("POI.Delete", placeID2, &placeID)


### Запрос списка PoI

Название метода: `POI.Get`.

Входящие данные: `string` - идентификатор группы.

Формат ответа: `*[]Place` - массив описания мест.

**Пример**

	var places = make([]Place, 0)
	err = client.Call("POI.Get", groupID, &places)


### Получение списка PoI для текущего местоположения

Название метода: `POI.In`.

Входящие данные:

	type PlacePoint struct {
		Group string // идентификатор группы
		Point [2]float32  // координаты точки
	}

Формат ответа: `*[]string` - список идентификаторов мест, в которые входит данная точка.

**Пример**

	placePoint := PlacePoint{
		Group: place.Group,
		Point: place.Center,
	}
	var placeIDs = make([]string, 0)
	err = client.Call("POI.In", placePoint, &placeIDs)


## Информация об устройствах

Данный сервис позволяет сохранять произвольную информацию с привязкой с идентификатору устройства.


### Сохранение информации об устройстве

Название метода: `Devices.Save`.

	type DeviceData struct {
		Device	string       // ключ
		Data    interface{}  // данные хранения
	}

**Пример**

	var key string
	var data = DeviceData{
		Device: "deviceid",
		Data: "тестовые данные",
	}
	err = client.Call("Devices.Save", data, &key)

Если данные будут пустые, то запись из хранилище с этим ключем будет удалена.


### Получение информации об устройстве

Название метода: `Devices.Get`.

**Пример**

	var data DeviceData
	err = client.Call("Devices.Get", "deviceid", &data)


