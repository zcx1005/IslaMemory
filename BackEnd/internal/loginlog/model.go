package loginlog

import "time"

type LoginLog struct {
	ID               uint64    `gorm:"primaryKey;autoIncrement" json:"id"`
	UserID           *uint64   `gorm:"index" json:"user_id"`
	AccountSnapshot  string    `gorm:"size:64" json:"account_snapshot"`
	UsernameSnapshot string    `gorm:"size:64" json:"username_snapshot"`
	IP               string    `gorm:"size:64" json:"ip"`
	UserAgent        string    `gorm:"size:255" json:"user_agent"`
	LoginStatus      uint8     `gorm:"not null;default:1"  json:"login_status"`
	CreatedAt        time.Time `gorm:"index" json:"created_at"`
}

func (LoginLog) TableName() string {
	return "login_logs"
}
