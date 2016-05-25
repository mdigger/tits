package main

import (
	"flag"
	"log"
)

func main() {
	addr := flag.String("addr", ":7778", "service address")
	config := flag.String("config", "config.json", "configuration filename")
	flag.Parse()
	// читаем конфигурацию из файла
	service, err := LoadConfig(*config)
	if err != nil {
		log.Fatal(err)
	}
	// регистрируем и запускаем сервисы
	if err := service.Run(*addr); err != nil {
		log.Fatal(err)
	}
	service.Close()
}
