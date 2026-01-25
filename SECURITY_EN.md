# Security Policy

🌍 **Language**: [中文](SECURITY.md) | English

## Supported Versions

Currently supported versions for security updates:

| Version | Support Status |
| --- | --- |
| 0.6.x | ✅ |
| 0.5.x | ✅ |
| 0.1.x - 0.4.x | ⚠️ Limited |
| < 0.1.0 | ❌ |

## Reporting Security Vulnerabilities

If you discover a security vulnerability, please **DO NOT** publicly submit an issue.

Please report privately through the following methods:

1. **GitHub Security Advisory**: 
   - Visit https://github.com/zhucl121/langchain-go/security/advisories/new
   - Fill in vulnerability details

2. **Email Report**: 
   - Send email to project maintainers (if provided)
   - Email subject: `[SECURITY] LangChain-Go - Vulnerability Brief`

### Report Should Include

- Vulnerability type
- Affected versions
- Steps to reproduce
- Potential impact
- Suggested fix (if any)

### Response Timeline

- **Acknowledgment**: We will acknowledge receipt within 48 hours
- **Assessment**: Complete vulnerability assessment within 7 days
- **Fix**: Release fix within 14-30 days depending on severity

### Disclosure Policy

- We follow responsible disclosure principles
- After fix is released, we will publicly disclose at an appropriate time
- We will acknowledge reporters in release notes (unless you choose to remain anonymous)

## Security Best Practices

Security recommendations when using this project:

### 1. Key Management

- Never hardcode API keys in code
- Use environment variables to store sensitive information
- Regularly rotate API keys

**Good Practice:**

```go
// ❌ Bad - Hardcoded API key
model := openai.New(openai.Config{
    APIKey: "sk-proj-xxxxxxxxxxxxx",
})

// ✅ Good - Use environment variable
model := openai.New(openai.Config{
    APIKey: os.Getenv("OPENAI_API_KEY"),
})
```

### 2. Dependency Management

- Regularly update dependencies to latest versions
- Use `go mod tidy` to clean up unnecessary dependencies
- Follow security announcements
- Use `go list -m -u all` to check for updates

**Commands:**

```bash
# Check for updates
go list -m -u all

# Update dependencies
go get -u ./...
go mod tidy

# Verify dependencies
go mod verify
```

### 3. Input Validation

- Validate all user inputs
- Perform security checks on LLM outputs
- Avoid directly executing unverified code
- Sanitize inputs before processing

**Example:**

```go
// Validate user input
func ValidateQuery(query string) error {
    if len(query) == 0 {
        return errors.New("query cannot be empty")
    }
    if len(query) > 10000 {
        return errors.New("query too long")
    }
    // Check for malicious patterns
    if containsMaliciousPattern(query) {
        return errors.New("potentially malicious input")
    }
    return nil
}
```

### 4. Network Security

- Use HTTPS for API calls
- Verify TLS certificates
- Implement rate limiting
- Use timeouts for all network operations

**Example:**

```go
// Configure secure HTTP client
client := &http.Client{
    Timeout: 30 * time.Second,
    Transport: &http.Transport{
        TLSClientConfig: &tls.Config{
            MinVersion: tls.VersionTLS12,
        },
    },
}
```

### 5. Authentication & Authorization

- Implement proper authentication (v0.6.0+)
- Use RBAC for access control
- Enable audit logging
- Implement multi-tenancy isolation

**Example (v0.6.0+):**

```go
// Configure RBAC
rbacConfig := rbac.Config{
    Enabled: true,
    Store:   postgresStore,
}
rbacChecker := rbac.NewChecker(rbacConfig)

// Check permissions
if err := rbacChecker.CheckPermission(ctx, userID, "agent:execute"); err != nil {
    return fmt.Errorf("permission denied: %w", err)
}
```

### 6. Data Protection

- Encrypt sensitive data at rest
- Use secure data masking for logs
- Implement data retention policies
- Enable encryption in transit

### 7. Error Handling

- Don't expose sensitive information in error messages
- Log errors securely
- Use structured logging

**Example:**

```go
// ❌ Bad - Exposes sensitive info
return fmt.Errorf("failed to connect to database: %s with password %s", 
    dbHost, dbPassword)

// ✅ Good - Secure error handling
logger.Error("database connection failed", 
    zap.String("host", dbHost),
    zap.Error(err))
return errors.New("database connection failed")
```

## Security Features

### Enterprise Security (v0.6.0+)

LangChain-Go provides enterprise-grade security features:

1. **RBAC (Role-Based Access Control)**
   - Fine-grained permission control
   - Role hierarchy
   - Permission caching

2. **Multi-Tenancy**
   - Tenant isolation
   - Resource quota management
   - Tenant-specific configurations

3. **Audit Logging**
   - Complete operation tracking
   - Searchable audit trails
   - Compliance reporting

4. **Data Security**
   - Field-level encryption
   - Data masking
   - Secure key management

5. **API Authentication**
   - JWT token support
   - API key authentication
   - OAuth2 integration

See [v0.6.0 Documentation](docs/V0.6.0_PROGRESS.md) for details.

## Security Checklist

### Development Phase

- [ ] All API keys stored in environment variables
- [ ] Input validation implemented
- [ ] Error messages don't expose sensitive data
- [ ] Secure dependencies reviewed
- [ ] Code linter checks passed
- [ ] Security tests written

### Deployment Phase

- [ ] HTTPS enabled for all endpoints
- [ ] Rate limiting configured
- [ ] Authentication enabled
- [ ] RBAC permissions configured
- [ ] Audit logging enabled
- [ ] Monitoring and alerting set up
- [ ] Secrets properly managed
- [ ] Database encryption enabled

### Production Monitoring

- [ ] Regular security audits
- [ ] Dependency vulnerability scanning
- [ ] Log monitoring for suspicious activity
- [ ] Incident response plan in place
- [ ] Regular backups configured
- [ ] Disaster recovery tested

## Known Security Considerations

### 1. LLM-Specific Risks

- **Prompt Injection**: Validate and sanitize all prompts
- **Data Leakage**: Don't send sensitive data to external LLMs
- **Model Hallucination**: Verify LLM outputs before executing

### 2. Vector Store Security

- Secure connection strings
- Use authentication for vector databases
- Implement access control for collections

### 3. Tool Execution

- Sandbox tool execution when possible
- Validate tool inputs and outputs
- Limit tool capabilities based on context

## Security Updates

### Subscribe to Security Updates

- Watch this repository and select "Security alerts"
- Follow GitHub Security Advisories
- Join our security mailing list (if available)

### Security Contact

For security concerns, please contact:
- GitHub Security Advisory: [Create Advisory](https://github.com/zhucl121/langchain-go/security/advisories/new)
- Email: (To be added by maintainers)

## Vulnerability Disclosure Timeline

1. **Day 0**: Report received
2. **Day 1-2**: Acknowledgment sent
3. **Day 3-7**: Vulnerability assessment
4. **Day 8-30**: Fix development and testing
5. **Day 31**: Security patch released
6. **Day 45**: Public disclosure (if applicable)

## Security Hall of Fame

We thank the following security researchers for their contributions:

- (To be updated with contributors)

---

**Security is a shared responsibility. Thank you for helping keep LangChain-Go secure!** 🔒
