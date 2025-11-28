# westack-go Documentation

Comprehensive documentation for the westack-go framework.

## 📚 Table of Contents

### 🔒 [Security](./01-security/)

Security best practices, vulnerabilities prevention, and secure coding guidelines.

#### Authentication
- [Password Requirements](./01-security/01-authentication/01-password-requirements.md) - Secure password policies and validation
- [Cookie Security](./01-security/01-authentication/02-cookie-security.md) - OAuth cookies and session security

#### Authorization
- [Permission Propagation](./01-security/02-authorization/01-permission-propagation.md) - Bearer token flow and RBAC enforcement

#### Injection Prevention
- [NoSQL Injection](./01-security/03-injection-prevention/01-nosql-injection.md) - MongoDB operator whitelisting and sanitization

#### Error Handling
- [Information Disclosure](./01-security/04-error-handling/01-information-disclosure.md) - Secure error handling practices

### 🚀 [Deployment](./02-deployment/)

Deployment guides and production best practices.

- [Docker Deployment](./02-deployment/01-docker-deployment.md) - Containerization and Docker best practices

## 🔍 Quick Navigation

**By Topic**:
- **Authentication & Login**: [Passwords](./01-security/01-authentication/01-password-requirements.md), [Cookies](./01-security/01-authentication/02-cookie-security.md)
- **Authorization & Permissions**: [RBAC](./01-security/02-authorization/01-permission-propagation.md)
- **Security Vulnerabilities**: [NoSQL Injection](./01-security/03-injection-prevention/01-nosql-injection.md), [Info Disclosure](./01-security/04-error-handling/01-information-disclosure.md)
- **Production**: [Docker](./02-deployment/01-docker-deployment.md)

## 📖 Documentation Standards

All documentation in this repository follows these principles:

1. **No Internal Implementation**: Documentation describes behavior and interfaces, not internal code
2. **Clear Structure**: Hierarchical organization with numbered categories
3. **Cross-References**: Links between related documents
4. **Examples**: Practical usage examples without revealing implementation details
5. **Security First**: Security considerations highlighted in every guide

## 🤝 Contributing

When adding new documentation:

1. Place it in the appropriate category directory
2. Use numbered prefixes for ordering (e.g., `03-new-topic.md`)
3. Follow existing document structure
4. Add entry to this README
5. Cross-reference related documents
6. Avoid exposing internal implementation details

## 📝 Document Structure Template

```markdown
# Title

## Overview
Brief description of the topic

## The Threat / The Problem
What issue does this address?

## Implementation
How is it implemented? (behavior, not code)

## Best Practices
Recommended approaches

## Testing
How to verify correct implementation

## Related Topics
Links to related documentation
```

## 🔗 External Resources

- [OWASP Top 10](https://owasp.org/www-project-top-ten/)
- [MongoDB Security Checklist](https://docs.mongodb.com/manual/administration/security-checklist/)
- [Docker Security Best Practices](https://docs.docker.com/engine/security/)

---

**Framework**: westack-go  
**Documentation Version**: 1.0  
**Last Updated**: 2025-11-28
