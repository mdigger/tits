package main

import (
	"encoding/json"
	"errors"
	"fmt"
)

type User struct {
	IsUser    bool   // флаг пользователя
	RecID     string // внутренний идентификатор MX
	Login     string // логин
	Password  string // пароль для авторизации
	Pin       string // пин-код
	FirstName string // имя
	LastName  string // фамилия
	Ext       string // внутренний номер
	CellPhone string // телефонный номер
	HomePhone string // домашний телефонный номер
	Email     string // почтовый адрес
}

func (c *MXAClient) Add(obj User) (string, error) {
	bracelet := struct {
		ID          string `json:"userId,omitempty"`
		Login       string `json:"login,omitempty"`
		Password    string `json:"password,omitempty"`
		Pin         string `json:"pin,omitempty"`
		FirstName   string `json:"firstName,omitempty"`
		LastName    string `json:"lastName,omitempty"`
		UserProfile string `json:"userProfile,omitempty"`
		Ext         string `json:"extension,omitempty"`
		CellPhone   string `json:"cellPhone,omitempty"`
		HomePhone   string `json:"homePhone,omitempty"`
		Email       string `json:"email,omitempty"`
	}{
		Login:     obj.Login,
		Password:  obj.Password,
		Pin:       obj.Pin,
		FirstName: obj.FirstName,
		LastName:  obj.LastName,
		Ext:       obj.Ext,
		CellPhone: obj.CellPhone,
		HomePhone: obj.HomePhone,
		Email:     obj.Email,
	}
	if obj.IsUser {
		bracelet.ID = fmt.Sprintf("u%s", obj.Login)
		bracelet.UserProfile = c.userProfile
	} else {
		bracelet.ID = fmt.Sprintf("b%s", obj.Login)
		bracelet.UserProfile = c.braceletProfile
		if bracelet.FirstName == "" {
			bracelet.FirstName = "Bracelet"
		}
		if bracelet.LastName == "" {
			bracelet.LastName = obj.Login
		}
	}
	data, err := json.Marshal(bracelet)
	if err != nil {
		return "", err
	}
	// pretty.Println(string(data))
	resp, err := c.post("add_user", data)
	if err != nil {
		return "", err
	}
	userID := new(struct {
		UserRecId string `json:"userRecId"`
		Success   bool   `json:"success"`
	})
	err = json.Unmarshal(resp, userID)
	if err != nil {
		return "", err
	}
	if userID.UserRecId == "" || !userID.Success {
		return "", errors.New("user add error")
	}
	return userID.UserRecId, nil
}

func (c *MXAClient) Update(obj User) error {
	bracelet := struct {
		RecID       []string `json:"userRecId,omitempty"` // внутренний идентификатор MX
		ID          string   `json:"userId,omitempty"`
		Login       string   `json:"login,omitempty"`
		Password    string   `json:"password,omitempty"`
		Pin         string   `json:"pin,omitempty"`
		FirstName   string   `json:"firstName,omitempty"`
		LastName    string   `json:"lastName,omitempty"`
		UserProfile string   `json:"userProfile,omitempty"`
		Ext         string   `json:"extension,omitempty"`
		CellPhone   string   `json:"cellPhone,omitempty"`
		HomePhone   string   `json:"homePhone,omitempty"`
		Email       string   `json:"email,omitempty"`
	}{
		RecID:     []string{obj.RecID},
		Login:     obj.Login,
		Password:  obj.Password,
		FirstName: obj.FirstName,
		LastName:  obj.LastName,
		Ext:       obj.Ext,
		CellPhone: obj.CellPhone,
		HomePhone: obj.HomePhone,
		Email:     obj.Email,
	}
	if obj.IsUser {
		bracelet.ID = fmt.Sprintf("u%s", obj.Login)
		bracelet.UserProfile = c.userProfile
	} else {
		bracelet.ID = fmt.Sprintf("b%s", obj.Login)
		bracelet.UserProfile = c.braceletProfile
		if bracelet.FirstName == "" {
			bracelet.FirstName = "Bracelet"
		}
		if bracelet.LastName == "" {
			bracelet.LastName = obj.Login
		}
	}
	data, err := json.Marshal(bracelet)
	if err != nil {
		return err
	}
	// pretty.Println(string(data))
	resp, err := c.post("update_user", data)
	if err != nil {
		return err
	}
	success := new(struct {
		Success bool `json:"success"`
	})
	err = json.Unmarshal(resp, success)
	if err != nil {
		return err
	}
	if !success.Success {
		return errors.New("user update error")
	}
	return nil
}

func (c *MXAClient) Delete(recID string) error {
	bracelet := struct {
		RecID []string `json:"userRecId,omitempty"` // внутренний идентификатор MX
	}{
		RecID: []string{recID},
	}
	data, err := json.Marshal(bracelet)
	if err != nil {
		return err
	}
	// pretty.Println(string(data))
	resp, err := c.post("delete_user", data)
	if err != nil {
		return err
	}
	success := new(struct {
		Success bool `json:"success"`
	})
	err = json.Unmarshal(resp, success)
	if err != nil {
		return err
	}
	if !success.Success {
		return errors.New("user delete error")
	}
	return nil
}
