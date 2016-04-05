package main

import (
	"io"
	"net/http"
	"path"
	"time"

	"gopkg.in/mgo.v2"
	"gopkg.in/mgo.v2/bson"
)

// Store описывает конфигурацию хранилища файлов.
type Store struct {
	CacheTime time.Duration // время хранения файлов в хранилище

	prefix string      // путь запроса
	grid   *mgo.GridFS // коллекция файлов
}

// ServeHTTP сохраняет файл в хранилище файлов или отдает его.
func (s *Store) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case "GET": // получение файла
		id := path.Base(r.URL.Path) // идентификатор файла
		if !bson.IsObjectIdHex(id) {
			http.Error(w, http.StatusText(http.StatusNotFound), http.StatusNotFound)
			return
		}
		if err := s.get(id, w); err != nil {
			if err == mgo.ErrNotFound {
				http.Error(w, http.StatusText(http.StatusNotFound), http.StatusNotFound)
				return
			}
			http.Error(w, err.Error(), http.StatusInternalServerError)
		}
	case "POST": // сохранение файла
		if id, err := s.save(r); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
		} else {
			w.Header().Set("Location", path.Join(s.prefix, id))
			w.WriteHeader(http.StatusCreated)
		}
	default: // метод не поддерживается
		w.Header().Set("Allow", "GET, POST")
		http.Error(w, http.StatusText(http.StatusMethodNotAllowed),
			http.StatusMethodNotAllowed)
	}
}

// save сохраняет переданный в запросе файл в хранилище.
func (s *Store) save(r *http.Request) (id string, err error) {
	defer r.Body.Close()
	file, err := s.grid.Create("")
	if err != nil {
		return
	}
	defer file.Close()
	if _, err = io.Copy(file, r.Body); err != nil {
		return
	}
	// получаем уникальный идентификатор
	id = file.Id().(bson.ObjectId).Hex()
	// если передан тип, то сохраняем его
	if ctype := r.Header.Get("Content-Type"); ctype != "" {
		file.SetContentType(ctype)
	}
	return
}

// get возвращает содержимое файла
func (s *Store) get(id string, w http.ResponseWriter) error {
	file, err := s.grid.OpenId(id)
	if err != nil {
		return err
	}
	defer file.Close()
	if ctype := file.ContentType(); ctype != "" {
		w.Header().Set("Content-Type", ctype)
	}
	io.Copy(w, file)
	return nil
}
