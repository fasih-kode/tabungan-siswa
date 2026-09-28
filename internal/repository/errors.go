package repository

import "errors"

var (
	// ErrNotFound menunjukkan bahwa data yang dicari tidak ditemukan.
	ErrNotFound = errors.New("repository: not found")

	// ErrConflict menunjukkan bahwa operasi ditolak karena
	// terjadi konflik dengan data atau constraint yang sudah ada.
	ErrConflict = errors.New("repository: conflict")
)
