package main

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/asaskevich/govalidator"
	"github.com/gorilla/mux"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
)

func (app *application) LoginHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	var u User
	json.NewDecoder(r.Body).Decode(&u)
	fmt.Printf("The user request value %v", u)
	result, err := govalidator.ValidateStruct(u)
	if err != nil {
		println("error: " + err.Error())
		app.serverError(w, err)
		return
	}
	println(result)

	// Find user by email
	m, err := app.users.FindUserByEmail(u.Email)
	if err != nil {
		app.serverError(w, err)
		return
	}
	temp := u.Password
	u.Password = m.Password
	m.Password = temp
	//check password
	credentialError := u.CheckPassword(m.Password)
	if credentialError != nil {
		app.serverError(w, credentialError)
		return
	}

	if u.Email == m.Email {
		tokenString, err := createToken(u.Email)
		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			fmt.Errorf("No username found")
			app.serverError(w, errors.New("user not found"))
		}
		w.WriteHeader(http.StatusOK)
		fmt.Fprint(w, tokenString)
		return
	} else {
		w.WriteHeader(http.StatusUnauthorized)
		fmt.Fprint(w, "Invalid credentials")
	}
}

func (app *application) RegisterHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	var u User
	json.NewDecoder(r.Body).Decode(&u)
	fmt.Printf("The user request value %v", u)

	result, err := govalidator.ValidateStruct(u)
	if err != nil {
		println("error: " + err.Error())
		app.serverError(w, err)
		return
	}
	println(result)

	// Find user by id
	m, err := app.users.FindUserByEmail(u.Email)
	if m != nil {
		app.serverError(w, errors.New("user already exists with id: "+u.Email))
		return

	}

	if err := u.HashPassword(u.Password); err != nil {
		app.serverError(w, err)
		return
	}
	// Insert new user
	insertResult, err := app.users.Insert(u)
	if err != nil {
		app.serverError(w, err)
		return
	}

	app.infoLog.Printf("New user have been created, id=%s", insertResult.InsertedID)
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	w.Write([]byte("User created: " + u.Email))
}

func (app *application) getBooks(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	books := []Book{}
	collbooks := app.users.C.Database().Collection("books")

	//Find books with copies/stock greater than zero
	bookCursor, err := collbooks.Find(context.TODO(), bson.M{"copies": bson.M{"$gt": 0}})
	if err != nil {
		app.serverError(w, err)
		return
	}
	err = bookCursor.All(context.TODO(), &books)
	if err != nil {
		if err == mongo.ErrNoDocuments {
			fmt.Println("No documents found")
		} else {
			app.serverError(w, err)
			return
		}
	}

	// Send response back
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)

	b, err := json.Marshal(books)
	if err != nil {
		app.serverError(w, err)
		return
	}
	w.Write(b)
}

func (app *application) getBooksByCategories(w http.ResponseWriter, r *http.Request) {
	// Get id from incoming url
	vars := mux.Vars(r)
	cats := vars["categories"]

	var books []Book
	categories := strings.Split(cats, ",")
	for _, category := range categories {
		bok, err := app.users.FindBooksByCategory(category)
		if err != nil {
			app.serverError(w, err)
			return
		}
		books = append(books, bok...)
	}

	// Send response back
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)

	b, err := json.Marshal(books)
	if err != nil {
		app.serverError(w, err)
		return
	}
	w.Write(b)
}

