# Contributing to LangChain-Go

First off, thank you for considering contributing to LangChain-Go! It's people like you that make LangChain-Go such a great tool.

## Code of Conduct

This project and everyone participating in it is governed by our Code of Conduct. By participating, you are expected to uphold this code. Please report unacceptable behavior to [your.email@example.com].

## How Can I Contribute?

### Reporting Bugs

Before creating bug reports, please check the issue list as you might find out that you don't need to create one. When you are creating a bug report, please include as many details as possible:

* **Use a clear and descriptive title**
* **Describe the exact steps which reproduce the problem**
* **Provide specific examples to demonstrate the steps**
* **Describe the behavior you observed after following the steps**
* **Explain which behavior you expected to see instead and why**
* **Include Go version, OS, and relevant environment details**

**Bug Report Template:**

```markdown
## Bug Description
A clear and concise description of the bug.

## To Reproduce
Steps to reproduce the behavior:
1. Go to '...'
2. Run '...'
3. See error

## Expected Behavior
What you expected to happen.

## Actual Behavior
What actually happened.

## Environment
- Go version: [e.g. 1.22.0]
- OS: [e.g. macOS 14.0]
- LangChain-Go version: [e.g. 1.3.0]

## Additional Context
Add any other context about the problem here.
```

### Suggesting Enhancements

Enhancement suggestions are tracked as GitHub issues. When creating an enhancement suggestion, please include:

* **Use a clear and descriptive title**
* **Provide a step-by-step description of the suggested enhancement**
* **Provide specific examples to demonstrate the steps**
* **Describe the current behavior and explain the expected behavior**
* **Explain why this enhancement would be useful**

**Feature Request Template:**

```markdown
## Feature Description
A clear and concise description of what you want to happen.

## Use Case
Describe the problem you're trying to solve.

## Proposed Solution
Describe how you envision this feature working.

## Alternatives Considered
Describe alternatives you've considered.

## Additional Context
Add any other context or screenshots about the feature request here.
```

### Pull Requests

* Fill in the required template
* Follow the Go coding style
* Include thoughtfully-worded, well-structured tests
* Document new code
* End all files with a newline

## Development Process

### Setting Up Your Development Environment

1. **Fork the repository**

```bash
# Fork on GitHub, then clone your fork
git clone https://github.com/YOUR_USERNAME/langchain-go.git
cd langchain-go
```

2. **Install dependencies**

```bash
go mod download
```

3. **Create a branch**

```bash
git checkout -b feature/my-new-feature
# or
git checkout -b fix/issue-123
```

### Coding Guidelines

#### Go Style Guide

