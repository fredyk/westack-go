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

5. **Create the main application file**:

   Create a `main.go` file with the following content:

   ```go
   package main

   import (
       "log"

       "github.com/fredyk/westack-go/westack"
   )

   func main() {
       app := westack.New()

       app.Boot()

       log.Fatal(app.Start())
   }
   ```

6. **Run the server**:

   ```bash
   go run main.go
   ```

7. **Test the API**:

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

Models are the backbone of westack-go, defined in JSON files under the `models/` directory. These JSON files specify attributes, relationships, and access policies using Casbin.

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

This will establish the relationship where `Footer` belongs to `Note` and `Note` has one `Footer`, allowing CRUD operations to respect the relationship automatically.

---

## Advanced Features

### Swagger Integration

westack-go automatically generates Swagger documentation for your APIs. The Swagger UI is available at:

- `/swagger`: Interactive API documentation.
- `/swagger/doc.json`: The OpenAPI specification in JSON format.

---
# Filters in westack-go

## Overview

Filters in westack-go allow developers to:

- Query, sort, paginate, and limit data retrieved from models.
- Build flexible APIs that support custom data slices without hardcoding query logic.

### Automatic Fields

westack-go automatically manages the following fields:

- **`created`**: Added when a record is created.
- **`modified`**: Updated whenever a record is modified.

### Use Cases

Filters can be applied to standard CRUD endpoints, such as `GET /notes`, to refine the data returned.

## Syntax and Structure

Filters are specified as JSON objects within the `filter` query parameter. Spaces within filter values should be replaced with the `+` character to ensure proper parsing by the API.

### 1. Filtering by Fields

You can filter records based on specific field values using the following format:

```http
GET /notes?filter={"where":{"field":"value"}}
```

Example:

```http
GET /notes?filter={"where":{"title":"Meeting"}}
```

This retrieves all `Note` records where the `title` field equals `Meeting`.

### 2. Advanced Conditions

For more complex filtering, you can use comparison operators:

- `$gt` (greater than)
- `$gte` (greater than or equal to)
- `$lt` (less than)
- `$lte` (less than or equal to)
- `$ne` (not equal to)
- `$in` (in array)
- `$regex` (regular expression)

Example:

```http
GET /notes?filter={"where":{"content":{"$regex":".*important.*"}}}
```

This retrieves all `Note` records where the `content` field contains the substring "important".

### 3. Sorting

You can sort records by one or more fields using the `order` parameter:

```http
GET /notes?filter={"order":["field+ASC"]}
```

Example:

```http
GET /notes?filter={"order":["title+ASC"]}
```

This retrieves all `Note` records sorted by the `title` field in ascending order.

### 4. Pagination

To limit the number of results returned and implement pagination, use the `limit` and `skip` parameters:

- `limit`: Specifies the maximum number of records to return.
- `skip`: Skips the specified number of records before returning results.

Example:

```http
GET /notes?filter={"limit":10,"skip":20}
```

This retrieves 10 `Note` records starting from the 21st record.

### 5. Field Selection

To retrieve only specific fields from a record, use the `fields` parameter:

```http
GET /notes?filter={"fields":{"field1":true,"field2":false}}
```

Example:

```http
GET /notes?filter={"fields":{"title":true,"content":false}}
```

This retrieves only the `title` field and excludes the `content` field for all `Note` records.

## Combining Filters

Filters can be combined to build complex queries:

```http
GET /notes?filter={"where":{"title":"Meeting"},"order":["title+DESC"],"limit":5}
```

This retrieves up to 5 `Note` records where the `title` is "Meeting", sorted in descending order by `title`.

## Examples in Practice

### Example 1: Basic Filtering

Retrieve all notes where `content` contains "urgent":

```http
GET /notes?filter={"where":{"content":{"$regex":".*urgent.*"}}}
```

### Example 2: Pagination and Sorting

Retrieve the first 10 notes sorted by `created` in descending order:

```http
GET /notes?filter={"order":["created+DESC"],"limit":10}
```

### Example 3: Advanced Pagination

Skip the first 5 notes and retrieve the next 15 notes:

```http
GET /notes?filter={"limit":15,"skip":5}
```

### Example 4: Complex Filtering

Retrieve all notes where `title` is "Meeting" and `content` does not contain "canceled":

```http
GET /notes?filter={"where":{"title":"Meeting","content":{"$not":{"$regex":".*canceled.*"}}}}
```

## Including Relations

To include related models, use the `include` parameter:

```http
GET /notes?filter={"include":[{"relation":"footer"}]}
```

This retrieves `Note` records along with their related `Footer` records.

### Including Relations with Filtering

You can include related models and apply additional filters simultaneously. For example:

```http
GET /notes?filter={"where":{"title":"Meeting"},"include":[{"relation":"footer"}]}
```

This retrieves `Note` records where the `title` is "Meeting" and includes the related `Footer` records.

## Limitations and Considerations

**Performance**: Complex filters might impact query performance, especially with large datasets.

Filters are a powerful feature of westack-go, making it easy to build flexible, queryable APIs. For further customization or troubleshooting, consult the source code or westack-go examples.


---

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