func (app *application) addBook(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	tokenString := r.Header.Get("Authorization")
	emailHeader := r.Header.Get("Email")

	if tokenString == "" {
		w.WriteHeader(http.StatusUnauthorized)
		fmt.Fprint(w, "Missing authorization header")
		return
	}
	//tokenString = tokenString[len("Bearer "):]

	emailToken, err := verifyToken(tokenString)
	if err != nil {
		w.WriteHeader(http.StatusUnauthorized)
		fmt.Fprint(w, "Invalid token")
		return
	}
	if emailToken != emailHeader {
		app.serverError(w, errors.New(emailHeader+" Email does not match with Token: "+emailToken))
		return
	}

	fmt.Fprintln(w, "Auth successfull for secured link")

	//check if admin
	isAdmin, err := app.users.CheckAdmin(emailHeader)
	if err != nil {
		app.serverError(w, err)
		return
	} else if !isAdmin {
		app.serverError(w, errors.New(emailHeader+" is not admin"))
		return
	}

	// Define book model
	var book Book
	// Get request information
	err = json.NewDecoder(r.Body).Decode(&book)
	fmt.Printf("The category request value %v", book)
	if err != nil {
		app.serverError(w, err)
		return
	}
	fmt.Printf("The user request value %v", book)

	res1, err := govalidator.ValidateStruct(book)
	if err != nil {
		app.serverError(w, err)
		return
	}
	println(res1)

	coll := app.users.C.Database().Collection("books")

	// Find book by name
	var book2 *Book
	err = coll.FindOne(context.TODO(), bson.M{"title": book.Title}).Decode(&book2)
	if err != nil {
		if err == mongo.ErrNoDocuments {
			fmt.Println("No documents found")
		} else {
			panic(err)
		}
	}
	if book2 != nil {
		app.serverError(w, errors.New("Book already exists with name: "+book.Title))
		return
	}
	res, _ := bson.MarshalExtJSON(book2, false, false)
	fmt.Println(string(res))

	collcat := app.users.C.Database().Collection("categories")
	// Find book by name
	var cat *Category
	err = collcat.FindOne(context.TODO(), bson.M{"categoryName": book.Category}).Decode(&cat)
	if err != nil {
		if err == mongo.ErrNoDocuments {
			fmt.Println("No documents found")
			app.serverError(w, errors.New("Category does not exist with name: "+book.Category))
			return
		} else {
			app.serverError(w, err)
			return
		}
	}

	//Update existing category with new book that is being added
	totalBooks := cat.Books
	if cat.Books == "" {
		totalBooks = book.Title
	} else {
		totalBooks += "," + book.Title
	}

	filter := bson.M{"categoryName": cat.CategoryName}
	// update category books by its name
	update := bson.D{{"$set", bson.D{{Key: "books", Value: totalBooks}}}}
	res2, err := collcat.UpdateOne(context.TODO(), filter, update)
	if err != nil {
		if err == mongo.ErrNoDocuments {
			fmt.Println("No documents found")
		} else {
			app.serverError(w, err)
			return
		}
	}
	fmt.Println(res2)

	result, err := coll.InsertOne(context.TODO(), book)
	if err != nil {
		app.serverError(w, err)
		return
	}
	fmt.Printf("Inserted document with _id: %v\n", result.InsertedID)

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	w.Write([]byte("Book added: " + book.Title))
}

func (app *application) updateBook(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	tokenString := r.Header.Get("Authorization")
	emailHeader := r.Header.Get("Email")

	if tokenString == "" {
		w.WriteHeader(http.StatusUnauthorized)
		fmt.Fprint(w, "Missing authorization header")
		return
	}
	//tokenString = tokenString[len("Bearer "):]

	emailToken, err := verifyToken(tokenString)
	if err != nil {
		w.WriteHeader(http.StatusUnauthorized)
		fmt.Fprint(w, "Invalid token")
		return
	}
	if emailToken != emailHeader {
		app.serverError(w, errors.New(emailHeader+" Email does not match with Token: "+emailToken))
		return
	}

	fmt.Fprintln(w, "Auth successfull for secured link")

	//check if admin
	isAdmin, err := app.users.CheckAdmin(emailHeader)
	if err != nil {
		app.serverError(w, err)
		return
	} else if !isAdmin {
		app.serverError(w, errors.New(emailHeader+" is not admin"))
		return
	}

	// Define book model
	var book Book
	// Get request information
	err = json.NewDecoder(r.Body).Decode(&book)
	fmt.Printf("The book request value %v", book)
	if err != nil {
		app.serverError(w, err)
		return
	}

	res1, err := govalidator.ValidateStruct(book)
	if err != nil {
		app.serverError(w, err)
		return
	}
	println(res1)

	// Find book by name
	coll := app.users.C.Database().Collection("books")
	filter := bson.M{"title": book.Title}
	res, err := coll.ReplaceOne(context.TODO(), filter, book)
	if err != nil {
		if err == mongo.ErrNoDocuments {
			fmt.Println("No documents found")
		} else {
			app.serverError(w, err)
			return
		}
	}
	fmt.Println(res)
}

