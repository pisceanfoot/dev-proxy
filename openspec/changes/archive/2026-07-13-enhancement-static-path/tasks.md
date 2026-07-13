## 1. Path Safety

- [x] 1.1 Add `filepath.EvalSymlinks` resolution of `staticDir` at handler construction time so symlinks in the root are resolved once
- [x] 1.2 After joining `staticDir` with the URL path, resolve symlinks on the joined path and check that it is still prefixed by the resolved `staticDir`; return 403 if outside

## 2. Core Error Response Logic

- [x] 2.1 Replace the current `os.Stat` error branch: if `os.IsNotExist(err)` return HTTP 404 with body `404 Not Found\n` and `Content-Type: text/plain`
- [x] 2.2 For any other `os.Stat` error (permission denied, I/O error, etc.) return HTTP 500 with body `500 Internal Server Error: <err.Error()>\n` and `Content-Type: text/plain`
- [x] 2.3 Remove the fall-through to `next.ServeHTTP` for error cases; the static handler is now authoritative when `static_dir` is set

## 3. Directory Listing

- [x] 3.1 When `info.IsDir()` is true, call `os.ReadDir(filePath)`; on error return HTTP 500 with the OS error message
- [x] 3.2 Define an `html/template` (package-level `var`) that renders a directory listing with table columns: Name (hyperlink), Last Modified, Size
- [x] 3.3 Execute the template with the entry list and write the result to the response with `Content-Type: text/html` and HTTP 200
- [x] 3.4 Ensure subdirectory entries in the listing link to their path with a trailing slash; file entries link without trailing slash

## 4. Retain Normal File Serving

- [x] 4.1 Verify the happy-path (`http.ServeFile`) still works correctly after the refactor; existing `detectContentType` function should remain unchanged
- [x] 4.2 Confirm that a file at the root of `static_dir` (e.g. `/index.html`) is still served with HTTP 200 and correct `Content-Type`

## 5. Tests

- [x] 5.1 Write a test for 404 when file does not exist (`internal/static/static_test.go`)
- [x] 5.2 Write a test for 500 when `os.Stat` returns a non-not-found error
- [x] 5.3 Write a test for 403 on a path-traversal URL
- [x] 5.4 Write a test for directory listing: response is 200, `Content-Type: text/html`, body contains entry names as links
- [x] 5.5 Write a test for empty directory: response is 200, body is valid HTML with no entries
- [x] 5.6 Write a test for 500 when `os.ReadDir` fails on an unreadable directory
- [x] 5.7 Write a test for successful file serving (existing behaviour)
