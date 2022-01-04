package boot

import (
	"fmt"
	"github.com/fredyk/westack-go/examples/basic-auth-example/common/models"
	"github.com/fredyk/westack-go/westack"
	"go.mongodb.org/mongo-driver/bson"
	"log"
	"time"
)

func SetupNotes(app *westack.WeStack) {

	// Declare models
	userModel := app.FindModel("user")
	noteModel := app.FindModel("note")

	// Check if user exists
	user, _ := userModel.FindOne(&map[string]interface{}{"where": map[string]interface{}{"email": "test@example.com"}})
	if user == nil {
		user, _ = userModel.Create(map[string]interface{}{"email": "test@example.com", "password": "1234"})
	}
	var typedUser models.User
	err := user.Transform(&typedUser)
	if err != nil {
		panic(err)
	}

	// Create a note for the user
	if note, err := noteModel.Create(map[string]interface{}{"title": "A note", "body": "this is a note", "userId": typedUser.Id}); err != nil {
		panic(fmt.Sprintf("Could not create note %v", err))
	} else {
		var typedNote models.Note
		err := note.Transform(&typedNote)
		if err != nil {
			panic(err)
		}

		//id := typedNote.Id
		log.Println("Created note", typedNote, "for user", typedUser)

		// Update the note
		updated, err := note.UpdateAttributes(&bson.M{"date": time.Now()})
		if err != nil {
			panic(nil)
		}
		log.Println("Updated note ", updated.ToJSON())

		// List user notes
		notes, _ := noteModel.FindMany(&map[string]interface{}{"where": map[string]interface{}{"userId": typedUser.Id.Hex()}})

		log.Println("User notes:", len(notes))

		// Delete the note
		deletedCount, err := noteModel.DeleteById(note.Id)
		if err != nil {
			panic(nil)
		}
		if deletedCount != 1 {
			panic(fmt.Sprintf("Note was not deleted: count=%v", deletedCount))
		}
		log.Println("Deleted notes: ", deletedCount)

		// Again list user notes
		notes, _ = noteModel.FindMany(&map[string]interface{}{"where": map[string]interface{}{"userId": typedUser.Id.Hex()}})

		log.Println("User notes:", len(notes))

	}

}
