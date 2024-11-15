package boot

import (
	"fmt"
	"github.com/fredyk/westack-go/v2/model"
	"github.com/fredyk/westack-go/westack"
	"log"
	"time"

	"github.com/fredyk/westack-go/examples/basic-auth-example/common/models"
	wst "github.com/fredyk/westack-go/v2/common"
)

func SetupNotes(app *westack.WeStack) {

	// Declare models
	noteModel, err := app.FindModel("note")
	if err != nil {
		log.Printf("ERROR: SetupNotes() --> %v\n", err)
		return
	}
	accountModel, _ := app.FindModel("account")

	// Check if user exists
	user, err := accountModel.FindOne(&wst.Filter{Where: &wst.Where{"email": "test@example.com"}}, nil)
	if err != nil {
		panic(err)
	}
	if user == nil {
		user, err = accountModel.Create(wst.M{"email": "test@example.com", "password": "1234"}, nil)
		if err != nil {
			panic(err)
		}
	}
	var typedAccount models.Account
	err = user.(*model.StatefulInstance).Transform(&typedAccount)
	if err != nil {
		panic(err)
	}

	// Create a note for the user
	if note, err := noteModel.Create(wst.M{"title": "A note", "body": "this is a note", "accountId": typedAccount.Id}, nil); err != nil {
		panic(fmt.Sprintf("Could not create note %v", err))
	} else {
		var typedNote models.Note
		err := note.(*model.StatefulInstance).Transform(&typedNote)
		if err != nil {
			panic(err)
		}

		log.Println("Created note", typedNote, "for user", typedAccount)

		// Update the note
		updated, err := note.UpdateAttributes(&wst.M{"date": time.Now()}, nil)
		if err != nil {
			panic(nil)
		}
		log.Println("Updated note ", updated.ToJSON())

		// List user notes
		notes, _ := noteModel.FindMany(&wst.Filter{Where: &wst.Where{"accountId": typedAccount.Id.Hex()}}, nil)

		log.Println("Account notes:", len(notes))

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
		notes, _ = noteModel.FindMany(&wst.Filter{Where: &wst.Where{"accountId": typedAccount.Id.Hex()}}, nil)

		log.Println("Account notes:", len(notes))

	}

}
