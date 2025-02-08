package routes

import (
	"errors"
	"lifeSync/internal/models"
	"log"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

type EventResponse struct {
	ID    uint      `json:"id"`
	Title string    `json:"title"`
	Start time.Time `json:"start"`
	End   time.Time `json:"end"`
	Color string    `json:"color"`
}

func CreateCalendarEvent(db *gorm.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		var payload struct {
			Title string    `json:"title"`
			Start time.Time `json:"start"`
			End   time.Time `json:"end"`
			Color string    `json:"color"`
		}

		if err := c.ShouldBindJSON(&payload); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}

		if payload.Title == "" {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Title is required"})
			return
		}

		if payload.Start.IsZero() || payload.End.IsZero() {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Start and End times are required"})
			return
		}

		if payload.Start.After(payload.End) {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Start time must be before End time"})
			return
		}

		claims, err := getUserClaimsFromCookie(c)
		if err != nil {
			c.JSON(http.StatusUnauthorized, gin.H{"error": err.Error()})
			return
		}

		userID, ok := claims["userid"].(float64)
		if !ok {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "userid not found in token claims", "claims": claims})
			return
		}

		newEvent := models.Event{
			Title:  payload.Title,
			Start:  payload.Start,
			End:    payload.End,
			Color:  payload.Color,
			Userid: uint(userID),
		}

		result := db.Create(&newEvent)
		if result.Error != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": result.Error.Error()})
			return
		}

		response := EventResponse{
			ID:    newEvent.ID,
			Title: newEvent.Title,
			Start: newEvent.Start,
			End:   newEvent.End,
			Color: newEvent.Color,
		}

		c.JSON(http.StatusCreated, gin.H{"event": response})
	}
}

func UpdateCalendarEvent(db *gorm.DB) gin.HandlerFunc {
	return func(c *gin.Context) {

		var payload struct {
			EventID uint      `json:"id"`
			Title   string    `json:"title"`
			Start   time.Time `json:"start"`
			End     time.Time `json:"end"`
		}

		if err := c.ShouldBindJSON(&payload); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}

		if payload.Title == "" {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Title is required"})
			return
		}

		if payload.Start.IsZero() || payload.End.IsZero() {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Start and End times are required"})
			return
		}

		if payload.Start.After(payload.End) {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Start time must be before End time"})
			return
		}

		claims, err := getUserClaimsFromCookie(c)
		if err != nil {
			c.JSON(http.StatusUnauthorized, gin.H{"error": err.Error()})
			return
		}

		userID, ok := claims["userid"].(float64)
		if !ok {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "userid not found in token claims", "claims": claims})
			return
		}

		var updatedEvent models.Event
		result := db.First(&updatedEvent, "id = ? AND userid = ?", payload.EventID, userID)
		if result.Error != nil {
			c.JSON(http.StatusNotFound, gin.H{"error": "No event with that ID exists"})
			return
		}

		updatedEvent.Title = payload.Title
		updatedEvent.Start = payload.Start
		updatedEvent.End = payload.End

		db.Save(&updatedEvent)

		response := EventResponse{
			ID:    updatedEvent.ID,
			Title: updatedEvent.Title,
			Start: updatedEvent.Start,
			End:   updatedEvent.End,
		}

		c.JSON(http.StatusOK, gin.H{"event": response})
	}
}

func GetCalendarEvents(db *gorm.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		claims, err := getUserClaimsFromCookie(c)
		if err != nil {
			c.JSON(http.StatusUnauthorized, gin.H{"error": err.Error()})
			return
		}

		userID, ok := claims["userid"].(float64)
		if !ok {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "userid not found in token claims"})
			return
		}

		var events []EventResponse

		result := db.Model(&models.Event{}).
			Select(`id, title, start, "end", color`).
			Where("userid = ?", uint(userID)).
			Find(&events)

		if result.Error != nil {
			log.Printf("Error fetching events: %v", result.Error)
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch events"})
			return
		}

		c.JSON(http.StatusOK, gin.H{"event": events})
	}
}

func DeleteCalendarEvent(db *gorm.DB) gin.HandlerFunc {
	return func(c *gin.Context) {

		eventID := c.Param("id")

		if eventID == "" {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Event ID is required"})
			return
		}

		claims, err := getUserClaimsFromCookie(c)
		if err != nil {
			c.JSON(http.StatusUnauthorized, gin.H{"error": err.Error()})
			return
		}

		userID, ok := claims["userid"].(float64)
		if !ok {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "userid not found in token claims"})
			return
		}

		var event models.Event
		result := db.First(&event, "id = ? AND userid = ?", eventID, uint(userID))
		if result.Error != nil {
			if errors.Is(result.Error, gorm.ErrRecordNotFound) {
				c.JSON(http.StatusNotFound, gin.H{"error": "Event not found or access denied"})
			} else {
				c.JSON(http.StatusInternalServerError, gin.H{"error": result.Error.Error()})
			}
			return
		}

		result = db.Delete(&event)
		if result.Error != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": result.Error.Error()})
			return
		}

		c.JSON(http.StatusOK, gin.H{"message": "Event deleted successfully"})
	}
}
