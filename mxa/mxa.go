package main

import (
	"crypto/tls"
	"encoding/json"
	"errors"
	"io/ioutil"
	"net"
	"net/http"
	"net/url"
	"time"
)

type MXAClient struct {
	url             string       // адрес сервера
	client          *http.Client // HTTP-клиент для запросов к API
	loginData       url.Values   // информация для логина
	braceletProfile string       // идентификатор профиля для браслета
	userProfile     string       // идентификатор профиля для пользователя
	session         string       // ключ сессии
	sessionTime     time.Time    // время получения сессии
}

func NewMXAClient(config *Config) *MXAClient {
	loginData, _ := json.Marshal(struct {
		Login    string `json:"login"`
		Password string `json:"password"`
	}{config.Login, config.Password})
	params := url.Values{}
	params.Set("data", string(loginData))
	return &MXAClient{
		url: config.URL,
		client: &http.Client{
			Transport: &http.Transport{
				Proxy: http.ProxyFromEnvironment,
				Dial: (&net.Dialer{
					Timeout:   30 * time.Second,
					KeepAlive: 30 * time.Second,
				}).Dial,
				TLSHandshakeTimeout: 10 * time.Second,
				TLSClientConfig: &tls.Config{
					InsecureSkipVerify: true, // отключаем проверку валидности
				},
			},
			Timeout: time.Second * 20, // 20-секундный таймаут
		},
		loginData:       params,
		braceletProfile: config.BraceletProfile,
		userProfile:     config.UserProfile,
	}
}

func (c *MXAClient) Login() error {
	resp, err := c.client.PostForm(c.url, c.loginData)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode != 200 {
		return errors.New(resp.Status)
	}
	data, _ := ioutil.ReadAll(resp.Body)
	session := new(struct {
		Session string `json:"session"`
	})
	err = json.Unmarshal(data, session)
	if err != nil {
		return err
	}
	if session.Session == "" {
		return errors.New("empty session key")
	}
	c.session = session.Session
	c.sessionTime = time.Now()
	return nil
}

func (c *MXAClient) post(command string, data []byte) ([]byte, error) {
	if time.Since(c.sessionTime) > time.Minute*15 {
		if err := c.Login(); err != nil {
			return nil, err
		}
	}
	params := url.Values{}
	params.Set("session", c.session)
	params.Set("command", command)
	params.Set("data", string(data))
	resp, err := c.client.PostForm(c.url, params)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	data, _ = ioutil.ReadAll(resp.Body)
	if resp.StatusCode != 200 {
		return data, errors.New(resp.Status)
	}
	return data, nil
}
