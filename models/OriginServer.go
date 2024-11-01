package models

type OriginServer struct {
	ID             uint   `gorm:"primaryKey"`
	SiteIdentifier string `gorm:"uniqueIndex"`
	OriginURL      string
}
