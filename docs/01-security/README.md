# Security Documentation

Comprehensive security documentation for westack-go framework.

## 📋 Overview

This section covers all security aspects of the framework, from authentication to vulnerability prevention.

## 📚 Categories

### 🔐 [01. Authentication](./01-authentication/)

User authentication, password security, and session management.

- **[Password Requirements](./01-authentication/01-password-requirements.md)**
  - Minimum 8 characters with complexity rules
  - Bcrypt hashing with JWT secret salt
  - Validation at account creation and password updates

- **[Cookie Security](./01-authentication/02-cookie-security.md)**
  - OAuth SSID cookie security flags (HTTPOnly, Secure, SameSite)
  - CSRF protection in OAuth 2.0 flow
  - Session cookie best practices

### 🛡️ [02. Authorization](./02-authorization/)

Permissions, RBAC, and access control.

- **[Permission Propagation](./02-authorization/01-permission-propagation.md)**
  - Bearer token flow from HTTP to datasource
  - EventContext chain and BaseContext linking
  - EnforceEx() permission verification with Casbin
  - System context for internal operations

### 💉 [03. Injection Prevention](./03-injection-prevention/)

Protection against injection attacks.

- **[NoSQL Injection](./03-injection-prevention/01-nosql-injection.md)**  
  - MongoDB operator whitelisting (comparison, logical, array)
  - Dangerous operator blocking ($where, $function, $accumulator)
  - ReDoS protection for regex patterns
  - Automatic sanitization in ParseFilter()

### 🚨 [04. Error Handling](./04-error-handling/)

Secure error handling and information disclosure prevention.

- **[Information Disclosure](./04-error-handling/01-information-disclosure.md)**
  - Error message sanitization
  - DEBUG-based detail filtering (production vs development)
  - Stack trace protection
  - Database error exposure prevention

## 🔍 Security by Layer

### HTTP Layer
- [Cookie Security](./01-authentication/02-cookie-security.md) - Session cookies
- [NoSQL Injection](./03-injection-prevention/01-nosql-injection.md) - Query parameter sanitization

### Application Layer
- [Password Requirements](./01-authentication/01-password-requirements.md) - Input validation
- [Permission Propagation](./02-authorization/01-permission-propagation.md) - Authorization checks

### Data Layer
- [NoSQL Injection](./03-injection-prevention/01-nosql-injection.md) - Database query protection
- [Permission Propagation](./02-authorization/01-permission-propagation.md) - Data access control

## ⚠️ Critical Security Features

### ✅ Implemented & Verified

| Feature | Status | Documentation |
|---------|--------|---------------|
| Password Complexity | ✅ Active | [Password Requirements](./01-authentication/01-password-requirements.md) |
| NoSQL Injection Protection | ✅ Active | [NoSQL Injection](./03-injection-prevention/01-nosql-injection.md) |
| Permission Propagation | ✅ Active | [Permission Propagation](./02-authorization/01-permission-propagation.md) |
| OAuth Cookie Security | ✅ Active | [Cookie Security](./01-authentication/02-cookie-security.md) |
| Timing Attack Protection | ✅ Active | Login implementation |
| Account Lockout | ✅ Active | Login implementation (5 attempts, 30 min) |

### ⚠️ Recommendations

| Feature | Priority | Documentation |
|---------|----------|---------------|
| Error Detail Filtering | 🟡 High | [Information Disclosure](./04-error-handling/01-information-disclosure.md) |
| Rate Limiting | 🟡 Medium | Available but optional |

## 🔐 Security Checklist

Use this checklist when deploying to production:

- [ ] **Authentication**
  - [ ] Secure passwords enforced (8+ chars, complexity)
  - [ ] OAuth cookies have security flags
  - [ ] HTTPS enforced for Secure cookies

- [ ] **Authorization**
  - [ ] Casbin policies configured
  - [ ] Permission propagation tested
  - [ ] System context only for internal operations

- [ ] **Injection Prevention**
  - [ ] Query sanitization active
  - [ ] NoSQL operator whitelist verified
  - [ ] Regex patterns validated

- [ ] **Error Handling**
  - [ ] DEBUG=false in production
  - [ ] Generic error messages to clients
  - [ ] Detailed logging server-side only

- [ ] **General**
  - [ ] All dependencies updated
  - [ ] Security logging enabled
  - [ ] Account lockout active
  - [ ] Rate limiting configured

## 📖 Related Documentation

- [Deployment](../02-deployment/) - Production deployment guides
- [Main README](../README.md) - Complete documentation index

## 🔗 External Security Resources

- **OWASP**
  - [Top 10](https://owasp.org/www-project-top-ten/)
  - [Authentication Cheat Sheet](https://cheatsheetseries.owasp.org/cheatsheets/Authentication_Cheat_Sheet.html)
  - [Authorization Cheat Sheet](https://cheatsheetseries.owasp.org/cheatsheets/Authorization_Cheat_Sheet.html)
  - [NoSQL Injection](https://owasp.org/www-project-web-security-testing-guide/latest/4-Web_Application_Security_Testing/07-Input_Validation_Testing/05.6-Testing_for_NoSQL_Injection)

- **CWE**
  - [CWE-89: SQL Injection](https://cwe.mitre.org/data/definitions/89.html)
  - [CWE-209: Information Exposure Through an Error Message](https://cwe.mitre.org/data/definitions/209.html)
  - [CWE-521: Weak Password Requirements](https://cwe.mitre.org/data/definitions/521.html)

---

**Security First**: All features in westack-go are designed with security as a primary concern.  
**Defense in Depth**: Multiple layers of protection prevent single points of failure.  
**Continuous Improvement**: Security documentation updated as new threats emerge.
