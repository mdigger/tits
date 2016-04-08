package main

import (
	"fmt"
	"math"

	"gopkg.in/mgo.v2/bson"
)

// Point описывает гео-координаты точки. Последовательность координат:
// долгота (longitude), широта (latitude).
type Point [2]float64

// NewPoint возвращает инициализированную структуру Point. В случае задания
// недопустимых данных для координат, генерируется panic.
func NewPoint(lon, lat float64) Point {
	if lon < -180 || lon > 180 {
		panic("bad longitude")
	}
	if lat < -90 || lat > 90 {
		panic("bad latitude")
	}
	return Point{lon, lat}
}

// GetBSON возвращает представление точки в виде GeoJSON.
// Поддерживает интерфейс кодирования BSON.
func (p Point) GetBSON() (interface{}, error) {
	return struct {
		Type        string
		Coordinates [2]float64
	}{
		Type:        "Point",
		Coordinates: p,
	}, nil
}

// SetBSON десериализует представление точки в формате GoeJSON в формат Point.
// Поддерживает интерфейс декодирования BSON.
func (p *Point) SetBSON(raw bson.Raw) error {
	geopoint := new(struct {
		Type        string
		Coordinates [2]float64
	})
	if err := raw.Unmarshal(geopoint); err != nil {
		return err
	}
	if geopoint.Type != "Point" {
		return fmt.Errorf("bad Geo Point type: %s", geopoint.Type)
	}
	*p = Point(geopoint.Coordinates)
	return nil
}

// Polygon описывает информацию о координатах многоугольника.
// Может так же включать вложенные многоугольники, которые являются "выемками"
// из основного.
type Polygon [][][2]float64

// NewPolygon возвращает новое описание многоугольника, состоящего из заданных
// точек (без изъятий).
func NewPolygon(points ...[2]float64) Polygon {
	p1, p2 := points[0], points[len(points)-1]
	if p1[0] != p2[0] || p1[1] != p2[1] {
		points = append(points, p1)
	}
	return Polygon{points}
}

// GetBSON возвращает представление многоугольника в формате GeoJSON.
func (p Polygon) GetBSON() (interface{}, error) {
	return struct {
		Type        string
		Coordinates [][][2]float64
	}{
		Type:        "Polygon",
		Coordinates: p,
	}, nil
}

// SetBSON декодирует представление многоугольника из формата GeoJSON.
func (p *Polygon) SetBSON(raw bson.Raw) error {
	geopolygon := new(struct {
		Type        string
		Coordinates [][][2]float64
	})
	if err := raw.Unmarshal(geopolygon); err != nil {
		return err
	}
	if geopolygon.Type != "Polygon" {
		return fmt.Errorf("bad Geo Polygon type: %s", geopolygon.Type)
	}
	*p = Polygon(geopolygon.Coordinates)
	return nil
}

const earthRadius float64 = 6378137.0 // радиус Земли в метрах
const circleToPolygonSegments = 16    // количество сегментов круга

// Circle2Polygon возвращает представление круга в виде многоугольника.
// Как не странно, GeoJSON не поддерживает окружности, поэтому приходится
// "конвертировать" окружность в многоугольник, чтобы использовать его в
// индексе MongoDB.
func Circle2Polygon(center [2]float64, radius float64) Polygon {
	rLat := radius / earthRadius * 180.0 / math.Pi
	rLng := rLat / math.Cos(center[1]*math.Pi/180.0)
	dRad := 2.0 * math.Pi / circleToPolygonSegments
	points := make([][2]float64, circleToPolygonSegments+1)
	for i := 0; i <= circleToPolygonSegments; i++ {
		theta := dRad * float64(i)
		x := math.Cos(theta)
		if math.Abs(x) < 0.01 {
			x = 0.0
		}
		y := math.Sin(theta)
		if math.Abs(y) < 0.01 {
			y = 0.0
		}
		points[i] = [2]float64{center[0] + y*rLng, center[1] + x*rLat}
	}
	return NewPolygon(points...)
}
