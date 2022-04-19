# westack-go

### Introduction
westack-go is a strongly opinionated framework which allows you to quickly setup a REST API server in a few minutes.

Just define your models in `json` format and westack-go will setup and expose all basic [CRUD](https://es.wikipedia.org/wiki/CRUD) methods for you 

### Technologies
westack-go uses technologies like [gofiber](https://github.com/gofiber/fiber) and [casbin](github.com/casbin/casbin) for REST and authentication

### Databases
It is only compatible with [mongo](go.mongodb.org/mongo-driver).

### Authentication
Define [RBAC](https://casbin.org/docs/en/rbac) policies in your `json` models to restrict access to data.

### Getting started

Create your file structure like this:
```
- common
  |- models
  |  |- user.json
  |  |- note.json
  
- server
  |- main.go
  |- datasource.json
  |- model-config.json
```

`user.json`
```json
{
  "name": "user",
  "base": "User",
  "public": true,
  "hidden": [
    "password"
  ],
  "properties": {
    "email": {
      "type": "string",
      "required": true
    },
    "password": {
      "type": "string",
      "required": true
    }
  },
  "relations": {
    "notes": {
      "type": "hasMany",
      "model": "note"
    }
  }
}
```

`note.json`
```json
{
  "name": "note",
  "base": "PersistedModel",
  "public": true,
  "properties": {
    "title": {
      "type": "string",
      "required": true
    },
    "body": {
      "type": "string",
      "required": true
    }
  },
  "relations": {
    "user": {
      "type": "belongsTo",
      "model": "user"
    }
  },
  "casbin": {
    "policies": [
      "$everyone,*,*,deny",
      "$authenticated,*,create,allow",
      "$owner,*,*,allow"
    ]
  }
}
```

`datasources.json`
```json
{
  "db": {
    "name": "db",
    "host": "localhost",
    "port": 27017,
    "database": "example_db",
    "password": "",
    "username": "",
    "connector": "mongodb"
  }
}
```

`model-config.json`
```json
{
  "user": {
    "dataSource": "db"
  },
  "note": {
    "dataSource": "db"
  }
}
```

`main.go`
```go
package main

import (
	"github.com/fredyk/westack-go/westack"
	"github.com/gofiber/fiber/v2"
)

func main() {

	app := westack.New(westack.WeStackOptions{
		Debug:       false,
		RestApiRoot: "/api/v1",
		Port:        8023,
	})

	app.Boot(func(app * westack.WeStack) {

		// Setup your custom routes here
		app.Server.Get("/status", func(c * fiber.Ctx) error {
			return c.JSON(fiber.Map{"status": "OK"})
		})

	})

	app.Start(fmt.Sprintf(":%v", app.Port))

}
```

Test it:

1. Create a user
```shell
$ curl -X POST http://localhost:8023/api/v1/users -d '{"email":"exampleuser@example.com","password":"1234"}'
```

2. Login
```shell
$ curl -X POST http://localhost:8023/api/v1/users/login -d '{"email":"exampleuser@example.com","password":"1234"}'

Response body: {"id":"eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJjcmVhdGVkIjoxNjQ3MjUzMDczMTQ0LCJyb2xlcyI6WyJVU0VSIl0sInR0bCI6MTIwOTYwMDAwMCwidXNlcklkIjoiNjIyZjE2NDMzNzdjYTNmMWEzOTI0MWY0In0.sbl7QA2--X7MiPZ4DLRL2f5_z08VD5quItBDl2ybmGk","userId":"622f1643377ca3f1a39241f4"}
```

3. Find user data
```shell
$ curl http://localhost:8023/api/v1/users/me -H 'Authorization: Bearer eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJjcmVhdGVkIjoxNjQ3MjUzMDczMTQ0LCJyb2xlcyI6WyJVU0VSIl0sInR0bCI6MTIwOTYwMDAwMCwidXNlcklkIjoiNjIyZjE2NDMzNzdjYTNmMWEzOTI0MWY0In0.sbl7QA2--X7MiPZ4DLRL2f5_z08VD5quItBDl2ybmGk'
 
Response body: {"email":"exampleuser@example.com","id":"622f1643377ca3f1a39241f4"}
```

4. Create a note for the user
```shell
$ curl -X POST http://localhost:8023/api/v1/notes -H 'Authorization: Bearer eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJjcmVhdGVkIjoxNjQ3MjUzMDczMTQ0LCJyb2xlcyI6WyJVU0VSIl0sInR0bCI6MTIwOTYwMDAwMCwidXNlcklkIjoiNjIyZjE2NDMzNzdjYTNmMWEzOTI0MWY0In0.sbl7QA2--X7MiPZ4DLRL2f5_z08VD5quItBDl2ybmGk' -d '{"title":"Note 1","body":"This is my first note","userId":"622f1643377ca3f1a39241f4"}'
```

5. Find again the user, now with their notes
```shell
$ curl 'http://localhost:8023/api/v1/users/me?filter=%7B"include":%5B%7B"relation":"notes"%7D%5D%7D' -H 'Authorization: Bearer eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJjcmVhdGVkIjoxNjQ3MjUzMDczMTQ0LCJyb2xlcyI6WyJVU0VSIl0sInR0bCI6MTIwOTYwMDAwMCwidXNlcklkIjoiNjIyZjE2NDMzNzdjYTNmMWEzOTI0MWY0In0.sbl7QA2--X7MiPZ4DLRL2f5_z08VD5quItBDl2ybmGk'

Response body: {"email":"exampleuser@example.com","id":"622f1643377ca3f1a39241f4","notes":[{"title":"Note 1","body":"This is my first note","userId":"622f1643377ca3f1a39241f4","id":"622f1643377ca3f1a39241f5"}]}
```

6. Find the single note
```shell
$ curl http://localhost:8023/api/v1/notes/622f1643377ca3f1a39241f5 -H 'Authorization: Bearer eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJjcmVhdGVkIjoxNjUwNDA2ODEzNDY3LCJyb2xlcyI6WyJVU0VSIl0sInR0bCI6MTIwOTYwMDAwMCwidXNlcklkIjoiNjI1ZjM1OTE0NzU5YWJiOGZhMmE1YzljIn0.hWeMlZrhTFAac4LXTSiSIQ7uy7VhAlg1L9DKG3QPTpg'

Response body: {"title":"Note 1","body":"This is my first note","userId":"622f1643377ca3f1a39241f4","id":"622f1643377ca3f1a39241f5"}
```

7. Update the note
```shell
$ curl -X PATCH http://localhost:8023/api/v1/notes/622f1643377ca3f1a39241f5 -H 'Authorization: Bearer eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJjcmVhdGVkIjoxNjUwNDA2ODEzNDY3LCJyb2xlcyI6WyJVU0VSIl0sInR0bCI6MTIwOTYwMDAwMCwidXNlcklkIjoiNjI1ZjM1OTE0NzU5YWJiOGZhMmE1YzljIn0.hWeMlZrhTFAac4LXTSiSIQ7uy7VhAlg1L9DKG3QPTpg' -d '{"body":"I modified the note body"}'

Response body: {"title":"Note 1","body":"I modified the note body","userId":"622f1643377ca3f1a39241f4","id":"622f1643377ca3f1a39241f5"}
```

### Contribute

Write to [westack.team@gmail.com](mailto://westack.team@gmail.com) if you want to contribute to the project: D

You can also create as many pull requests as you want