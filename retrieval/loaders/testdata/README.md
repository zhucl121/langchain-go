# PDF Loader Test Data

This directory contains test PDF files for the PDF loader tests.

## How to Test

To run the full PDF loader tests, you need to add test PDF files to this directory.

### Creating a Test PDF

You can create a simple test PDF using any of these methods:

1. **Using a PDF library (recommended for CI)**:
   ```bash
   # You can use a Go script with a PDF generation library
   ```

2. **Manual creation**:
   - Create a simple document in any text editor
   - Print/Save as PDF with the name `sample.pdf`
   - Place it in this `testdata` directory

3. **Download sample PDFs**:
   - Download any free sample PDF from the internet
   - Rename it to `sample.pdf`
   - Place it in this directory

### Expected Test Files

- `sample.pdf` - A multi-page PDF for general testing
- `encrypted.pdf` (optional) - A password-protected PDF for testing password功能

## Running Tests

```bash
# Run all tests (will skip tests without PDF files)
go test -v ./retrieval/loaders

# Run only PDF tests
go test -v ./retrieval/loaders -run TestPDF
```

## Notes

- Tests will automatically skip if `sample.pdf` is not found
- This is intentional to allow tests to run in environments without test files
- For CI/CD, consider using automated PDF generation
