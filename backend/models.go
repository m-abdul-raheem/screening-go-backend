package main

type User struct {
	Email    string `bson:"email,omitempty" validate:"required" valid:"email,required"`
	Password string `bson:"password,omitempty" validate:"required" valid:"required"`
	IsAdmin  bool   `bson:"isAdmin,omitempty"`
}

// Every book has a title, year published, author name, price in USD, and category.
// Each book can (and must) be assigned to a category. Every book also has a number of copies in stock.
// Books that are sold out should not be visible in the listing, and it should not be possible to buy them.
// Stock can be specified when a book is created, but can’t be edited later.
type Book struct {
	Title     string `bson:"title,omitempty" valid:"required"`
	Published string `bson:"published,omitempty" valid:"required"`
	Author    string `bson:"author,omitempty" valid:"required"`
	Category  string `bson:"category,omitempty" valid:"required"`
	Price     int    `bson:"price,omitempty" valid:"required"`
	Copies    int    `bson:"copies,omitempty" valid:"optional"`
}

// Every category has a name and books assigned to it.
// Categories hierarchy is flat - meaning that they can’t be nested.
type Category struct {
	CategoryName string `bson:"categoryName,omitempty" valid:"required"`
	Books        string `bson:"books,omitempty" valid:"optional"`
}

// Authenticated users should be able to add books to their cart.
// For simplicity, let’s assume that users can buy multiple books, but only one copy of each (so quantity is not necessary).
// There should be an endpoint that completes checkout and “buys” books currently in the cart.
// Please note that for simplicity, this endpoint should not take in any credit card details.
// It should simply pretend that it received them and can assume that a payment was made successfully,
// and should simply clear the cart and reduce the available quantity of books bought.
type Cart struct {
	UserEmail string `bson:"userEmail,omitempty" valid:"required"`
	Books     string `bson:"books,omitempty" valid:"required"`
}
