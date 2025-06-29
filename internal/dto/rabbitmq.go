package dto

import (
	"time"

	"github.com/google/uuid"
)

type MQPostCreatedMsg struct {
	PostID    int64     `json:"post_id"`
	UserID    uuid.UUID `json:"user_id"`
	PostTitle string    `json:"post_title"`
	CreatedAt time.Time `json:"created_at"`
}

type MQPostValidationStatusUpdateMsg struct {
	PostID    int64      `json:"post_id"`
	UserID    uuid.UUID  `json:"user_id"`
	StatusMsg string     `json:"status_msg"`
}
