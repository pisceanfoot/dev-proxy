## ADDED Requirements

### Requirement: Directory hit returns HTML listing
When a request targets a `static_dir` route and the resolved path is a directory, the server SHALL return HTTP 200 with an HTML page listing the directory's entries, including file names, sizes, and last-modified timestamps.

#### Scenario: Root directory listing
- **WHEN** a GET request is made for `/` on a route with `static_dir` set
- **THEN** the server responds with HTTP 200
- **THEN** the response `Content-Type` is `text/html`
- **THEN** the response body contains an HTML listing of the top-level entries in `static_dir`

#### Scenario: Sub-directory listing
- **WHEN** a GET request is made for `/assets/` and `assets` is a directory under `static_dir`
- **THEN** the server responds with HTTP 200
- **THEN** the response body lists the files and subdirectories inside `assets`

#### Scenario: Listing includes file metadata
- **WHEN** a directory listing is rendered
- **THEN** each entry shows its file name as a hyperlink
- **THEN** each entry shows its last-modified date
- **THEN** each file entry shows its size in bytes

#### Scenario: Directory entries are navigable
- **WHEN** a directory listing is rendered for `/assets/`
- **THEN** each file name is an anchor tag (`<a>`) with `href` pointing to the correct sub-path
- **THEN** subdirectory names link to their own listing path

### Requirement: Empty directory listing is handled gracefully
When a request resolves to an empty directory, the server SHALL return HTTP 200 with an HTML listing page that contains no entries (other than optional UI chrome).

#### Scenario: Empty directory
- **WHEN** a GET request is made for a directory that exists under `static_dir` but contains no files or subdirectories
- **THEN** the server responds with HTTP 200
- **THEN** the response body is valid HTML
- **THEN** no file entries are listed in the body

### Requirement: Directory read failure returns 500
When the server cannot read the directory contents (e.g. permission denied on `os.ReadDir`), the server SHALL return HTTP 500 with a plain-text error body.

#### Scenario: Unreadable directory
- **WHEN** a GET request is made for a directory that has no read permission
- **THEN** the server responds with HTTP 500
- **THEN** the response body contains the OS-level error message
