# Security Policy

## Supported Versions

We release patches for security vulnerabilities. Which versions are eligible for receiving such patches depends on the severity of the vulnerability and the version's age.

| Version | Supported          |
| ------- | ------------------ |
| 1.3.x   | :white_check_mark: |
| 1.2.x   | :white_check_mark: |
| 1.1.x   | :white_check_mark: |
| 1.0.x   | :white_check_mark: |
| < 1.0   | :x:                |

## Reporting a Vulnerability

**Please do not report security vulnerabilities through public GitHub issues.**

Instead, please report them via email to [security@example.com]. You should receive a response within 48 hours. If for some reason you do not, please follow up via email to ensure we received your original message.

Please include the following information (as much as you can provide) to help us better understand and resolve the issue:

* Type of issue (e.g., buffer overflow, SQL injection, cross-site scripting, etc.)
* Full paths of source file(s) related to the manifestation of the issue
* The location of the affected source code (tag/branch/commit or direct URL)
* Any special configuration required to reproduce the issue
* Step-by-step instructions to reproduce the issue
* Proof-of-concept or exploit code (if possible)
* Impact of the issue, including how an attacker might exploit the issue

This information will help us triage your report more quickly.

## Disclosure Policy

When we receive a security bug report, we will:

1. Confirm the problem and determine the affected versions
2. Audit code to find any similar problems
3. Prepare fixes for all affected versions still under maintenance
4. Release security advisories and new versions

## Comments on this Policy

If you have suggestions on how this process could be improved, please submit a pull request or open an issue to discuss.

## Security Best Practices

When using LangChain-Go in your applications:

### API Keys and Secrets

* **Never commit API keys or secrets to version control**
* Use environment variables or secret management services
* Rotate API keys regularly
* Use read-only or limited-scope API keys where possible

```go
// ❌ Bad
config := openai.Config{
    APIKey: "sk-1234567890abcdef",
}

// ✅ Good
config := openai.Config{
    APIKey: os.Getenv("OPENAI_API_KEY"),
}
```

### Input Validation

* Always validate and sanitize user input
* Use appropriate input limits to prevent resource exhaustion
* Be cautious with user-provided tool inputs

```go
// Validate input length
if len(userInput) > maxInputLength {
    return errors.New("input too long")
}

// Sanitize for specific use cases
sanitized := sanitizeInput(userInput)
```

### Tool Security

* Carefully review tool permissions before use
* Use whitelist-based tool access control
* Implement rate limiting for tool calls
* Monitor tool usage for anomalies

```go
// Example: Safe tool configuration
tool := tools.NewHTTPTool(tools.HTTPConfig{
    AllowedDomains: []string{"api.example.com"},
    MaxRequestSize: 1024 * 1024, // 1MB
    Timeout:        30 * time.Second,
})
```

### Database Security

* Use parameterized queries to prevent SQL injection
* Limit database permissions
* Encrypt sensitive data at rest
* Use connection pooling with appropriate limits

```go
// Example: Safe checkpoint usage
checkpointer, err := postgres.NewSaver(
    connectionString,
    postgres.WithMaxConnections(10),
    postgres.WithSSL(true),
)
```

### Error Handling

* Don't expose sensitive information in error messages
* Log errors appropriately without leaking secrets
* Handle panics gracefully

```go
// ❌ Bad - exposes API key
return fmt.Errorf("failed to call API with key %s", apiKey)

// ✅ Good - generic error
return fmt.Errorf("failed to call API: %w", err)
```

### Rate Limiting

* Implement rate limiting for API calls
* Use backoff strategies for retries
* Monitor usage to detect abuse

```go
// Example: Rate-limited client
client := openai.New(openai.Config{
    APIKey:      os.Getenv("OPENAI_API_KEY"),
    MaxRetries:  3,
    RetryDelay:  time.Second,
})
```

### Dependencies

* Keep dependencies up to date
* Review security advisories for dependencies
* Use `go mod verify` to check module authenticity

```bash
# Check for known vulnerabilities
go list -json -m all | nancy sleuth

# Update dependencies
go get -u ./...
go mod tidy
```

### Logging

* Don't log sensitive information (API keys, passwords, PII)
* Use structured logging
* Implement log rotation and retention policies

```go
// ❌ Bad - logs API key
log.Printf("Using API key: %s", apiKey)

// ✅ Good - logs without sensitive data
log.Printf("Initializing OpenAI client")
```

## Known Security Considerations

### LLM-Specific Risks

1. **Prompt Injection**: User input might manipulate the LLM's behavior
   - Solution: Use system prompts and input validation

2. **Data Leakage**: LLMs might inadvertently expose training data
   - Solution: Use appropriate models and review outputs

3. **Excessive Resource Use**: Complex queries can consume significant resources
   - Solution: Implement timeouts and resource limits

4. **Unreliable Output**: LLMs can produce incorrect or biased information
   - Solution: Implement verification and human review where critical

### Vector Store Security

1. **Data Privacy**: Vector stores contain embedded documents
   - Solution: Encrypt at rest, use access controls

2. **Injection Attacks**: Malicious documents could poison the search
   - Solution: Validate and sanitize document inputs

3. **Resource Exhaustion**: Large-scale searches can be expensive
   - Solution: Implement query limits and caching

## Security Updates

Subscribe to security advisories:

* Watch this repository for security notifications
* Follow us on [Twitter/X](https://twitter.com/yourhandle)
* Check the [GitHub Security Advisories](https://github.com/yourusername/langchain-go/security/advisories) page

## Hall of Fame

We recognize and thank security researchers who responsibly disclose vulnerabilities:

<!-- Will be populated as we receive reports -->

---

Thank you for helping keep LangChain-Go and our community safe!
