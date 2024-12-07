package api

import (
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/golang-jwt/jwt/v5"
	"github.com/zhitoo/cdn/config"
	"github.com/zhitoo/cdn/models"
	"github.com/zhitoo/cdn/requests"
	"github.com/zhitoo/cdn/utils"
)

func (s *APIServer) handleLogin(c *fiber.Ctx) error {

	// username := c.FormValue("username")
	// password := c.FormValue("password")

	payload := new(requests.Login)

	if err := c.BodyParser(payload); err != nil {
		return err
	}

	user, _ := s.storage.GetUserByUserName(payload.UserName)

	if user.ID == 0 {
		return c.Status(fiber.StatusUnauthorized).JSON(ApiError{Message: "unauthorized"})
	}

	passwordIsOk, _ := utils.HashCompare(payload.Password, user.Password)

	if !passwordIsOk {
		return c.Status(fiber.StatusUnauthorized).JSON(ApiError{Message: "unauthorized"})
	}

	// Create the Claims
	claims := jwt.MapClaims{
		"username": payload.UserName,
		"exp":      time.Now().Add(time.Hour * 72).Unix(),
	}

	// Create token
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)

	// Generate encoded token and send it as response.
	t, err := token.SignedString([]byte(config.Envs.JWTSecretKey))
	if err != nil {
		return c.SendStatus(fiber.StatusInternalServerError)
	}

	return c.JSON(fiber.Map{"token": t})
}

func (s *APIServer) handleCreateUser(c *fiber.Ctx) error {
	payload := new(requests.RegisterUser)

	if err := c.BodyParser(payload); err != nil {
		return err
	}

	// Validation
	errs := s.validator.Validate(payload)
	if errs != nil {
		c.Status(422)
		return c.JSON(errs)
	}

	hash, err := utils.Encrypt(payload.Password)

	if err != nil {
		return err
	}

	user, _ := s.storage.GetUserByUserName(payload.UserName)

	if user.ID != 0 {
		return c.Status(400).JSON(ApiError{Message: "user exists"})
	}

	user = &models.User{
		LastName:  payload.LastName,
		FirstName: payload.FirstName,
		UserName:  payload.UserName,
		Password:  hash,
	}
	user, err = s.storage.CreateUser(user)
	if err != nil {
		return err
	}
	return c.JSON(user)
}

func (s *APIServer) handleGetUser(c *fiber.Ctx) error {
	jwtToken := c.Locals("user").(*jwt.Token)
	claims := jwtToken.Claims.(jwt.MapClaims)
	username := claims["username"].(string)
	user, _ := s.storage.GetUserByUserName(username)
	if user.ID == 0 {
		return c.Status(404).JSON(ApiError{Message: "user not exists"})
	}
	return c.JSON(user)
}
