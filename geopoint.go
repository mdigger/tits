package main

import "gopkg.in/mgo.v2/bson"

// Point описывает координаты географической точки: долгота, широта.
type Point [2]float32

// NewPoint возвращает инициализированную структуру Point. В случае задания
// недопустимых данных для координат, генерируется panic.
func NewPoint(lon, lat float32) Point {
	if lon < -180 || lon > 180 {
		panic("bad longitude")
	}
	if lat < -90 || lat > 90 {
		panic("bad latitude")
	}
	return Point{lon, lat}
}

// GetBSON возвращает BSON представление GeoJSON-точки.
func (p Point) GetBSON() (interface{}, error) {
	return struct {
		Type        string
		Coordinates [2]float32
	}{
		Type:        "Point",
		Coordinates: p,
	}, nil
}

// SetBSON восстанавливает значение GeoJSON-точки из формата BSON.
func (p *Point) SetBSON(raw bson.Raw) error {
	geopoint := new(struct {
		Type        string
		Coordinates [2]float32
	})
	if err := raw.Unmarshal(geopoint); err != nil {
		return err
	}
	*p = geopoint.Coordinates
	return nil
}
