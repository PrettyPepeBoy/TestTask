package endpoint

import "testTask/internal/database"

type HttpHandler struct {
	storage *database.Database
}

func NewHttpHandler(db *database.Database) *HttpHandler {
	return &HttpHandler{
		storage: db,
	}
}