func (app *application) deleteBook(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	tokenString := r.Header.Get("Authorization")
	emailHeader := r.Header.Get("Email")

	if tokenString == "" {
		w.WriteHeader(http.StatusUnauthorized)
		fmt.Fprint(w, "Missing authorization header")
		return
	}
	//tokenString = tokenString[len("Bearer "):]

	emailToken, err := verifyToken(tokenString)
	if err != nil {
		w.WriteHeader(http.StatusUnauthorized)
		fmt.Fprint(w, "Invalid token")
		return
	}
	if emailToken != emailHeader {
		app.serverError(w, errors.New(emailHeader+" Email does not match with Token: "+emailToken))
		return
	}

	fmt.Fprintln(w, "Auth successfull for secured link")

	//check if admin
	isAdmin, err := app.users.CheckAdmin(emailHeader)
	if err != nil {
		app.serverError(w, err)
		return
	} else if !isAdmin {
		app.serverError(w, errors.New(emailHeader+" is not admin"))
		return
	}

	// Define book model
	var book Book
	// Get request information
	err = json.NewDecoder(r.Body).Decode(&book)
	fmt.Printf("The book request value %v", book)
	if err != nil {
		app.serverError(w, err)
		return
	}

	//Update existing category with book that is being deleted

	bookdb, err := app.users.FindBookByName(book.Title)
	if err != nil {
		app.serverError(w, err)
		return
	}

	collcat := app.users.C.Database().Collection("categories")

	var books []Book
	books, err = app.users.FindBooksByCategory(bookdb.Category)
	for i, v := range books {
		if v.Title == book.Title {
			books = append(books[:i], books[i+1:]...)
			break
		}
	}
	var booksstr []string
	for _, v := range books {
		booksstr = append(booksstr, v.Title)
	}

	totalBooks := strings.Join(booksstr, ",")
	fmt.Println("Total books now: " + totalBooks)

	filter := bson.M{"categoryName": bookdb.Category}
	// update category books by its name
	update := bson.D{{"$set", bson.D{{Key: "books", Value: totalBooks}}}}
	res2, err := collcat.UpdateOne(context.TODO(), filter, update)
	if err != nil {
		if err == mongo.ErrNoDocuments {
			fmt.Println("No documents found")
		} else {
			app.serverError(w, err)
			return
		}
	}
	fmt.Println(res2)

	// Delete book by name
	coll := app.users.C.Database().Collection("books")

	filter = bson.M{"title": book.Title}

	res, err := coll.DeleteOne(context.TODO(), filter)
	if err != nil {
		if err == mongo.ErrNoDocuments {
			fmt.Println("No documents found")
		} else {
			app.serverError(w, err)
			return
		}
	}
	fmt.Println(res)
}

func (app *application) getCategories(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	categories := []Category{}
	collcat := app.users.C.Database().Collection("categories")

	catCursor, err := collcat.Find(context.TODO(), bson.M{})
	if err != nil {
		app.serverError(w, err)
		return
	}
	err = catCursor.All(context.TODO(), &categories)
	if err != nil {
		if err == mongo.ErrNoDocuments {
			fmt.Println("No documents found")
		} else {
			app.serverError(w, err)
			return
		}
	}

	// Send response back
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	b, err := json.Marshal(categories)
	if err != nil {
		app.serverError(w, err)
		return
	}
	w.Write(b)
}

func (app *application) addCategory(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	tokenString := r.Header.Get("Authorization")
	emailHeader := r.Header.Get("Email")

	if tokenString == "" {
		w.WriteHeader(http.StatusUnauthorized)
		fmt.Fprint(w, "Missing authorization header")
		return
	}
	//tokenString = tokenString[len("Bearer "):]

	emailToken, err := verifyToken(tokenString)
	if err != nil {
		w.WriteHeader(http.StatusUnauthorized)
		fmt.Fprint(w, "Invalid token")
		return
	}
	if emailToken != emailHeader {
		app.serverError(w, errors.New(emailHeader+" Email does not match with Token: "+emailToken))
		return
	}

	fmt.Fprintln(w, "Auth successfull for secured link")

	//check if admin
	isAdmin, err := app.users.CheckAdmin(emailHeader)
	if err != nil {
		app.serverError(w, err)
		return
	} else if !isAdmin {
		app.serverError(w, errors.New(emailHeader+" is not admin"))
		return
	}

	// Define category model
	var cat Category
	// Get request information
	err = json.NewDecoder(r.Body).Decode(&cat)
	fmt.Printf("The category request value %v", cat)
	if err != nil {
		app.serverError(w, err)
		return
	}
	fmt.Printf("The user request value %v", cat)

	coll := app.users.C.Database().Collection("categories")

	res1, err := govalidator.ValidateStruct(cat)
	if err != nil {
		app.serverError(w, err)
		return
	}
	println(res1)

	// Find category by name
	var cat2 *Category
	err = coll.FindOne(context.TODO(), bson.M{"categoryName": cat.CategoryName}).Decode(&cat2)
	if err != nil {
		if err == mongo.ErrNoDocuments {
			fmt.Println("No documents found")
		} else {
			panic(err)
		}
	}
	if cat2 != nil {
		app.serverError(w, errors.New("category already exists with name: "+cat.CategoryName))
		return
	}
	res, _ := bson.MarshalExtJSON(cat2, false, false)
	fmt.Println(string(res))

	result, err := coll.InsertOne(context.TODO(), cat)
	if err != nil {
		app.serverError(w, err)
		return
	}
	fmt.Printf("Inserted document with _id: %v\n", result.InsertedID)

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	w.Write([]byte("Category added: " + cat.CategoryName))
}

