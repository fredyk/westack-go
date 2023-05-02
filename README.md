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

### Installing westack

```shell
go install github.com/fredyk/westack-go@v1.5.46
```

### Initialize a new project
```shell
mkdir my-new-project
cd my-new-project
westack-go init .
```

### Create a new model
```shell
# Usage: westack-go model add <model_name> <datasource_name>
#   <datasource_name> defaults to "db" when you run `westack-go init .`
 
westack-go model add Note db
```

### Getting started

#### (Optional) Customize your models and datasources

<details>
  <summary>User.json</summary>

```json
{
  "name": "User",
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

</details>

<details>
  <summary>Role.json</summary>

```json
{
  "name": "Note",
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
      "model": "User"
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

</details>

<details>
  <summary>datasources.json</summary>

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

</details>

<details>
  <summary>model-config.json</summary>

```json
{
  "User": {
    "dataSource": "db"
  },
  "Note": {
    "dataSource": "db"
  }
}
```

</details>


### Run

```shell
westack-go server start
```

### Test it:

1. Create a user
```shell
$ curl -X POST http://localhost:8023/api/v1/users -H 'Content-Type: application/json' -d '{"email":"exampleuser@example.com","password":"1234"}'
```

2. Login
```shell
$ curl -X POST http://localhost:8023/api/v1/users/login -H 'Content-Type: application/json' -d '{"email":"exampleuser@example.com","password":"1234"}'

Response body: {"id":"eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJjcmVhdGVkIjoxNjQ3MjUzMDczMTQ0LCJyb2xlcyI6WyJVU0VSIl0sInR0bCI6MTIwOTYwMDAwMCwidXNlcklkIjoiNjIyZjE2NDMzNzdjYTNmMWEzOTI0MWY0In0.sbl7QA2--X7MiPZ4DLRL2f5_z08VD5quItBDl2ybmGk","userId":"622f1643377ca3f1a39241f4"}
```

3. Find user data
```shell
$ curl http://localhost:8023/api/v1/users/me -H 'Authorization: Bearer eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJjcmVhdGVkIjoxNjQ3MjUzMDczMTQ0LCJyb2xlcyI6WyJVU0VSIl0sInR0bCI6MTIwOTYwMDAwMCwidXNlcklkIjoiNjIyZjE2NDMzNzdjYTNmMWEzOTI0MWY0In0.sbl7QA2--X7MiPZ4DLRL2f5_z08VD5quItBDl2ybmGk'
 
Response body: {"email":"exampleuser@example.com","id":"622f1643377ca3f1a39241f4"}
```

4. Create a note for the user
```shell
$ curl -X POST http://localhost:8023/api/v1/notes -H 'Content-Type: application/json' -H 'Authorization: Bearer eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJjcmVhdGVkIjoxNjQ3MjUzMDczMTQ0LCJyb2xlcyI6WyJVU0VSIl0sInR0bCI6MTIwOTYwMDAwMCwidXNlcklkIjoiNjIyZjE2NDMzNzdjYTNmMWEzOTI0MWY0In0.sbl7QA2--X7MiPZ4DLRL2f5_z08VD5quItBDl2ybmGk' -d '{"title":"Note 1","body":"This is my first note","userId":"622f1643377ca3f1a39241f4"}'
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
$ curl -X PATCH http://localhost:8023/api/v1/notes/622f1643377ca3f1a39241f5 -H 'Content-Type: application/json' -H 'Authorization: Bearer eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJjcmVhdGVkIjoxNjUwNDA2ODEzNDY3LCJyb2xlcyI6WyJVU0VSIl0sInR0bCI6MTIwOTYwMDAwMCwidXNlcklkIjoiNjI1ZjM1OTE0NzU5YWJiOGZhMmE1YzljIn0.hWeMlZrhTFAac4LXTSiSIQ7uy7VhAlg1L9DKG3QPTpg' -d '{"body":"I modified the note body"}'

Response body: {"title":"Note 1","body":"I modified the note body","userId":"622f1643377ca3f1a39241f4","id":"622f1643377ca3f1a39241f5"}
```
### Change Log

* **v1.6.0**

    * Added parameter `strictSingleRelatedDocumentCheck` in config.json, defaults to `true`in new projects, and `false` in existing ones.
    * `"hasOne"` and `"belongsTo"` relations are now checked after fetching documents from Mongo. If `strictSingleRelatedDocumentCheck` is `true` and the relation returns more than 1 document, an error is thrown. Otherwise, only the first document is used and a warning is logged.
    * **Breaking changes**:
      * `model.Build()` requires now parameter `sameLevelCache *buildCache` to be passed in. Can be generated with `model.NewBuildCache()`
      * `model.Build()` returns now `error` as second value, in addition to the instance. So it is now `func (loadedModel *Model) Build(data wst.M, sameLevelCache *buildCache, baseContext *EventContext) (Instance, error)`

* **v1.5.48**

    * **Breaking change**: environment variables `WST_ADMIN_USERNAME` and `WST_ADMIN_PWD` are required to start the server

### Contribute

Write to [westack.team@gmail.com](mailto://westack.team@gmail.com) if you want to contribute to the project: D

You can also create as many pull requests as you want
