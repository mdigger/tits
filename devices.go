package main

import (
	"errors"

	"gopkg.in/mgo.v2"
)

// Devices описывает сервис сохранения и получения вспомогательных данных.
type Devices struct {
	coll *mgo.Collection // соединение с MongoDB
}

// DeviceKey описывает ключ для сохранения данных с привязкой к устройствам.
type DeviceKey struct {
	Group  string // идентификатор группы
	Device string // идентификатор устройства
}

// DeviceData описывает данные с привязкой к устройству.
type DeviceData struct {
	DeviceKey `bson:"_id"` // ключ
	Data      interface{}  // данные хранения
}

// Save сохраняет данные с привязкой к устройствам.
func (d *Devices) Save(data DeviceData, key *DeviceKey) error {
	if d.coll == nil {
		return errors.New("Devices: service not initialized")
	}
	if data.DeviceKey.Group == "" {
		return errors.New("empty group id")
	}
	if data.DeviceKey.Device == "" {
		return errors.New("empty device id")
	}
	*key = data.DeviceKey
	session := d.coll.Database.Session.Copy()
	defer session.Close()
	coll := session.DB(d.coll.Database.Name).C(d.coll.Name)
	var err error
	if data.Data != nil {
		_, err = coll.UpsertId(data.DeviceKey, data)
	} else {
		err = coll.RemoveId(data.DeviceKey)
	}
	return err
}

// Get возвращает данные для указанного устройства.
func (d *Devices) Get(key DeviceKey, data *DeviceData) error {
	if d.coll == nil {
		return errors.New("Devices: service not initialized")
	}
	if key.Group == "" {
		return errors.New("empty group id")
	}
	if key.Device == "" {
		return errors.New("empty device id")
	}
	session := d.coll.Database.Session.Copy()
	defer session.Close()
	coll := session.DB(d.coll.Database.Name).C(d.coll.Name)
	return coll.FindId(key).One(data)
}