func (app *application) updateCategory(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	tokenString := r.Header.Get("Authorization")
	emailHeader := r.Header.Get("Email")

	if tokenString == "" {
		w.WriteHeader(http.StatusUnauthorized)
		fmt.Fprint(w, "Missing authorization header")
		return
	}
	//tokenString = tokenString[len("Bearer "):]

	emailToken, err := verifyToken(tokenString)
	if err != nil {
		w.WriteHeader(http.StatusUnauthorized)
		fmt.Fprint(w, "Invalid token")
		return
	}
	if emailToken != emailHeader {
		app.serverError(w, errors.New(emailHeader+" Email does not match with Token: "+emailToken))
		return
	}

	fmt.Fprintln(w, "Auth successfull for secured link")

	//check if admin
	isAdmin, err := app.users.CheckAdmin(emailHeader)
	if err != nil {
		app.serverError(w, err)
		return
	} else if !isAdmin {
		app.serverError(w, errors.New(emailHeader+" is not admin"))
		return
	}

	// Define category model
	var cat Category
	// Get request information
	err = json.NewDecoder(r.Body).Decode(&cat)
	fmt.Printf("The category request value %v", cat)
	if err != nil {
		app.serverError(w, err)
		return
	}

	res1, err := govalidator.ValidateStruct(cat)
	if err != nil {
		app.serverError(w, err)
		return
	}
	println(res1)

	// Find category by name

	coll := app.users.C.Database().Collection("categories")
	filter := bson.M{"categoryName": cat.CategoryName}
	// update category books by its name
	update := bson.D{{"$set", bson.D{{Key: "books", Value: cat.Books}}}}

	res, err := coll.UpdateOne(context.TODO(), filter, update)
	if err != nil {
		if err == mongo.ErrNoDocuments {
			fmt.Println("No documents found")
		} else {
			app.serverError(w, err)
			return
		}
	}
	fmt.Println(res)
}

func (app *application) deleteCategory(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	tokenString := r.Header.Get("Authorization")
	emailHeader := r.Header.Get("Email")

	if tokenString == "" {
		w.WriteHeader(http.StatusUnauthorized)
		fmt.Fprint(w, "Missing authorization header")
		return
	}
	//tokenString = tokenString[len("Bearer "):]

	emailToken, err := verifyToken(tokenString)
	if err != nil {
		w.WriteHeader(http.StatusUnauthorized)
		fmt.Fprint(w, "Invalid token")
		return
	}
	if emailToken != emailHeader {
		app.serverError(w, errors.New(emailHeader+" Email does not match with Token: "+emailToken))
		return
	}

	fmt.Fprintln(w, "Auth successfull for secured link")

	//check if admin
	isAdmin, err := app.users.CheckAdmin(emailHeader)
	if err != nil {
		app.serverError(w, err)
		return
	} else if !isAdmin {
		app.serverError(w, errors.New(emailHeader+" is not admin"))
		return
	}

	// Define category model
	var cat Category
	// Get request information
	err = json.NewDecoder(r.Body).Decode(&cat)
	fmt.Printf("The category request value %v", cat)
	if err != nil {
		app.serverError(w, err)
		return
	}

	// Find category by name
	coll := app.users.C.Database().Collection("categories")

	filter := bson.M{"categoryName": cat.CategoryName}

	res, err := coll.DeleteOne(context.TODO(), filter)
	if err != nil {
		if err == mongo.ErrNoDocuments {
			fmt.Println("No documents found")
		} else {
			app.serverError(w, err)
			return
		}
	}
	fmt.Println(res)
}

