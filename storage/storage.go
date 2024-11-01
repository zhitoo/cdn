package storage

import (
	_ "github.com/lib/pq"
	"github.com/zhitoo/go-api/config"
	"github.com/zhitoo/go-api/models"
	"golang.org/x/crypto/bcrypt"
	"gorm.io/driver/postgres"
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

type PostgresStorage struct {
	db *gorm.DB
}

func NewPostgresStore() (*PostgresStorage, error) {
	dsn := "host=" + config.Envs.DBHost + " user=" + config.Envs.DBUser + " password=" + config.Envs.DBPassword + " dbname=" + config.Envs.DBName + " port=" + config.Envs.DBPort + " sslmode=disable"
	db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{})

	// Migrate the user schema
	db.AutoMigrate(&models.User{})
	db.AutoMigrate(&models.OriginServer{})

	if err != nil {
		return nil, err
	}

	return &PostgresStorage{db: db}, nil
}

func (p *PostgresStorage) GetUserByID(ID string) (*models.User, error) {
	user := &models.User{}
	result := p.db.Find(user, ID)
	return user, result.Error
}

func (p *PostgresStorage) GetUserByUserName(userName string) (*models.User, error) {
	user := &models.User{}
	result := p.db.Take(user, "user_name = ?", userName)
	return user, result.Error
}

func (p *PostgresStorage) CreateUser(user *models.User) (*models.User, error) {
	result := p.db.Create(user)
	return user, result.Error
}

func (p *PostgresStorage) GetOriginServerBySiteIdentifier(siteIdentifier string) (*models.OriginServer, error) {
	os := &models.OriginServer{}
	result := p.db.Take(os, "site_identifier = ?", siteIdentifier)
	return os, result.Error
}

func (p *PostgresStorage) CreateOriginServer(os *models.OriginServer) (*models.OriginServer, error) {
	result := p.db.Create(os)
	return os, result.Error
}