We follow the official [Go Code Review Comments](https://github.com/golang/go/wiki/CodeReviewComments) and [Effective Go](https://golang.org/doc/effective_go).

**Key points:**

1. **Formatting**: Use `gofmt` or `go fmt`
   ```bash
   go fmt ./...
   ```

2. **Linting**: Use `golangci-lint`
   ```bash
   golangci-lint run
   ```

3. **Naming Conventions**:
   - Use MixedCaps for exported names
   - Use camelCase for unexported names
   - Keep names short and meaningful
   - Avoid stuttering (e.g., `http.HTTPServer` → `http.Server`)

4. **Documentation**:
   - Every exported function, type, and package must have a doc comment
   - Doc comments start with the name of the thing being described
   - Use complete sentences

   ```go
   // NewClient creates a new OpenAI client with the given configuration.
   //
   // The client is safe for concurrent use by multiple goroutines.
   func NewClient(config Config) (*Client, error) {
       // ...
   }
   ```

5. **Error Handling**:
   - Return errors, don't panic (except in truly exceptional cases)
   - Use `fmt.Errorf` with `%w` for error wrapping
   - Provide context in error messages

   ```go
   if err != nil {
       return fmt.Errorf("failed to load document: %w", err)
   }
   ```

6. **Testing**:
   - Write tests for all new code
   - Use table-driven tests where appropriate
   - Use meaningful test names

   ```go
   func TestVectorStore_Search(t *testing.T) {
       tests := []struct {
           name    string
           query   string
           k       int
           want    int
           wantErr bool
       }{
           {
               name:  "basic search",
               query: "test query",
               k:     5,
               want:  5,
           },
           // ... more test cases
       }
       
       for _, tt := range tests {
           t.Run(tt.name, func(t *testing.T) {
               // test implementation
           })
       }
   }
   ```

#### Project-Specific Guidelines

1. **Read `.cursorrules`**: Our detailed coding standards are documented in [.cursorrules](.cursorrules)

2. **Use Generics Appropriately**: LangChain-Go uses Go generics for type safety
   ```go
   type Runnable[In, Out any] interface {
       Invoke(ctx context.Context, input In, opts ...Option) (Out, error)
   }
   ```

3. **Context Handling**: Always accept `context.Context` as the first parameter
   ```go
   func Process(ctx context.Context, input string) error {
       // ...
   }
   ```

4. **Concurrency**: Use goroutines and channels judiciously
   - Document concurrency behavior
   - Ensure proper cleanup with `defer`
   - Use `sync.Mutex` or `sync.RWMutex` for shared state

5. **Package Organization**:
   - Keep packages focused and cohesive
   - Avoid circular dependencies
   - Use internal packages for implementation details

### Testing

#### Running Tests

```bash
# Run all tests
go test ./...

# Run tests with coverage
go test -coverprofile=coverage.out ./...
go tool cover -html=coverage.out

# Run tests for a specific package
go test ./core/chat/...

# Run tests with race detector
go test -race ./...

# Run benchmarks
go test -bench=. ./...
```

#### Writing Tests

1. **Unit Tests**: Test individual functions and methods
2. **Integration Tests**: Test interactions between components
3. **Benchmark Tests**: Measure performance

```go
// Example unit test
func TestCalculator_Evaluate(t *testing.T) {
    calc := NewCalculator()
    result, err := calc.Evaluate("2 + 2")
    
    assert.NoError(t, err)
    assert.Equal(t, 4.0, result)
}

// Example benchmark
func BenchmarkCalculator_Evaluate(b *testing.B) {
    calc := NewCalculator()
    
    b.ResetTimer()
    for i := 0; i < b.N; i++ {
        _, _ = calc.Evaluate("2 + 2 * 3")
    }
}
```

#### Test Coverage

- Aim for at least 70% coverage for new code
- Critical paths should have 90%+ coverage
- Don't sacrifice test quality for coverage numbers

### Documentation

#### Code Documentation

- Use GoDoc format for all public APIs
- Include examples in doc comments when helpful
- Document concurrency behavior
- Document panics if any

```go
// Process processes the input and returns the result.
//
// This function is safe for concurrent use.
//
// Example:
//
//	result, err := Process(ctx, "input")
//	if err != nil {
//	    log.Fatal(err)
//	}
//	fmt.Println(result)
func Process(ctx context.Context, input string) (string, error) {
    // ...
}
```

#### Markdown Documentation

- Keep documentation up-to-date with code changes
- Use clear, concise language
- Include code examples
- Add diagrams where helpful

### Commit Messages

We follow [Conventional Commits](https://www.conventionalcommits.org/):

```
<type>(<scope>): <subject>

<body>

<footer>
```

**Types:**
- `feat`: New feature
- `fix`: Bug fix
- `docs`: Documentation changes
- `style`: Code style changes (formatting, etc.)
- `refactor`: Code refactoring
- `perf`: Performance improvements
- `test`: Test additions or changes
- `chore`: Build process or auxiliary tool changes

**Examples:**

```
feat(vectorstore): add Milvus hybrid search support

Implement hybrid search combining vector similarity and BM25 keyword
search for Milvus 2.6+. Supports both RRF and weighted reranking.

Closes #123
```

```
fix(agent): prevent infinite loop in ReAct agent

Add max steps check to prevent infinite loops when the agent
cannot reach a conclusion.

Fixes #456
```

### Pull Request Process

1. **Update Documentation**: Update README.md, CHANGELOG.md, and relevant docs
2. **Add Tests**: Ensure your PR includes tests for new functionality
3. **Run Tests**: Make sure all tests pass
4. **Update CHANGELOG**: Add an entry to [CHANGELOG.md](CHANGELOG.md)
5. **Fill PR Template**: Use our PR template

**PR Template:**

```markdown
## Description
Brief description of the changes.

## Type of Change
- [ ] Bug fix (non-breaking change which fixes an issue)
- [ ] New feature (non-breaking change which adds functionality)
- [ ] Breaking change (fix or feature that would cause existing functionality to not work as expected)
- [ ] Documentation update

## How Has This Been Tested?
Describe the tests you ran and how to reproduce them.

## Checklist
- [ ] My code follows the style guidelines of this project
- [ ] I have performed a self-review of my own code
- [ ] I have commented my code, particularly in hard-to-understand areas
- [ ] I have made corresponding changes to the documentation
- [ ] My changes generate no new warnings
- [ ] I have added tests that prove my fix is effective or that my feature works
- [ ] New and existing unit tests pass locally with my changes
- [ ] Any dependent changes have been merged and published
```

### Review Process

1. **Automated Checks**: CI/CD will run tests and linters
2. **Peer Review**: At least one maintainer will review your PR
3. **Feedback**: Address review comments
4. **Approval**: Once approved, your PR will be merged

## Community

### Getting Help

- **GitHub Discussions**: Ask questions and discuss ideas
- **GitHub Issues**: Report bugs and request features
- **Documentation**: Check our [docs](./docs) folder and [QUICK_START.md](./QUICK_START.md)

### Recognition

Contributors are recognized in:
- [AUTHORS](AUTHORS) file
- Release notes
- Project documentation

## License

By contributing to LangChain-Go, you agree that your contributions will be licensed under the MIT License.

---

Thank you for contributing to LangChain-Go! 🎉
