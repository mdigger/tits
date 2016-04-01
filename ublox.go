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
	Servers     []string      // список серверов
	Timeout     time.Duration // время ожидания ответа
	CacheTime   time.Duration // время кеширования ответа от сервисов
	MaxDistance float32       // максимальная дистанция совпадения
	Pacc        uint32        // расстояние погрешности в метрах

	client *http.Client    // http-клиент для запроса
	coll   *mgo.Collection // соединение с MongoDB
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

// Get запрашивает и возвращает данные для инициализации геолокации браслета
// с помощью сервиса U-Blox.
func (u *Ublox) Get(req UbloxRequest, data *[]byte) error {
	if u.client == nil || u.coll == nil {
		return errors.New("UBLOX: service not initialized")
	}
	// ищем данные в кеш для указанного профиля и координат
	session := u.coll.Database.Session.Copy()
	defer session.Close()
	coll := session.DB(u.coll.Database.Name).C(u.coll.Name)
	search := bson.M{
		"profile": req.Profile,
		"point": bson.D{ // важен порядок следования элементов запроса
			{"$nearSphere", req.Point},
			{"$maxDistance", u.MaxDistance},
		}}
	filter := bson.M{"data": 1, "_id": 0}
	var cacheData struct{ Data []byte }
	err := coll.Find(search).Select(filter).One(&cacheData)
	if err == nil {
		*data = cacheData.Data
		return nil
	}
	// данные к кеш не найдены — делаем запрос данных у внешнего сервиса
	*data, err = u.requestServers(req)
	if err != nil {
		return err
	}
	// сохраняем полученные данные в кеш
	return coll.Insert(struct {
		Profile UbloxProfile // профиль
		Point   Point        // координаты
		Data    []byte       // содержимое ответа
		Time    time.Time    // временная метка
	}{
		Profile: req.Profile,
		Point:   req.Point,
		Data:    *data,
		Time:    time.Now(),
	})
}

// requestServers осуществляет запрос к сервису U-Blox, перебирая все доступные
// в конфигурации сервера, и возвращает данные для инициализации браслета.
func (u *Ublox) requestServers(req UbloxRequest) ([]byte, error) {
	// формируем параметры для запроса данных у сервиса
	profile := req.Profile // профиль устройства для запроса
	var queryBuf = new(bytes.Buffer)
	fmt.Fprintf(queryBuf, "token=%s", u.Token)
	if profile.Format != "" {
		fmt.Fprintf(queryBuf, ";format=%s", profile.Format)
	}
	if len(profile.Datatype) > 0 {
		fmt.Fprintf(queryBuf, ";datatype=%s", strings.Join(profile.Datatype, ","))
	}
	if len(profile.GNSS) > 0 {
		fmt.Fprintf(queryBuf, ";gnss=%s", strings.Join(profile.GNSS, ","))
	}
	fmt.Fprintf(queryBuf, ";lon=%f;lat=%f", req.Point[0], req.Point[1])
	if u.Pacc != 300000 && u.Pacc < 6000000 {
		fmt.Fprintf(queryBuf, ";pacc=%d", u.Pacc)
	}
	if profile.FilterOnPos {
		queryBuf.WriteString(";filteronpos")
	}
	query := queryBuf.String()
	// перебираем все сервера сервиса по порядку
	for i, server := range u.Servers {
		reqURL := fmt.Sprintf("%s?%s", server, query)
		data, err := u.getData(reqURL)
		if err == nil {
			return data, nil
		}
		if i == len(u.Servers)-1 {
			return nil, err // для последнего сервера возвращаем ошибку
		}
	}
	return nil, errors.New("UBLOX: no servers")
}

// getData осуществляет запрос к серверу и возвращает данные от него.
func (u *Ublox) getData(url string) ([]byte, error) {
	resp, err := u.client.Get(url)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != 200 {
		return nil, fmt.Errorf("UBLOX: bad response %s", resp.Status)
	}
	return ioutil.ReadAll(resp.Body)
}
