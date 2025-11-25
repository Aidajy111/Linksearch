package handlers

import (
	"net/http"
)

func MainHandler(w http.ResponseWriter, r *http.Request) {
	w.Write([]byte("Добро пожаловать, мирный житель!"))
}
