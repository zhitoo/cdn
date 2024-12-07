package storage

import (
	"github.com/zhitoo/cdn/models"
	"golang.org/x/crypto/bcrypt"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

type Storage interface {
	GetUserByID(string) (*models.User, error)
	CreateUser(user *models.User) (*models.User, error)
	GetUserByUserName(userName string) (*models.User, error)
	GetOriginServerBySiteIdentifier(siteIdentifier string) (*models.OriginServer, error)
	CreateOriginServer(os *models.OriginServer) (*models.OriginServer, error)
}

func HashPassword(password string) (string, error) {
	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return "", err
	}
	return string(hash), nil
}

type SQLiteStorage struct {
	db *gorm.DB
}

func NewSQLiteStore() (*SQLiteStorage, error) {
	//dsn := "host=" + config.Envs.DBHost + " user=" + config.Envs.DBUser + " password=" + config.Envs.DBPassword + " dbname=" + config.Envs.DBName + " port=" + config.Envs.DBPort + " sslmode=disable"
	db, err := gorm.Open(sqlite.Open("storage/cdn.db"), &gorm.Config{})

	// Migrate the user schema
	db.AutoMigrate(&models.User{})
	db.AutoMigrate(&models.OriginServer{})

	if err != nil {
		return nil, err
	}

	return &SQLiteStorage{db: db}, nil
}

func (p *SQLiteStorage) GetUserByID(ID string) (*models.User, error) {
	user := &models.User{}
	result := p.db.Find(user, ID)
	return user, result.Error
}

func (p *SQLiteStorage) GetUserByUserName(userName string) (*models.User, error) {
	user := &models.User{}
	result := p.db.Take(user, "user_name = ?", userName)
	return user, result.Error
}

func (p *SQLiteStorage) CreateUser(user *models.User) (*models.User, error) {
	result := p.db.Create(user)
	return user, result.Error
}

func (p *SQLiteStorage) GetOriginServerBySiteIdentifier(siteIdentifier string) (*models.OriginServer, error) {
	os := &models.OriginServer{}
	result := p.db.Take(os, "site_identifier = ?", siteIdentifier)
	return os, result.Error
}

func (p *SQLiteStorage) CreateOriginServer(os *models.OriginServer) (*models.OriginServer, error) {
	result := p.db.Create(os)
	return os, result.Error
}
