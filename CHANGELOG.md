# Changelog

All notable changes to westack-go will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.0.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [Unreleased]

### Added
- GitHub OAuth username and private email retrieval via dual API calls

## [2.4.0-rc01] - 2025-02-19

### 🚀 Added
- **GitHub OAuth Enhanced Profile Retrieval**
  - Dual API call implementation for complete GitHub user data
  - Username extraction from `/user` endpoint (`login` field)
  - Private email extraction from `/user/emails` endpoint
  - Intelligent email selection algorithm (primary → verified → first)
  - Comprehensive debug logging for troubleshooting

### 🔧 Fixed
- **GitHub OAuth Configuration**
  - Updated userinfo URL from `/user/emails` to `/user` for profile data
  - Added `user:email` scope for private email access
  - Removed array processing (now handles single user object)
  - Eliminated unused variables and duplicate code

### 🔄 Changed
- **GitHub Profile Auto-Extraction**
  - Automatic extraction of GitHub profile fields when no custom mapping
  - Fields extracted: `username`, `name`, `picture`, `bio`, `location`
  - Combined with private email from separate API call

### 📊 Technical Details
- **API Flow:**
  1. `GET /user` → username, name, bio, avatar, location
  2. `GET /user/emails` → private emails with primary/verified flags
  3. Intelligent email selection algorithm
  4. Data combination in `userInfoData`

- **Email Selection Priority:**
  1. Primary email (`"primary": true`)
  2. Verified email (`"verified": true`) 
  3. First available email (fallback)

- **Response Format Support:**
  - Standard GitHub email objects
  - Enterprise Managed Users (EMUs) with placeholder emails
  - Noreply emails (`@users.noreply.github.com`)
  - All visibility levels (`public`, `private`, `null`)

### 🛡️ Security
- Maintained OAuth security flags (HTTPOnly, Secure, SameSite)
- Robust error handling with fallbacks
- No breaking changes for other OAuth providers

### 📝 Examples
**Debug Output:**
```
[DEBUG] Making second call to GitHub /user/emails endpoint
[DEBUG] GitHub emails response: [{email:octocat@github.com primary:true verified:true visibility:private}]
[DEBUG] Found primary email: octocat@github.com
[DEBUG] Added email to userInfoData: octocat@github.com
[DEBUG] GitHub username (login): octocat
[DEBUG] GitHub OAuth profile fields:
[DEBUG] - login (username): octocat
[DEBUG] - name: The Octocat
[DEBUG] - avatar_url: https://github.com/images/error/octocat_happy.gif
[DEBUG] Final additionalUserInfo: map[username:octocat name:The Octocat picture:https://github.com/images/error/octocat_happy.gif]
```

**Final Combined Data:**
```json
{
  "login": "octocat",
  "email": "octocat@github.com",
  "name": "The Octocat",
  "avatar_url": "https://github.com/images/error/octocat_happy.gif",
  "bio": "There once was...",
  "location": "San Francisco"
}
```

### 🎯 Impact
- **Before**: Only public email (often `null`) + no username
- **After**: Complete profile + private email + username
- **Use Case**: Perfect for user account creation and profile management

---

## [2.3.x] - Previous Releases

*Previous changelog entries would be documented here*
