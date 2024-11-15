package models

import "go.mongodb.org/mongo-driver/bson/primitive"

type Account struct {
	Id    primitive.ObjectID `json:"id"`
	Email string             `json:"email"`
}
