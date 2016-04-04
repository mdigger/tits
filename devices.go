package main

import (
	"errors"

	"gopkg.in/mgo.v2"
)

// Devices описывает сервис сохранения и получения вспомогательных данных.
type Devices struct {
	coll *mgo.Collection // соединение с MongoDB
}

// DeviceData описывает данные с привязкой к устройству.
type DeviceData struct {
	Device string `bson:"_id"` // ключ
	Data   []byte // данные хранения
}

// Save сохраняет данные с привязкой к устройствам.
func (d *Devices) Save(data DeviceData, key *string) error {
	if d.coll == nil {
		return errors.New("Devices: service not initialized")
	}
	if data.Device == "" {
		return errors.New("empty device id")
	}
	*key = data.Device
	session := d.coll.Database.Session.Copy()
	defer session.Close()
	coll := session.DB(d.coll.Database.Name).C(d.coll.Name)
	var err error
	if data.Data != nil {
		_, err = coll.UpsertId(data.Device, data)
	} else {
		err = coll.RemoveId(data.Device)
	}
	return err
}

// Get возвращает данные для указанного устройства.
func (d *Devices) Get(key string, data *DeviceData) error {
	if d.coll == nil {
		return errors.New("Devices: service not initialized")
	}
	if key == "" {
		return errors.New("empty device id")
	}
	session := d.coll.Database.Session.Copy()
	defer session.Close()
	coll := session.DB(d.coll.Database.Name).C(d.coll.Name)
	return coll.FindId(key).One(data)
}
