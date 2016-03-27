package main

import (
	"encoding/json"
	"flag"
	"io/ioutil"
	"log"
	"net"
	"net/rpc"
	"time"

	"gopkg.in/mgo.v2"
)

// TrackInTouch описывает настройки конфигурации сервисов.
type TrackInTouch struct {
	MongoDB string   // адрес MongoDB-сервера
	Ublox   *Ublox   // сервис U-Blox
	LBS     *Locator // сервис LBS

	listener net.Listener // TCP-сервер
	session  *mgo.Session // сессия связи с MongoDB
	dbname   string       // название базы данных
}

// LoadConfig читает конфигурацию из файла и возвращает инициализированный
// сервис.
func LoadConfig(filename string) (*TrackInTouch, error) {
	data, err := ioutil.ReadFile(filename)
	if err != nil {
		return nil, err
	}
	service := new(TrackInTouch)
	if err := json.Unmarshal(data, service); err != nil {
		return nil, err
	}
	return service, nil
}

// index инициализирует индексы базы данных
func (s *TrackInTouch) index() error {
	if s.Ublox != nil {
		coll := s.session.DB(s.dbname).C("ublox")
		if err := coll.EnsureIndexKey("profile", "$2dsphere:point"); err != nil {
			return err
		}
		if s.Ublox.CacheTime <= 0 {
			s.Ublox.CacheTime = time.Minute * 30
		}
		if err := coll.EnsureIndex(mgo.Index{
			Key:         []string{"time"},
			ExpireAfter: s.Ublox.CacheTime,
		}); err != nil {
			return err
		}
	}

	return nil
}

// run запускает сервис. При запуске дальнейшее выполнение блокируется,
// пока сервер не будет остановлен.
func (s *TrackInTouch) run(addr string) (err error) {
	// регистрируем обработчики RPC
	if err = rpc.Register(s); err != nil {
		return err
	}
	// инициализируем соединение с MongoDB
	if s.MongoDB == "" {
		s.MongoDB = "mongodb://localhost/"
	}
	di, err := mgo.ParseURL(s.MongoDB)
	if err != nil {
		return err
	}
	if di.Database != "" {
		s.dbname = di.Database
	} else {
		s.dbname = "trackintouch"
	}
	s.session, err = mgo.DialWithInfo(di)
	if err != nil {
		return err
	}
	// проверяем или инициализируем индексы в базе данных
	if err = s.index(); err != nil {
		return err
	}
	// инициализируем TCP-сервер
	tcpAddr, err := net.ResolveTCPAddr("tcp", addr)
	if err != nil {
		return err
	}
	s.listener, err = net.ListenTCP("tcp", tcpAddr)
	if err != nil {
		return err
	}
	rpc.Accept(s.listener) // блокирующий вызов
	return nil
}

// close закрывает соединения.
func (s *TrackInTouch) close() {
	if s.listener != nil {
		s.listener.Close()
		s.listener = nil
	}
	if s.session != nil {
		s.session.Close()
		s.session = nil
	}
}

func main() {
	addr := flag.String("addr", ":7777", "service address")
	config := flag.String("config", "config.json", "configuration filename")
	flag.Parse()

	service, err := LoadConfig(*config)
	if err != nil {
		log.Fatal(err)
	}
	if err := service.run(*addr); err != nil {
		service.close()
		log.Fatal(err)
	}
	service.close()
}
