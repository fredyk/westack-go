package boot

import (
	"github.com/fredyk/westack-go/examples/basic-auth-example/common/models"
	"github.com/fredyk/westack-go/westack"
	"log"
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
		log.Fatalln("Could not create note", err)
	} else {
		var typedNote models.Note
		err := note.Transform(&typedNote)
		if err != nil {
			panic(err)
		}

		//id := typedNote.Id
		log.Println("Created note", typedNote, "for user", typedUser)

		// List user notes
		notes, _ := noteModel.FindMany(&map[string]interface{}{"where": map[string]interface{}{"userId": typedUser.Id}})

		log.Println("User notes:", len(notes))
	}

}
