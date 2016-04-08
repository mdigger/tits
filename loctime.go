package main

import (
	"errors"

	"github.com/bradfitz/latlong"
)

// LocTime описывает сервис для получения информации о временной зоне
// для гео-координат.
type LocTime struct{}

// Get возвращает описание временной зоны для указанных координат.
func (l *LocTime) Get(p Point, zone *string) (err error) {
	*zone = latlong.LookupZoneName(p[1], p[0])
	if *zone == "tables not generated yet" {
		return errors.New("LocTime: tables data not initialized")
	}
	if *zone == "" {
		return errors.New("LocTime: unknown zone")
	}
	return nil
}
