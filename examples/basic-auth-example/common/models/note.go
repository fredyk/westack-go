package models

import "go.mongodb.org/mongo-driver/bson/primitive"

type Note struct {
	Id        primitive.ObjectID `json:"id"`
	AccountId primitive.ObjectID `json:"accountId"`
}
