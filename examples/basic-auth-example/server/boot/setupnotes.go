package boot

import (
	"github.com/fredyk/westack-go/westack"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"log"
)

func SetupNotes(app *westack.WeStack) {

	userModel := app.FindModel("user")
	noteModel := app.FindModel("note")

	user, _ := userModel.FindOne(&map[string]interface{}{"where": map[string]interface{}{"email": "test@example.com"}})

	if user == nil {
		user, _ = userModel.Create(map[string]interface{}{"email": "test@example.com", "password": "1234"})
	}

	if note, err := noteModel.Create(map[string]interface{}{"title": "A note", "body": "this is a note", "userId": user.Id}); err != nil {
		log.Fatalln("Could not create note", err)
	} else {
		id := note.Id.(primitive.ObjectID)
		log.Println("Created note", id.Hex(), "for user", note.ToJSON()["userId"].(primitive.ObjectID).Hex())

		notes, _ := noteModel.FindMany(&map[string]interface{}{"where": map[string]interface{}{"userId": user.Id}})

		log.Println("User notes:", len(notes))
	}

}
