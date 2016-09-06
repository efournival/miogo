package main

import (
	"log"

	"golang.org/x/crypto/bcrypt"

	"gopkg.in/mgo.v2"
	"gopkg.in/mgo.v2/bson"
)

var db *mgo.Database

func InitDB(host, adminEmail, adminPassword string) {
	session, err := mgo.Dial(host)

	if err != nil {
		log.Fatalf("Cannot connect to MongoDB: %s\n", err)
	}

	db = session.DB("miogo")

	selector := bson.M{"path": "/"}

	// Init DB if it's the first time Miogo is launched
	if count, err := session.DB("miogo").C("folders").Find(selector).Count(); count == 0 && err == nil {
		db.C("folders").Insert(selector)
	}

	if count, err := session.DB("miogo").C("users").Find(bson.M{"is_admin": true}).Count(); count == 0 && err == nil {
		hashedAdminPassword, _ := bcrypt.GenerateFromPassword([]byte(adminPassword), bcrypt.DefaultCost)
		db.C("users").Insert(bson.M{"email": adminEmail, "password": string(hashedAdminPassword), "is_admin": true})
	}
}