func (app *application) addToCart(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	tokenString := r.Header.Get("Authorization")
	emailHeader := r.Header.Get("Email")

	if tokenString == "" {
		w.WriteHeader(http.StatusUnauthorized)
		fmt.Fprint(w, "Missing authorization header")
		return
	}

	emailToken, err := verifyToken(tokenString)
	if err != nil {
		w.WriteHeader(http.StatusUnauthorized)
		fmt.Fprint(w, "Invalid token")
		return
	}
	if emailToken != emailHeader {
		app.serverError(w, errors.New(emailHeader+" Email does not match with Token: "+emailToken))
		return
	}

	fmt.Fprintln(w, "Auth successfull for secured link")

	// Define book model
	var book Book
	// Get request information
	err = json.NewDecoder(r.Body).Decode(&book)
	fmt.Printf("The book request value %v", book)
	if err != nil {
		app.serverError(w, err)
		return
	}
	fmt.Printf("The book request value %v", book)
	// Find user by id
	m, err := app.users.FindBookByName(book.Title)
	if m == nil {
		app.serverError(w, errors.New(book.Title+" Book doesn't exist."))
		return
	}

	if m.Copies < 1 {
		app.serverError(w, errors.New(book.Title+" Book is out of Stock"))
		return
	}

	coll := app.users.C.Database().Collection("cart")

	// Find Cart by email
	var cart *Cart
	var cart2 Cart
	err = coll.FindOne(context.TODO(), bson.M{"userEmail": emailToken}).Decode(&cart)
	if err != nil {
		if err == mongo.ErrNoDocuments {
			fmt.Println("No documents found")
		} else {
			panic(err)
		}
	}
	if cart != nil {
		books := strings.Split(cart.Books, ",")
		for _, b := range books {
			if b == book.Title {
				app.serverError(w, errors.New("Book already exists in Cart, name: "+book.Title))
				return
			}
		}
		cart.Books += "," + book.Title
		filter := bson.M{"userEmail": emailToken}
		res, err := coll.ReplaceOne(context.TODO(), filter, cart)
		if err != nil {
			app.serverError(w, err)
			return
		}
		fmt.Printf("Update document with _id: %v\n", res)

		_, err = app.users.UpdateBookStock(m.Title, -1)
		if err != nil {
			if err.Error() == "ErrNoDocuments" {
				app.infoLog.Println("User not found")
				return
			}
			// Any other error will send an internal server error
			app.serverError(w, err)
			return
		}

		//Remove book from cart after time
		time.AfterFunc(30*time.Minute, func() {
			removeFromCart(1, emailToken, book.Title, app)
		})

	} else {
		cart2.UserEmail = emailToken
		cart2.Books = book.Title

		result, err := coll.InsertOne(context.TODO(), cart2)
		if err != nil {
			app.serverError(w, err)
			return
		}
		fmt.Printf("Inserted document with _id: %v\n", result.InsertedID)

		_, err = app.users.UpdateBookStock(m.Title, -1)
		if err != nil {
			if err.Error() == "ErrNoDocuments" {
				app.infoLog.Println("User not found")
				return
			}
			// Any other error will send an internal server error
			app.serverError(w, err)
			return
		}

		//Remove book from cart after time
		time.AfterFunc(30*time.Minute, func() {
			removeFromCart(1, emailToken, book.Title, app)
		})
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	w.Write([]byte("Added in cart: " + book.Title))
}

func (app *application) buyAllInCart(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	tokenString := r.Header.Get("Authorization")
	emailHeader := r.Header.Get("Email")

	if tokenString == "" {
		w.WriteHeader(http.StatusUnauthorized)
		fmt.Fprint(w, "Missing authorization header")
		return
	}
	//tokenString = tokenString[len("Bearer "):]

	emailToken, err := verifyToken(tokenString)
	if err != nil {
		w.WriteHeader(http.StatusUnauthorized)
		fmt.Fprint(w, "Invalid token")
		return
	}
	if emailToken != emailHeader {
		app.serverError(w, errors.New(emailHeader+" Email does not match with Token: "+emailToken))
		return
	}

	fmt.Fprintln(w, "Auth successfull for secured link")

	coll := app.users.C.Database().Collection("cart")

	// Find Cart by email
	var cart *Cart
	err = coll.FindOne(context.TODO(), bson.M{"userEmail": emailToken}).Decode(&cart)
	if err != nil {
		if err == mongo.ErrNoDocuments {
			fmt.Println("No documents found")
		} else {
			panic(err)
		}
	}
	if cart.Books != "" {
		books := strings.Split(cart.Books, ",")
		for _, b := range books {
			removeFromCart(0, emailToken, b, app)
		}
	} else {
		app.serverError(w, errors.New("No books in cart for user "+emailToken))
		return
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	w.Write([]byte("Bought all in cart for user: " + emailToken))
}

// stockEffect parameter tells us if book amount will be increased/decreased/remain same
// Increased if cart timer runs out i.e stockEffect = 1
// remains same if books are bought i.e stockEffect = 0
func removeFromCart(stockEffect int, userEmail string, bookTitle string, app *application) {
	bookexisted, err := app.users.RemoveBookFromCart(userEmail, bookTitle)
	if err != nil {
		return
	}
	if bookexisted {
		_, err = app.users.UpdateBookStock(bookTitle, stockEffect)
		if err != nil {
			return
		}
	}
}
