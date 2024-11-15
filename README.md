# westack-go Documentation

## Introduction

westack-go is a modular Go framework designed to simplify the process of building scalable and extensible APIs. It provides utilities for managing data models, routing, and middleware, along with powerful integrations for Swagger documentation, CLI tools, and more.

### Key Features

- **Model-Driven Architecture**: Define data models with ease and generate APIs automatically.
- **Extensible Datasources**: Support for in-memory and MongoDB datasources out of the box.
- **Role Management**: Built-in role-based access control (RBAC) support.
- **CLI Utilities**: Command-line tools for common development tasks.

### Prerequisites

- Go (version 1.21 or higher)
- MongoDB (if using MongoDB as a datasource)
- Basic knowledge of Go programming

## Quick Start

Follow these steps to quickly set up and run a simple API:

1. **Create the project directory**:

   Create a directory for your project and initialize a Go module:

   ```bash
   mkdir myproject && cd myproject
   go mod init myproject
   ```

   This sets up the directory and initializes Go module management for your project.

2. **Install the westack-go CLI**:

   Install the CLI tool to simplify project setup and management:

   ```bash
   go install github.com/fredyk/westack-go@latest
   ```

   The CLI provides commands like `init` and `model add` for rapid development.

3. **Initialize the project**:

   Use the CLI to set up the project structure:

   ```bash
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
  <summary>Account.json</summary>

```json
{
  "name": "Account",
  "base": "Account",
  "public": true,
  "hidden": [
    "password"
  ],
  "properties": {
    "title": {
      "type": "string",
      "required": true
    },
    "content": {
      "type": "string"
    }
  },
  "casbin": {
    "policies": [
      "$authenticated,*,*,allow",
      "$everyone,*,read,allow",
      "$owner,*,__get__footer,allow"
    ]
  }
}
```

By default, `westack-go` generates the following standard CRUD routes for the `Note` model:

- `POST /notes`: Create a new note
- `GET /notes`: Retrieve all notes
- `GET /notes/{id}`: Retrieve a specific note by ID
- `PATCH /notes/{id}`: Partially update fields of a specific note
- `DELETE /notes/{id}`: Delete a specific note by ID

#### Relating Models

You can relate models using the `relations` property in the JSON definition. For example, to relate `Footer` to `Note` (and define that `Note` has one `Footer`):

Create or update `models/footer.json`:

```json
{
  "name": "Footer",
  "base": "PersistedModel",
  "properties": {
    "content": {
      "type": "string",
      "required": true
    }
  },
  "relations": {
    "note": {
      "type": "belongsTo",
      "model": "Note",
      "foreignKey": "noteId"
    }
  },
  "casbin": {
    "policies": [
      "$authenticated,*,*,allow",
      "$everyone,*,read,allow",
      "$owner,*,__get__note,allow"
    ]
  }
}
```

Create or update `models/note.json`:

```json
{
  "name": "Note",
  "base": "PersistedModel",
  "properties": {
    "title": {
      "type": "string",
      "required": true
    },
    "content": {
      "type": "string"
    }
  },
  "relations": {
    "account": {
      "type": "belongsTo",
      "model": "Account"
    }
  },
  "casbin": {
    "policies": [
      "$authenticated,*,*,allow",
      "$everyone,*,read,allow",
      "$owner,*,__get__footer,allow"
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
  "Account": {
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

1. Create an account
```shell
$ curl -X POST http://localhost:8023/api/v1/accounts -H 'Content-Type: application/json' -d '{"email":"exampleuser@example.com","password":"1234"}'
```

2. Login
```shell
$ curl -X POST http://localhost:8023/api/v1/accounts/login -H 'Content-Type: application/json' -d '{"email":"exampleuser@example.com","password":"1234"}'

Response body: {"id":"eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJjcmVhdGVkIjoxNjQ3MjUzMDczMTQ0LCJyb2xlcyI6WyJVU0VSIl0sInR0bCI6MTIwOTYwMDAwMCwidXNlcklkIjoiNjIyZjE2NDMzNzdjYTNmMWEzOTI0MWY0In0.sbl7QA2--X7MiPZ4DLRL2f5_z08VD5quItBDl2ybmGk","accountId":"622f1643377ca3f1a39241f4"}
```

3. Find account data
```shell
$ curl http://localhost:8023/api/v1/accounts/me -H 'Authorization: Bearer eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJjcmVhdGVkIjoxNjQ3MjUzMDczMTQ0LCJyb2xlcyI6WyJVU0VSIl0sInR0bCI6MTIwOTYwMDAwMCwidXNlcklkIjoiNjIyZjE2NDMzNzdjYTNmMWEzOTI0MWY0In0.sbl7QA2--X7MiPZ4DLRL2f5_z08VD5quItBDl2ybmGk'
 
Response body: {"email":"exampleuser@example.com","id":"622f1643377ca3f1a39241f4"}
```

4. Create a note for the account
```shell
$ curl -X POST http://localhost:8023/api/v1/notes -H 'Content-Type: application/json' -H 'Authorization: Bearer eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJjcmVhdGVkIjoxNjQ3MjUzMDczMTQ0LCJyb2xlcyI6WyJVU0VSIl0sInR0bCI6MTIwOTYwMDAwMCwidXNlcklkIjoiNjIyZjE2NDMzNzdjYTNmMWEzOTI0MWY0In0.sbl7QA2--X7MiPZ4DLRL2f5_z08VD5quItBDl2ybmGk' -d '{"title":"Note 1","body":"This is my first note","accountId":"622f1643377ca3f1a39241f4"}'
```

5. Find again the account, now with their notes
```shell
$ curl 'http://localhost:8023/api/v1/accounts/me?filter=%7B"include":%5B%7B"relation":"notes"%7D%5D%7D' -H 'Authorization: Bearer eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJjcmVhdGVkIjoxNjQ3MjUzMDczMTQ0LCJyb2xlcyI6WyJVU0VSIl0sInR0bCI6MTIwOTYwMDAwMCwidXNlcklkIjoiNjIyZjE2NDMzNzdjYTNmMWEzOTI0MWY0In0.sbl7QA2--X7MiPZ4DLRL2f5_z08VD5quItBDl2ybmGk'

Response body: {"email":"exampleuser@example.com","id":"622f1643377ca3f1a39241f4","notes":[{"title":"Note 1","body":"This is my first note","accountId":"622f1643377ca3f1a39241f4","id":"622f1643377ca3f1a39241f5"}]}
```

6. Find the single note
```shell
$ curl http://localhost:8023/api/v1/notes/622f1643377ca3f1a39241f5 -H 'Authorization: Bearer eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJjcmVhdGVkIjoxNjUwNDA2ODEzNDY3LCJyb2xlcyI6WyJVU0VSIl0sInR0bCI6MTIwOTYwMDAwMCwidXNlcklkIjoiNjI1ZjM1OTE0NzU5YWJiOGZhMmE1YzljIn0.hWeMlZrhTFAac4LXTSiSIQ7uy7VhAlg1L9DKG3QPTpg'

Response body: {"title":"Note 1","body":"This is my first note","accountId":"622f1643377ca3f1a39241f4","id":"622f1643377ca3f1a39241f5"}
```

7. Update the note
```shell
$ curl -X PATCH http://localhost:8023/api/v1/notes/622f1643377ca3f1a39241f5 -H 'Content-Type: application/json' -H 'Authorization: Bearer eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJjcmVhdGVkIjoxNjUwNDA2ODEzNDY3LCJyb2xlcyI6WyJVU0VSIl0sInR0bCI6MTIwOTYwMDAwMCwidXNlcklkIjoiNjI1ZjM1OTE0NzU5YWJiOGZhMmE1YzljIn0.hWeMlZrhTFAac4LXTSiSIQ7uy7VhAlg1L9DKG3QPTpg' -d '{"body":"I modified the note body"}'

Response body: {"title":"Note 1","body":"I modified the note body","accountId":"622f1643377ca3f1a39241f4","id":"622f1643377ca3f1a39241f5"}
```
### Change Log

- **v1.6.14**

  - [#475 - Create tests for Datasource.DeleteMany()](https://github.com/fredyk/westack-go/issues/475)
  - [#478 - Create tests for Datasource.Close()](https://github.com/fredyk/westack-go/issues/478)
  - [Updated github.com/gofiber/fiber/v2 to v2.49.0](https://github.com/fredyk/westack-go/pull/499)

- **v1.6.0**

  - Added parameter `strictSingleRelatedDocumentCheck` in config.json, defaults to `true`in new projects, and `false` in existing ones.
  - `"hasOne"` and `"belongsTo"` relations are now checked after fetching documents from Mongo. If `strictSingleRelatedDocumentCheck` is `true` and the relation returns more than 1 document, an error is thrown. Otherwise, only the first document is used and a warning is logged.
  - **Breaking changes**:
    - `model.Build()` requires now parameter `sameLevelCache *buildCache` to be passed in. Can be generated with `model.NewBuildCache()`
    - `model.Build()` returns now `error` as second value, in addition to the instance. So it is now `func (loadedModel *Model) Build(data wst.M, sameLevelCache *buildCache, baseContext *EventContext) (Instance, error)`

- **v1.5.48**

  - **Breaking change**: environment variables `WST_ADMIN_USERNAME` and `WST_ADMIN_PWD` are required to start the server

### Contribute

Write to [westack.team@gmail.com](mailto://westack.team@gmail.com) if you want to contribute to the project: D

You are also welcome on our official [Discord](https://discord.gg/tFRYbGQWjZ)

And of course... create as many pull requests as you want!

