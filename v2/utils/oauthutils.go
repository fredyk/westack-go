package utils

import "sync"

var lock = &sync.Mutex{}
var oauthStatesByCookies = make(map[string]string)

func RegisterOauthState(cookie, state string) {
	lock.Lock()
	defer lock.Unlock()
	oauthStatesByCookies[cookie] = state
}

func GetOauthState(cookie string) (string, bool) {
	lock.Lock()
	defer lock.Unlock()
	state, ok := oauthStatesByCookies[cookie]
	return state, ok
}
