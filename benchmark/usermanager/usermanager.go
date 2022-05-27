package usermanager

import (
	"encoding/json"
	"log"
	"os"
)

const (
	USERS_DATA_FILENAME = "users.json"
)

var (
	users map[string]User
)

type User struct {
	Userkey   string
	LDAP      string
	Team      string
	Region    string
	SubRegion string
	Role      string
}

type UsersBlob struct {
	Users []User `json:"users"`
}

func init() {
	initUsers(USERS_DATA_FILENAME)
}

func initUsers(filename string) map[string]User {
	if users != nil {
		return users
	}

	jsonFromFile, err := os.ReadFile(filename)
	if err != nil {
		log.Fatal(err)
	}

	var jsonData UsersBlob
	err = json.Unmarshal(jsonFromFile, &jsonData)
	if err != nil {
		log.Fatal(err)
	}

	users = make(map[string]User)
	for _, user := range jsonData.Users {
		users[user.Userkey] = user
	}

	return users
}

func GetNumOfUsers() int {
	return len(users)
}

func GetLDAPByUserkey(userkey string) string {
	if user, ok := users[userkey]; ok {
		return user.LDAP
	}

	return ""
}

func ExistUser(userkey string) bool {
	_, exist := users[userkey]

	return exist
}
