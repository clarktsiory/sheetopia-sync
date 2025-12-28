package database

import "slices"

type FileType string

const (
	FileTypePDF FileType = "pdf"
)

func (f FileType) Valid() bool {
	return slices.Contains([]FileType{
		FileTypePDF,
	}, f)
}
