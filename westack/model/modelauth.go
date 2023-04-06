package model

import (
	"fmt"
	"log"
	"strings"
	"sync"
	"time"

	"github.com/gofiber/fiber/v2"
)

var AuthMutex = sync.Mutex{}

func (loadedModel *Model) EnforceEx(token *BearerToken, objId string, action string, eventContext *EventContext) (error, bool) {

	if token != nil && token.User != nil && token.User.System == true {
		return nil, true
	}

	if token == nil {
		log.Printf("WARNING: Trying to enforce without token at %v.%v\n", loadedModel.Name, action)
	}

	var bearerUserIdSt string
	var targetObjId string

	var locked bool

	if token == nil || token.User == nil {
		bearerUserIdSt = "_EVERYONE_"
		targetObjId = "*"
		if result, isPresent := loadedModel.authCache[bearerUserIdSt][targetObjId][action]; isPresent {
			if loadedModel.App.Debug || !result {
				log.Printf("DEBUG: Cache hit for %v.%v ---> %v\n", loadedModel.Name, action, result)
			}
			return nil, result
		}

	} else {

		bearerUserIdSt = fmt.Sprintf("%v", token.User.Id)
		targetObjId = objId

		var created int64
		var ttl int64
		// try to cast
		if v, ok := token.Claims["created"].(int64); ok {
			created = v
		} else {
			created = int64(token.Claims["created"].(float64))
		}
		if v, ok := token.Claims["ttl"].(int64); ok {
			ttl = v
		} else {
			ttl = int64(token.Claims["ttl"].(float64))
		}
		expiresAtTimestamp := created + ttl
		if time.Now().Unix() > expiresAtTimestamp {
			if loadedModel.App.Debug {
				fmt.Println("Token expired for user", bearerUserIdSt)
			}
			return fiber.ErrUnauthorized, false
		}

		if result, isPresent := loadedModel.authCache[bearerUserIdSt][targetObjId][action]; isPresent {
			if loadedModel.App.Debug || !result {
				log.Printf("DEBUG: Cache hit for %v.%v ---> %v\n", loadedModel.Name, action, result)
			}
			return nil, result
		}

		AuthMutex.Lock()
		defer AuthMutex.Unlock()
		locked = true

		_, err := loadedModel.Enforcer.AddRoleForUser(bearerUserIdSt, "_EVERYONE_")
		if err != nil {
			return err, false
		}
		_, err = loadedModel.Enforcer.AddRoleForUser(bearerUserIdSt, "_AUTHENTICATED_")
		if err != nil {
			return err, false
		}
		for _, r := range token.Roles {
			_, err := loadedModel.Enforcer.AddRoleForUser(bearerUserIdSt, r.Name)
			if err != nil {
				return err, false
			}
		}
		err = loadedModel.Enforcer.SavePolicy()
		if err != nil {
			return err, false
		}

	}

	if !locked {
		AuthMutex.Lock()
		defer AuthMutex.Unlock()
	}

	allow, exp, err := loadedModel.Enforcer.EnforceEx(bearerUserIdSt, targetObjId, action)

	if loadedModel.App.Debug || !allow {
		if len(exp) > 0 {
			log.Println("Explain", exp)
		}
		fmt.Printf("DEBUG: EnforceEx for %v.%v (subj=%v,obj=%v) ---> %v\n", loadedModel.Name, action, bearerUserIdSt, targetObjId, allow)
		if eventContext.Remote != nil && eventContext.Remote.Name != "" {
			fmt.Printf("DEBUG: ... at remote method %v %v%v\n", strings.ToUpper(eventContext.Remote.Http.Verb), loadedModel.BaseUrl, eventContext.Remote.Http.Path)
		}
	}
	if err != nil {
		updateAuthCache(loadedModel, bearerUserIdSt, targetObjId, action, false)
		return err, false
	}
	if allow {
		updateAuthCache(loadedModel, bearerUserIdSt, targetObjId, action, true)
		return nil, true
	}
	return fiber.ErrUnauthorized, false
}

//go:noinline
func updateAuthCache(loadedModel *Model, bearerUserIdSt string, targetObjId string, action string, allow bool) {
	if loadedModel.authCache[bearerUserIdSt] == nil {
		addSubjectAuthCacheEntry(loadedModel, bearerUserIdSt)
	}
	if loadedModel.authCache[bearerUserIdSt][targetObjId] == nil {
		addObjectAuthCacheEntry(loadedModel, bearerUserIdSt, targetObjId)
	}
	addActionAuthCacheEntry(loadedModel, bearerUserIdSt, targetObjId, action, allow)
}

//go:noinline
func addSubjectAuthCacheEntry(loadedModel *Model, bearerUserIdSt string) {
	loadedModel.authCache[bearerUserIdSt] = make(map[string]map[string]bool)
}

//go:noinline
func addObjectAuthCacheEntry(loadedModel *Model, bearerUserIdSt string, targetObjId string) {
	loadedModel.authCache[bearerUserIdSt][targetObjId] = make(map[string]bool)
}

//go:noinline
func addActionAuthCacheEntry(loadedModel *Model, bearerUserIdSt string, targetObjId string, action string, allow bool) {
	loadedModel.authCache[bearerUserIdSt][targetObjId][action] = allow
}
