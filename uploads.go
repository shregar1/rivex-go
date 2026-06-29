package rivex

import (
	"bytes"
	"io"
)

// UploadFile represents an uploaded file from a multipart form request.
type UploadFile struct {
	// Filename is the original filename from the upload.
	Filename string

	// ContentType is the MIME type of the uploaded file.
	ContentType string

	// Headers contains the MIME headers from the multipart part.
	Headers map[string]string

	// Size is the size of the file in bytes.
	Size int64

	// data holds the file content in memory.
	data []byte

	// reader provides io.ReadSeeker access to the file data.
	reader *bytes.Reader
}

// Read reads up to len(p) bytes from the file. Implements io.Reader.
func (u *UploadFile) Read(p []byte) (int, error) {
	u.ensureReader()
	return u.reader.Read(p)
}

// Seek sets the offset for the next Read. Implements io.Seeker.
func (u *UploadFile) Seek(offset int64, whence int) (int64, error) {
	u.ensureReader()
	return u.reader.Seek(offset, whence)
}

// ReadAll reads and returns all bytes from the uploaded file.
func (u *UploadFile) ReadAll() ([]byte, error) {
	cp := make([]byte, len(u.data))
	copy(cp, u.data)
	return cp, nil
}

// WriteTo writes the file content to the given writer. Implements
// io.WriterTo.
func (u *UploadFile) WriteTo(w io.Writer) (int64, error) {
	n, err := w.Write(u.data)
	return int64(n), err
}

func (u *UploadFile) ensureReader() {
	if u.reader == nil {
		u.reader = bytes.NewReader(u.data)
	}
}

// NewUploadFile creates an UploadFile from raw data. This is primarily useful
// for testing.
func NewUploadFile(filename, contentType string, data []byte) *UploadFile {
	return &UploadFile{
		Filename:    filename,
		ContentType: contentType,
		Headers:     map[string]string{"Content-Type": contentType},
		Size:        int64(len(data)),
		data:        data,
	}
}
