# API Documentation

Documentation for westack-go REST API endpoints and behaviors.

## 📚 Table of Contents

### Relations
- [Relation Endpoints](./01-relations/01-relation-endpoints.md) - Access related data through REST endpoints

## Overview

westack-go automatically generates RESTful API endpoints for all models. This section documents the available endpoints, their behaviors, and how to use them effectively.

## Base Endpoints

For each model, the following base endpoints are automatically created:

| Method | Path | Description |
|--------|------|-------------|
| GET | `/{modelPlural}` | Find all instances |
| GET | `/{modelPlural}/count` | Count instances |
| GET | `/{modelPlural}/:id` | Find by ID |
| POST | `/{modelPlural}` | Create new instance |
| PATCH | `/{modelPlural}/:id` | Update instance |
| DELETE | `/{modelPlural}/:id` | Delete instance |

## Relation Endpoints

For models with relations, additional endpoints are automatically mounted:

| Method | Path | Description |
|--------|------|-------------|
| GET | `/{modelPlural}/:id/{relationName}` | Get related items (array or object) |
| GET | `/{modelPlural}/:id/{relationName}/:fk` | Get specific related item by ID |
| GET | `/{modelPlural}/:id/{relationName}/count` | Count related items (hasMany only) |

See [Relation Endpoints](./01-relations/01-relation-endpoints.md) for detailed documentation.

## Related Topics

- [Permission Propagation](../01-security/02-authorization/01-permission-propagation.md) - How permissions are checked for API calls
- [NoSQL Injection Prevention](../01-security/03-injection-prevention/01-nosql-injection.md) - Query sanitization
