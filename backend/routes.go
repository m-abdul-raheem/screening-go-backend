package main

import (
	"github.com/gorilla/mux"
)

func (app *application) routes() *mux.Router {
	r := mux.NewRouter()

	//Login
	r.HandleFunc("/login/", app.LoginHandler).Methods("POST")
	//Register
	r.HandleFunc("/register/", app.RegisterHandler).Methods("POST")

	//CATEGORY CRUD
	//read
	r.HandleFunc("/getCategories/", app.getCategories).Methods("GET")
	//create
	r.HandleFunc("/addCategory/", app.addCategory).Methods("POST")
	//update
	r.HandleFunc("/updateCategory/", app.updateCategory).Methods("POST")
	//delete
	r.HandleFunc("/deleteCategory/", app.deleteCategory).Methods("POST")

	//BOOKS CRUD
	//read - filter by categories
	r.HandleFunc("/getBooks/{categories}", app.getBooksByCategories).Methods("GET")
	//read
	r.HandleFunc("/getBooks/", app.getBooks).Methods("GET")
	//create
	r.HandleFunc("/addBook/", app.addBook).Methods("POST")
	//update
	r.HandleFunc("/updateBook/", app.updateBook).Methods("POST")
	//delete
	r.HandleFunc("/deleteBook/", app.deleteBook).Methods("POST")

	//Cart
	r.HandleFunc("/addCart/", app.addToCart).Methods("POST")
	//r.HandleFunc("/removeCart/", app.addToCart).Methods("POST")
	r.HandleFunc("/buyCart/", app.buyAllInCart).Methods("POST")

	return r
}
