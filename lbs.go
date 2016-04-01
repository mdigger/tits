package main

import (
	"errors"

	"github.com/mdigger/geolocate"
)

// LBS сервис определения координат по данным сотовых вышек и Wi-Fi.
type LBS struct {
	Type  string // название сервиса (Google, Mozilla, Yandex)
	Token string // токен для пользования сервисом

	locator geolocate.Locator // инициализированный сервис гео-локации
}

// LBSResponse описывает ответ сервиса.
type LBSResponse struct {
	Point    Point   // координаты точки
	Accuracy float32 // точность вычисления (погрешность)
}

// Get передает параметры с данными LBS на внешний сервер геолокации и
// возвращает полученные от сервера данные.
func (s *LBS) Get(req geolocate.Request, resp *LBSResponse) error {
	if s.locator == nil {
		return errors.New("LBS: service not initialized")
	}
	// осуществляем запрос к внешнему сервису геолокации
	respData, err := s.locator.Get(req)
	if err != nil {
		return err
	}
	resp.Point = NewPoint(respData.Location.Lon, respData.Location.Lat)
	resp.Accuracy = respData.Accuracy
	return nil
}
