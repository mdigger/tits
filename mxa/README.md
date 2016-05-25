# Сервис MX Admin

Адрес сервера и названия файла с конфигурацией задается в виде параметров при запуске приложения. По умолчанию используется адрес `:7778` и имя файла - `config.json`.

## Конфигурация

Для определения настроек сервиса используется файл в формате JSON:

	{
		URL:             "https://maxhome-msk.dyndns.org/admin/newapi/",
		Login:           "usrsvc",
		Password:        "9801",
		BraceletProfile: "590890912028694065",
		UserProfile:     "",
	}


## Инициализация клиента

Для взаимодействия с сервисом используется стандартный протокол GO RPC поверх HTTP-соединения:

	import "net/rpc"
	client, err := rpc.DialHTTP("tcp", ":7778")
	defer client.Close()

## Добавление

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

Если поле `IsUser` `false`, то данные интерпретируются как браслет, `true` — как пользователь.

	bracelet := User{
		IsUser:    false,
		Login:     "005",
		Password:  "005",
		Ext:       "005",
		CellPhone: "79031111111",
	}

	var id string
	err = client.Call("MXA.Add", bracelet, &id)

В ответ возвращается внутренний уникальный идентификатор в MX.


## Обновление информации о браслете

	bracelet := User{
		RecID:     "14430159392108689",
		Login:     "006",
		Password:  "006",
		Ext:       "006",
		CellPhone: "79032222222",
	}

	var ok bool
	err = client.Call("MXA.Update", bracelet, &ok)

Для обновления информации о браслете необходимо указать уникальный идентификатор в MX `RecID` и те поля, которые вы хотите изменить.

## Удаление информации о браслете

	var ok bool
	err = client.Call("MXA.Delete", "14430159392108689", &ok)


<!-- ## Create user

	session=b3964c61-11c6-40e5-9fed-d45dac081604&
	command=add_user&
	data={
		"newUser":true,
		"devices":[],
		"userId":"t001",
		"firstName":"Bracelet",
		"lastName":"001",
		"login":"001",
		"password":"001",
		"pin":"",
		"userProfile":"590890912028694065",
		"cellPhone":"79031111111",
		"extension":"001"
	}

## Update user

	session=b3964c61-11c6-40e5-9fed-d45dac081604&
	command=update_user&
	data={
		"userRecId":["14430159120614461"],
		"lastName":"002",
		"login":"002",
		"userId":"t002",
		"extension":"002"
	}

## Delete user

	session=28660281-3a9b-4bd0-a7d0-4564039031cc&
	command=delete_user&
	data={
		"userRecId":["14430159619230970"]
	}
 -->