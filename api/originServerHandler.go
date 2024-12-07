package api

import (
	"os"

	"github.com/gofiber/fiber/v2"
	"github.com/zhitoo/cdn/models"
	"github.com/zhitoo/cdn/requests"
)

func (s *APIServer) registerOriginServer(c *fiber.Ctx) error {
	payload := new(requests.RegisterOriginServerRequest)

	if err := c.BodyParser(payload); err != nil {
		return err
	}
	// Validation
	errs := s.validator.Validate(payload)
	if errs != nil {
		return c.Status(422).JSON(errs)
	}

	// Authenticate the request using APIKey
	expectedAPIKey := os.Getenv("API_KEY")
	if payload.APIKey != expectedAPIKey {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
			"error": "Unauthorized",
		})
	}

	origin, _ := s.storage.GetOriginServerBySiteIdentifier(payload.SiteIdentifier)

	if origin.ID == 0 {
		//must register it
		origin = &models.OriginServer{
			SiteIdentifier: payload.SiteIdentifier,
			OriginURL:      payload.OriginURL,
		}
		_, err := s.storage.CreateOriginServer(origin)
		if err != nil {
			return err
		}
	}

	return c.JSON(fiber.Map{
		"message": "Origin server registered successfully",
	})
}
