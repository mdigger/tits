package main

import (
	"errors"
	"fmt"
	"strings"

	"github.com/mdigger/geolocate"
)

// Locator описывает сервис определения гео-координат на основании данных
// сотовых вышек и Wi-Fi.
type Locator struct {
	Type  string // название сервиса (Google, Mozilla, Yandex)
	Token string // токен для пользования сервисом

	locator geolocate.Locator // инициализированный сервис гео-локации
}

// LBSResponse описывает ответ сервиса.
type LBSResponse struct {
	Point    Point   // координаты точки
	Accuracy float32 // точность вычисления (погрешность)
}

// GetLBS возвращает уточненные координаты по данным LBS, обращаясь к внешним
// сервисам гео-локации.
func (s *TrackInTouch) GetLBS(in geolocate.Request, out *LBSResponse) error {
	if s.LBS == nil || s.LBS.Type == "" {
		return errors.New("LBS service didn't initialized")
	}
	if s.LBS.locator == nil {
		var serviceURL string
		switch strings.ToLower(s.LBS.Type) {
		case "mozilla":
			serviceURL = geolocate.Mozilla
		case "google":
			serviceURL = geolocate.Google
		case "yandex":
			serviceURL = geolocate.Yandex
		default:
			return fmt.Errorf("unknown LBS service name: %s", s.LBS.Type)
		}
		var err error
		s.LBS.locator, err = geolocate.New(serviceURL, s.LBS.Token)
		if err != nil {
			return err
		}
	}

	resp, err := s.LBS.locator.Get(in)
	if err != nil {
		return err
	}
	out.Point = NewPoint(resp.Location.Lon, resp.Location.Lat)
	out.Accuracy = resp.Accuracy
	return nil
}
