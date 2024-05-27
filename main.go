package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"time"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

type Server struct {
	client *mongo.Client
}

func NewServer(c *mongo.Client) *Server {
	return &Server{client: c}
}

func (s *Server) handleGetAllFacts(w http.ResponseWriter, r *http.Request) {
	coll := s.client.Database("catfact").Collection("facts")

	query := bson.M{}
	cursor, err := coll.Find(context.TODO(), query)
	if err != nil {
		log.Fatal(err)
	}

	results := []bson.M{}
	if err = cursor.All(context.TODO(), &results); err != nil {
		log.Fatal(err)
	}

	w.WriteHeader(http.StatusOK)
	w.Header().Add("Content-Type", "application/json")
	json.NewEncoder(w).Encode(results)
}

type CatFactWorker struct {
	client *mongo.Client
}

func NewCatFactWorker(c *mongo.Client) *CatFactWorker {
	return &CatFactWorker{client: c}
}

func (cw *CatFactWorker) start() error {
	coll := cw.client.Database("catfact").Collection("facts")
	ticker := time.NewTicker(2 * time.Second)

	for {
		resp, err := http.Get("https://catfact.ninja/fact")
		if err != nil {
			return err
		}

		var fact bson.M
		if err := json.NewDecoder(resp.Body).Decode(&fact); err != nil {
			return err
		}

		_, err = coll.InsertOne(context.TODO(), fact)
		if err != nil {
			return err
		}

		<-ticker.C
	}
}

func main() {
	client, err := mongo.Connect(
		context.TODO(),
		options.Client().ApplyURI("mongodb://localhost:27017"),
	)

	if err != nil {
		log.Fatalln("mongo connect isn't established")
	}

	worker := NewCatFactWorker(client)
	go worker.start()

	server := NewServer(client)
	http.HandleFunc("/facts", server.handleGetAllFacts)
	http.ListenAndServe(":8080", nil)

	fmt.Println(client)
}
