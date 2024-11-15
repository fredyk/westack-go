package boot

import (
	"fmt"
	"log"
	"time"

	"github.com/fredyk/westack-go/examples/basic-auth-example/common/models"
	"github.com/fredyk/westack-go/v2"
	wst "github.com/fredyk/westack-go/v2/common"
)

func SetupNotes(app *westack.WeStack) {

	// Declare models
	noteModel, err := app.FindModel("note")
	if err != nil {
		log.Printf("ERROR: SetupNotes() --> %v\n", err)
		return
	}
	userModel, _ := app.FindModel("user")

	// Check if user exists
	user, err := userModel.FindOne(&wst.Filter{Where: &wst.Where{"email": "test@example.com"}}, nil)
	if err != nil {
		panic(err)
	}
	if user == nil {
		user, err = userModel.Create(wst.M{"email": "test@example.com", "password": "1234"}, nil)
		if err != nil {
			panic(err)
		}
	}
	var typedUser models.User
	err = user.(*model.StatefulInstance).Transform(&typedUser)
	if err != nil {
		panic(err)
	}

	// Create a note for the user
	if note, err := noteModel.Create(wst.M{"title": "A note", "body": "this is a note", "userId": typedUser.Id}, nil); err != nil {
		panic(fmt.Sprintf("Could not create note %v", err))
	} else {
		var typedNote models.Note
		err := note.(*model.StatefulInstance).Transform(&typedNote)
		if err != nil {
			panic(err)
		}

		log.Println("Created note", typedNote, "for user", typedUser)

		// Update the note
		updated, err := note.UpdateAttributes(&wst.M{"date": time.Now()}, nil)
		if err != nil {
			panic(nil)
		}
		log.Println("Updated note ", updated.ToJSON())

		// List user notes
		notes, _ := noteModel.FindMany(&wst.Filter{Where: &wst.Where{"userId": typedUser.Id.Hex()}}, nil)

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
		notes, _ = noteModel.FindMany(&wst.Filter{Where: &wst.Where{"userId": typedUser.Id.Hex()}}, nil)

		log.Println("User notes:", len(notes))

	}

}
