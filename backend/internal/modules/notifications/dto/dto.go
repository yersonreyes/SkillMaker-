// Package dto contains the HTTP-layer data transfer objects for the notifications module.
// Names are unique across all modules/*/dto/ — verified via grep before adding.
package dto

import "time"

// NotificationResponse is the per-item DTO returned by the list endpoint.
// swagger:model NotificationResponse
type NotificationResponse struct {
	ID       string    `json:"id"`
	Tipo     string    `json:"tipo"`
	Titulo   string    `json:"titulo"`
	Cuerpo   string    `json:"cuerpo"`
	Leida    bool      `json:"leida"`
	RefID    string    `json:"refId"`
	CreadoEn time.Time `json:"creadoEn"`
}

// NotificationListResponse is the paginated list envelope for notifications.
// swagger:model NotificationListResponse
type NotificationListResponse struct {
	Items      []NotificationResponse `json:"items"`
	Page       int                    `json:"page"`
	Size       int                    `json:"size"`
	Total      int64                  `json:"total"`
	TotalPages int                    `json:"totalPages"`
}

// UnreadCountResponse is the response body for the unread-count endpoint.
// swagger:model UnreadCountResponse
type UnreadCountResponse struct {
	Unread int64 `json:"unread"`
}
