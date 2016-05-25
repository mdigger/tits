package main

import (
	"encoding/json"
	"io/ioutil"
	"net"
	"net/http"
	"net/rpc"
)

type Config struct {
	URL             string
	Login           string
	Password        string
	BraceletProfile string
	UserProfile     string

	listener net.Listener // TCP-сервер
}

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

func (c *Config) Run(addr string) (err error) {
	err = rpc.Register(&MXA{
		mxa: NewMXAClient(c),
	})
	if err != nil {
		return err
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
	// rpc.Accept(listener)  // блокирующий вызов
	rpc.HandleHTTP()                 // регистрируем обработку по HTTP RPC
	return http.Serve(listener, nil) // запускаем обработчик HTTP
}

// Close закрывает подключение к сервису и останавливает его.
func (c *Config) Close() {
	if c.listener != nil {
		c.listener.Close()
		c.listener = nil
	}
}
