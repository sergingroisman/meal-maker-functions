package handlers

import "go.mongodb.org/mongo-driver/bson/primitive"

type Accompaniment struct {
	ID       primitive.ObjectID `bson:"_id,omitempty" json:"_id,omitempty"`
	Title    string             `bson:"title" json:"title"`
	Category string             `bson:"category" json:"category"`
}

type Dish struct {
	ID             primitive.ObjectID `bson:"_id,omitempty" json:"_id,omitempty"`
	Title          string             `bson:"title" json:"title"`
	Price          float64            `bson:"price" json:"price"`
	Discount       float64            `bson:"discount" json:"discount"`
	Description    string             `bson:"description" json:"description"`
	Observation    string             `bson:"observation" json:"observation"`
	Accompaniments []Accompaniment    `bson:"accompaniments" json:"accompaniments"`
}

type Menu struct {
	ID               primitive.ObjectID `bson:"_id,omitempty" json:"_id,omitempty"`
	Name             string             `bson:"name" json:"name"`
	SmallDescription string             `bson:"smallDescription" json:"smallDescription"`
	Dishes           []Dish             `bson:"dishes" json:"dishes"`
}
