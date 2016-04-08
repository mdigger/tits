package main

import (
	"errors"

	"gopkg.in/mgo.v2"
	"gopkg.in/mgo.v2/bson"
)

var errPOInotInitialized = errors.New("POI: service not initialized")

// POI описывает сервис работы с местами.
type POI struct {
	coll *mgo.Collection // соединение с MongoDB
}

// Place описывает место с помощью окружности.
type Place struct {
	Group  string     // уникальный идентификатор группы
	ID     string     // уникальный идентификатор
	Name   string     // отображаемое имя
	Center [2]float64 // точка цента окружности
	Radius float64    // радиус окружности в метрах
}

// Save сохраняет информацию о месте в хранилище.
func (p *POI) Save(place Place, id *string) error {
	if p.coll == nil {
		return errPOInotInitialized
	}
	// группа должна быть указана в обязательном порядке
	if place.Group == "" {
		return errors.New("empty group id")
	}
	// добавляем уникальный идентификатор места, если не определено
	if place.ID == "" {
		place.ID = bson.NewObjectId().Hex()
	}
	*id = place.ID
	// уникальный идентификатор составной, включая группу
	sID := PlaceID{
		Group: place.Group,
		ID:    place.ID,
	}
	// добавляем описание окружности в виде полигона для индексации
	storePlace := struct {
		Name    string
		Center  [2]float64
		Radius  float64
		Polygon Polygon
	}{
		Name:    place.Name,
		Center:  place.Center,
		Radius:  place.Radius,
		Polygon: Circle2Polygon(place.Center, place.Radius),
	}
	// инициализируем копию сессии связи с базой данных
	session := p.coll.Database.Session.Copy()
	defer session.Close()
	coll := session.DB(p.coll.Database.Name).C(p.coll.Name)
	_, err := coll.UpsertId(sID, storePlace)
	return err
}

// PlaceID описывает внутренний идентификатор места вместе с группой
type PlaceID struct {
	Group string // идентификатор группы
	ID    string // идентификатор места
}

// Delete удаляет запись о месте из базы данных.
func (p *POI) Delete(pid PlaceID, id *string) error {
	if p.coll == nil {
		return errPOInotInitialized
	}
	*id = pid.ID
	// инициализируем копию сессии связи с базой данных
	session := p.coll.Database.Session.Copy()
	defer session.Close()
	coll := session.DB(p.coll.Database.Name).C(p.coll.Name)
	return coll.RemoveId(pid)
}

// Get возвращает список всех мест, определенных для данной группы.
func (p *POI) Get(group string, list *[]Place) error {
	if p.coll == nil {
		return errPOInotInitialized
	}
	// инициализируем копию сессии связи с базой данных
	session := p.coll.Database.Session.Copy()
	defer session.Close()
	coll := session.DB(p.coll.Database.Name).C(p.coll.Name)
	return coll.Find(bson.M{"_id.group": group}).All(list)
}

// PlacePoint описывает группу и координаты.
type PlacePoint struct {
	Group string // идентификатор группы
	Point Point  // координаты точки
}

// In возвращает список всех мест, в которые входят данные координаты.
func (p *POI) In(place PlacePoint, list *[]string) error {
	if p.coll == nil {
		return errPOInotInitialized
	}
	// инициализируем копию сессии связи с базой данных
	session := p.coll.Database.Session.Copy()
	defer session.Close()
	coll := session.DB(p.coll.Database.Name).C(p.coll.Name)
	return coll.Find(bson.M{
		"_id.group": place.Group,
		"polygon": bson.M{
			"$geoIntersects": bson.M{
				"$geometry": place.Point,
			},
		},
	}).Distinct("_id.id", list)
}
