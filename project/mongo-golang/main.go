package main

import (
	"fmt"
	"github.com/julienschmidt/httprouter"
	"gopkg.in/mgo.v2"
	"net/http"
	"log"
	"github.com/abhira860/mongo-golang/controller"
)

func main() {
	r := httprouter.New()
	uc := controllers.NewUserController(getSession())
	r.GET("/user/:id", uc.GetUser)
	r.POST("/user", uc.CreateUser)
	r.DELETE("/user/:id", uc.DeleteUser)
	fmt.Println("Server running at port :9000")
	err := http.ListenAndServe(":9000", r)
	if err != nil {
		log.Fatal(err)
	}
}

func getSession() *mgo.Session {
	s, err := mgo.Dial("localhost:27017")
	if err != nil {
		panic(err)
	}
	return s
}