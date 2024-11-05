package main

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
)

// UserModel represent a mgo database session with a user model data.
type UserModel struct {
	C *mongo.Collection
}

// All method will be used to get all records from the users table.
func (m *UserModel) All() ([]User, error) {
	// Define variables
	ctx := context.TODO()
	uu := []User{}

	// Find all users
	userCursor, err := m.C.Find(ctx, bson.M{})
	if err != nil {
		return nil, err
	}
	err = userCursor.All(ctx, &uu)
	if err != nil {
		return nil, err
	}

	return uu, err
}

// All method will be used to get all records from the users table.
func (m *UserModel) AllCategories() ([]Category, error) {
	// Define variables
	ctx := context.TODO()
	uu := []Category{}

	// Find all users
	userCursor, err := m.C.Find(ctx, bson.M{})
	if err != nil {
		return nil, err
	}
	err = userCursor.All(ctx, &uu)
	if err != nil {
		return nil, err
	}

	return uu, err
}

// FindByID will be used to find a new user registry by id
func (m *UserModel) FindByID(id int) (*User, error) {
	//p, err := primitive.ObjectIDFromHex(id)
	//if err != nil {
	//	return nil, err
	//}

	// Find user by id
	var user = User{}
	err := m.C.FindOne(context.TODO(), bson.M{"id": id}).Decode(&user)
	if err != nil {
		// Checks if the user was not found
		if err == mongo.ErrNoDocuments {
			return nil, errors.New("ErrNoDocuments")
		}
		return nil, err
	}

	return &user, nil
}

// FindUserByEmail will be used to find a new user registry by id
func (m *UserModel) FindUserByEmail(email string) (*User, error) {

	// Find user by id
	var user = User{}
	err := m.C.FindOne(context.TODO(), bson.M{"email": email}).Decode(&user)
	if err != nil {
		// Checks if the user was not found
		if err == mongo.ErrNoDocuments {
			return nil, errors.New("ErrNoDocuments")
		}
		return nil, err
	}

	return &user, nil
}

// FindUserByEmail will be used to find a new user registry by id
func (m *UserModel) FindBookByName(title string) (*Book, error) {

	// Find user by id
	var book = Book{}
	coll := m.C.Database().Collection("books")
	err := coll.FindOne(context.TODO(), bson.M{"title": title}).Decode(&book)
	if err != nil {
		// Checks if the book was not found
		if err == mongo.ErrNoDocuments {
			return nil, errors.New("ErrNoDocuments")
		}
		return nil, err
	}

	return &book, nil
}

// FindBooksByCategory will be used to all books with same category
func (m *UserModel) FindBooksByCategory(category string) ([]Book, error) {

	// Find user by id
	book := []Book{}
	coll := m.C.Database().Collection("books")

	cursor, err := coll.Find(context.TODO(), bson.M{"category": category, "copies": bson.M{"$gt": 0}})
	if err != nil {
		return nil, errors.New("ErrNoDocuments")
	}
	if err = cursor.All(context.TODO(), &book); err != nil {
		return nil, errors.New("ErrNoDocuments")
	}
	for _, result := range book {
		res, _ := bson.MarshalExtJSON(result, false, false)
		fmt.Println(string(res))
	}

	return book, nil
}

func (m *UserModel) CheckAdmin(email string) (bool, error) {
	//p, err := primitive.ObjectIDFromHex(id)
	//if err != nil {
	//	return nil, err
	//}

	// Find user by id
	var user = User{}
	err := m.C.FindOne(context.TODO(), bson.M{"email": email}).Decode(&user)
	if err != nil {
		// Checks if the user was not found
		if err == mongo.ErrNoDocuments {
			return false, errors.New("ErrNoDocuments")
		}
		return false, err
	}

	fmt.Println(user.IsAdmin)
	if user.IsAdmin {
		return true, nil
	} else {
		return false, nil
	}
}

// Insert will be used to insert a new user
func (m *UserModel) Insert(user User) (*mongo.InsertOneResult, error) {
	return m.C.InsertOne(context.TODO(), user)
}

// FindUserByEmail will be used to find a new user registry by id
func (m *UserModel) UpdateBookStock(title string, stock int) (*Book, error) {

	// Find user by id
	var book = Book{}
	coll := m.C.Database().Collection("books")
	err := coll.FindOne(context.TODO(), bson.M{"title": title}).Decode(&book)
	if err != nil {
		// Checks if the book was not found
		if err == mongo.ErrNoDocuments {
			return nil, errors.New("ErrNoDocuments")
		}
		return nil, err
	}

	// Find book by name
	filter := bson.M{"title": book.Title}

	update := bson.D{{"$set", bson.D{{Key: "copies", Value: book.Copies + stock}}}}

	_, err = coll.UpdateOne(context.TODO(), filter, update)
	if err != nil {
		if err == mongo.ErrNoDocuments {
			fmt.Println("No documents found")
		} else {
			return nil, err
		}
	}

	return &book, nil
}

// returnn bool if book existed in card for this user
func (m *UserModel) RemoveBookFromCart(email string, bookTitle string) (bool, error) {

	// Find user by id
	var cart = Cart{}
	coll := m.C.Database().Collection("cart")
	err := coll.FindOne(context.TODO(), bson.M{"userEmail": email}).Decode(&cart)
	if err != nil {
		// Checks if the book was not found
		if err == mongo.ErrNoDocuments {
			return false, errors.New("ErrNoDocuments")
		}
		return false, err
	}

	books := strings.Split(cart.Books, ",")
	fmt.Printf("Cart books before: %v\n", cart.Books)
	bookExist := false
	for _, b := range books {
		if b == bookTitle {
			bookExist = true
		}
	}

	books = remove(books, bookTitle)
	cart.Books = strings.Join(books, ",")
	fmt.Printf("Cart books after: %v\n", cart.Books)

	filter := bson.M{"userEmail": email}
	res, err := coll.ReplaceOne(context.TODO(), filter, cart)
	if err != nil {
		return false, err
	}
	fmt.Printf("Update document with _id: %v\n", res)
	return bookExist, nil
}

func remove(s []string, r string) []string {
	for i, v := range s {
		if v == r {
			return append(s[:i], s[i+1:]...)
		}
	}
	return s
}
