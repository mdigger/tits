package main

import (
	"bytes"
	"errors"
	"fmt"
	"io/ioutil"
	"net/http"
	"strings"
	"time"

	"gopkg.in/mgo.v2"
	"gopkg.in/mgo.v2/bson"
)

// Ublox описывает сервис для получения инициализационных данных для
// настройки гео-трекинга для браслетов.
type Ublox struct {
	Token       string        // токен для авторизации на сервере
	Pacc        uint32        // расстояние погрешности в метрах
	Servers     []string      // список серверов
	Timeout     time.Duration // время ожидания ответа
	CacheTime   time.Duration // время кеширования ответа от сервисов
	MaxDistance float32       // максимальная дистанция совпадения
	client      *http.Client  // http-клиент для запроса
}

// UbloxProfile описывает профиль возвращаемых данных для данного устройства.
type UbloxProfile struct {
	Datatype    []string // A comma separated list of the data types required by the client (eph, alm, aux, pos)
	Format      string   // Specifies the format of the data returned (mga = UBX- MGA-* (M8 onwards); aid = UBX-AID-* (u7 or earlier))
	GNSS        []string // A comma separated list of the GNSS for which data should be returned (gps, qzss, glo)
	FilterOnPos bool     // If present, the ephemeris data returned to the client will only contain data for the satellites which are likely to be visible from the approximate position provided
}

// UbloxRequest описывает входящие параметры для получения данных инициализации
// геолокации браслета. В них передаются ориентировочные координаты точки и
// профиль, описывающий устройство.
type UbloxRequest struct {
	Point   Point        // координаты точки
	Profile UbloxProfile // профиль устройства
}

// GetUblox возвращает данные для инициализации гео-локации браслетов.
func (s *TrackInTouch) GetUblox(in UbloxRequest, out *[]byte) error {
	if s.Ublox == nil {
		return errors.New("u-blox service didn't initialized")
	}

	var (
		session *mgo.Session
		coll    *mgo.Collection
	)
	if s.session != nil {
		session = s.session.Copy()
		defer session.Close()
		coll = session.DB(s.dbname).C("ublox")
		search := bson.M{
			"profile": in.Profile,
			"point": bson.D{ // важен порядок следования элементов запроса или может быть ошибка!
				{"$nearSphere", in.Point},
				{"$maxDistance", s.Ublox.MaxDistance},
			}}
		var cacheData struct{ Data []byte }
		err := coll.Find(search).Select(bson.M{"data": 1, "_id": 0}).One(&cacheData)
		if err == nil {
			*out = cacheData.Data
			// log.Println("u-blox: from cache")
			return nil
		} else {
			// log.Println("u-blox: cache not found")
		}
	}

	data, err := s.Ublox.GetRequest(in.Point, in.Profile)
	if err != nil {
		// log.Println("u-blox: get request error")
		return err
	}
	*out = data
	if session != nil && coll != nil {
		// сохраняем ответ в хранилище
		err = coll.Insert(&struct {
			Profile UbloxProfile // профиль
			Point   Point        // координаты
			Data    []byte       // содержимое ответа
			Time    time.Time    // временная метка
		}{
			Profile: in.Profile,
			Point:   in.Point,
			Data:    data,
			Time:    time.Now(),
		})
		// log.Println("u-blox: save response")
	}
	return nil
}

// GetRequest осуществляет запрос к серверам и возвращает данные.
func (u *Ublox) GetRequest(point Point, profile UbloxProfile) ([]byte, error) {
	if u.client == nil {
		if u.Timeout <= 0 {
			u.Timeout = time.Minute * 2
		}
		u.client = &http.Client{Timeout: u.Timeout}
	}
	query := u.getQueryParams(point, profile)
	for i, server := range u.Servers {
		reqURL := fmt.Sprintf("%s?%s", server, query)
		data, err := u.getData(reqURL)
		if err != nil {
			if i < len(u.Servers)-1 {
				continue
			}
			return nil, err
		}
		return data, nil
	}
	return nil, errors.New("all servers are not response")
}

// getQueryParams возвращает строку с параметрами запроса для сервиса U-blox.
func (u *Ublox) getQueryParams(point Point, profile UbloxProfile) string {
	var query = new(bytes.Buffer)
	fmt.Fprintf(query, "token=%s", u.Token)
	if profile.Format != "" {
		fmt.Fprintf(query, ";format=%s", profile.Format)
	}
	if len(profile.Datatype) > 0 {
		fmt.Fprintf(query, ";datatype=%s", strings.Join(profile.Datatype, ","))
	}
	if len(profile.GNSS) > 0 {
		fmt.Fprintf(query, ";gnss=%s", strings.Join(profile.GNSS, ","))
	}
	fmt.Fprintf(query, ";lon=%f;lat=%f", point[0], point[1])
	if u.Pacc != 300000 && u.Pacc < 6000000 {
		fmt.Fprintf(query, ";pacc=%d", u.Pacc)
	}
	if profile.FilterOnPos {
		query.WriteString(";filteronpos")
	}
	return query.String()
}

// getData осуществляет запрос к серверу и возвращает данные от него.
func (u *Ublox) getData(url string) ([]byte, error) {
	resp, err := u.client.Get(url)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != 200 {
		return nil, fmt.Errorf("bad response: %s", resp.Status)
	}
	return ioutil.ReadAll(resp.Body)
}
