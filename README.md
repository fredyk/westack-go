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
   go install github.com/fredyk/westack-go/v2@latest
   ```

   The CLI provides commands like `init` and `generate` for rapid development.

3. **Initialize the project**:

   Use the CLI to set up the project structure:

   ```bash
   westack-go init .
   ```

   This creates the basic structure, including configuration files and directories for models and controllers.

4. **Define a model**:

   Create a `models/note.json` file:

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
     "casbin": {
       "policies": [
         "$authenticated,*,*,allow",
         "$everyone,*,read,allow",
         "$owner,*,__get__footer,allow"
       ]
     }
   }
   ```

5. **Generate the model**:

   ```bash
   westack-go generate
   ```

6. **Create the main application file**:

   Create a `main.go` file with the following content:

   ```go
   package main

   import (
       "log"

       "github.com/fredyk/westack-go/v2/westack"
       "myproject/models"
   )

   func main() {
       app := westack.New()

       app.Boot(westack.BootOptions{
           RegisterControllers: models.RegisterControllers,
       })

       log.Fatal(app.Start())
   }
   ```

7. **Run the server**:

   ```bash
   go run main.go
   ```

8. **Test the API**:

   Access the Swagger UI at `http://localhost:3000/swagger` to test your endpoints.

---

## Core Concepts

### Architecture Overview

westack-go is built around the following core components:

- **Models**: Define the structure of your data and generate APIs automatically.
- **Datasources**: Abstract the details of data storage, supporting MongoDB and in-memory stores.
- **Routing**: Manage API endpoints and middleware.
- **Controllers**: Centralize business logic.
- **CLI Utilities**: Simplify repetitive tasks like generating models and controllers.

### Key Components

#### Models

Models are the backbone of westack-go, defined in JSON files under the `models/` directory. These JSON files specify attributes, relationships, and access policies using Casbin. From these definitions, `westack-go` generates Go struct files with the `westack-go generate` command.

Example of a JSON Model:

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
  "casbin": {
    "policies": [
      "$authenticated,*,*,allow",
      "$everyone,*,read,allow",
      "$owner,*,__get__footer,allow"
    ]
  }
}
```

When you run:

```bash
westack-go generate
```

This generates `Note` in `models/note.go` (if it does not already exist). The Go file can then be extended for additional functionality without affecting the original JSON definitions.

This dual-layer approach allows developers to:

- Keep JSON files as the source of truth for relationships and Casbin policies.
- Extend models in Go for advanced functionality.

> **Note**: In the future, the JSON files may be deprecated, and direct Go struct definitions might become the standard.

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
    "footer": {
      "type": "hasOne",
      "model": "Footer",
      "foreignKey": "noteId"
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

Run the following command to regenerate the models:

```bash
westack-go generate
```

This will establish the relationship where `Footer` belongs to `Note` and `Note` has one `Footer`, allowing CRUD operations to respect the relationship automatically.

---

## Building APIs

### Creating a New API Endpoint

1. **Define the Model** Create a JSON file in the `models/` directory and define your data structure.

2. **Generate the Go Struct** Run the following command to generate the corresponding Go file:

   ```bash
   westack-go generate
   ```

3. **Extend the Model** If needed, extend the generated Go struct file for additional functionality.

4. **Adding Custom Logic** Use `BindRemoteOperationWithOptions` to add new functionality or routes for an existing model.

Example:

```go
package boot

import (
    "log"
    "github.com/fredyk/westack-go/v2/model"
    "github.com/fredyk/westack-go/v2/westack"
)

func SetupServer(app *westack.WeStack) {
    NoteModel, err := app.FindModel("Note")
    if err != nil {
        log.Fatalf("Error finding model: %v", err)
    }

    model.BindRemoteOperationWithOptions(NoteModel, CustomHandler, model.RemoteOptions().
        WithName("customEndpoint").
        WithPath("/notes/custom").
        WithContentType("application/json"))
}

type CustomInput struct {
    Field1 string `json:"field1"`
    Field2 int    `json:"field2"`
}

type CustomOutput struct {
    Message string `json:"message"`
    Status  string `json:"status"`
}

func CustomHandler(input CustomInput) (CustomOutput, error) {
    return CustomOutput{
        Message: "This is a custom endpoint.",
        Status:  "success",
    }, nil
}
```

---

## Advanced Features

### Swagger Integration

westack-go automatically generates Swagger documentation for your APIs. The Swagger UI is available at:

- `/swagger`: Interactive API documentation.
- `/swagger/doc.json`: The OpenAPI specification in JSON format.

---

## Testing

### Running Tests

Run all tests:

```bash
go test ./...
```

### Writing Test Cases

Create test files in the `tests/` directory.

### Change Log

- **v2.0.1-alpha**

  - Now the DELETE /\:id endpoint returns a [wst.DeleteResult](https://github.com/fredyk/westack-go/blob/39d4e5a7b71fd3f3ce11d926a967a730d665a9fe/v2/common/common.go#L608) schema object instead of an empty response
  - Now the GET /count endpoint returns a [wst.CountResult](https://github.com/fredyk/westack-go/blob/39d4e5a7b71fd3f3ce11d926a967a730d665a9fe/v2/common/common.go#L614) schema object instead of a root integer

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

