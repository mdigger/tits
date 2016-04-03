package main

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net"
	"net/http"
	"net/rpc"
	"strings"
	"time"

	"github.com/mdigger/geolocate"

	"gopkg.in/mgo.v2"
)

// Config описывает конфигурацию сервисов.
//
// Каждый из сервисов регистрируется отдельно. Если он не определен, то его
// инициализации не происходит. Т.е. если какой-то из сервисов не предполагается
// использовать, то достаточно его просто не описывать в конфигурации.
type Config struct {
	MongoDB string   // строка для подключения к MongoDB
	Ublox   *Ublox   // настройки сервиса U-Blox
	LBS     *LBS     // настройки LBS-сервиса
	POI     *POI     // настройки сервиса POI
	Devices *Devices // хранилище данных по устройствам

	listener net.Listener // TCP-сервер
}

// LoadConfig читает конфигурацию из файла и возвращает инициализированный
// сервис.
func LoadConfig(filename string) (*Config, error) {
	data, err := ioutil.ReadFile(filename)
	if err != nil {
		return nil, err
	}
	service := new(Config)
	if err := json.Unmarshal(data, service); err != nil {
		return nil, err
	}
	return service, nil
}

// Run запускает сервис по указанному адресу и порту.
// В процессе запуска происходит подключение к MongoDB и проверяется, что
// индексы в базе данных корректно инициализированы.
func (c *Config) Run(addr string) (err error) {
	// инициализируем соединение с MongoDB
	if c.MongoDB == "" {
		c.MongoDB = "mongodb://localhost/"
	}
	di, err := mgo.ParseURL(c.MongoDB) // разбираем строку соединения
	if err != nil {
		return err
	}
	if di.Database == "" {
		di.Database = "trackintouch"
	}
	session, err := mgo.DialWithInfo(di) // устанавливаем соединение
	if err != nil {
		return err
	}
	// инициализируем сервис U-blox и индексы
	if c.Ublox != nil {
		// инициализируем коллекцию для кеширования ответов
		coll := session.DB(di.Database).C("ublox")
		c.Ublox.coll = coll
		// индекс для поиска по профилю и координатам
		err := coll.EnsureIndexKey("profile", "$2dsphere:point")
		if err != nil {
			return err
		}
		// индекс времени жизни данных в кеш
		if c.Ublox.CacheTime <= 0 {
			c.Ublox.CacheTime = time.Minute * 30
		}
		err = coll.EnsureIndex(mgo.Index{
			Key:         []string{"time"},
			ExpireAfter: c.Ublox.CacheTime,
		})
		if err != nil {
			return err
		}
		// инициализируем клиента для запроса данных
		if c.Ublox.Timeout <= 0 {
			c.Ublox.Timeout = time.Minute * 2
		}
		c.Ublox.client = &http.Client{Timeout: c.Ublox.Timeout}
		// регистрируем обработчик
		err = rpc.Register(c.Ublox)
		if err != nil {
			return err
		}
	}
	// инициализируем сервис LBS
	if c.LBS != nil {
		// в зависимости от типа инициализируем разные сервисы LBS
		var serviceURL string
		switch strings.ToLower(c.LBS.Type) {
		case "mozilla":
			serviceURL = geolocate.Mozilla
		case "google":
			serviceURL = geolocate.Google
		case "yandex":
			serviceURL = geolocate.Yandex
		default:
			return fmt.Errorf("unknown LBS service name: %s", c.LBS.Type)
		}
		locator, err := geolocate.New(serviceURL, c.LBS.Token)
		if err != nil {
			return err
		}
		c.LBS.locator = locator
		// регистрируем обработчик
		err = rpc.Register(c.LBS)
		if err != nil {
			return err
		}
	}
	// инициализируем сервис POI
	if c.POI != nil {
		// инициализируем коллекцию для кеширования ответов
		coll := session.DB(di.Database).C("poi")
		c.POI.coll = coll
		// добавляем индекс мест по группам
		err := coll.EnsureIndexKey("_id.group", "$2dsphere:polygon")
		if err != nil {
			return err
		}
		// регистрируем обработчик
		err = rpc.Register(c.POI)
		if err != nil {
			return err
		}
	}
	// инициализируем хранилище данных по устройствам
	if c.Devices != nil {
		coll := session.DB(di.Database).C("devices")
		c.Devices.coll = coll
		// регистрируем обработчик
		err = rpc.Register(c.Devices)
		if err != nil {
			return err
		}
	}
	// инициализируем TCP-сервер
	tcpAddr, err := net.ResolveTCPAddr("tcp", addr)
	if err != nil {
		return err
	}
	listener, err := net.ListenTCP("tcp", tcpAddr)
	if err != nil {
		return err
	}
	c.listener = listener // сохраняем
	rpc.Accept(listener)  // блокирующий вызов
	session.Close()       // закрываем сессию соединения с базой данных
	return nil
}

// Close закрывает подключение к сервису и останавливает его.
func (c *Config) Close() {
	if c.listener != nil {
		c.listener.Close()
		c.listener = nil
	}
}
