package generator

import (
	"time"

	cloudevents "github.com/cloudevents/sdk-go/v2"
	"github.com/google/uuid"
	"github.com/jaswdr/faker"
	"github.com/jittakal/kafeventproducer/internal/config"
	"github.com/jittakal/kafeventproducer/internal/events"
	"go.uber.org/zap"
)

// Generator generates fake library events
type Generator struct {
	config *config.GeneratorConfig
	faker  faker.Faker
	logger *zap.Logger
}

// NewGenerator creates a new event generator
func NewGenerator(config config.GeneratorConfig, logger *zap.Logger) *Generator {
	return &Generator{
		config: &config,
		faker:  faker.New(),
		logger: logger,
	}
}

// GenerateBookIssuedEvent generates a CloudEvent for a book issued scenario
func (g *Generator) GenerateBookIssuedEvent() cloudevents.Event {
	event := cloudevents.NewEvent()
	event.SetSpecVersion(cloudevents.VersionV1)
	event.SetID(uuid.New().String())
	event.SetType(events.EventTypeBookIssued)
	event.SetSource(events.EventSource)
	event.SetTime(time.Now())
	event.SetDataContentType(events.ContentTypeJSON)

	// Generate book data
	bookData := events.BookIssuedData{
		BookID:      g.generateBookID(),
		Title:       g.faker.Lorem().Sentence(5),
		ISBN:        g.generateISBN(),
		Author:      g.faker.Person().Name(),
		Category:    g.randomCategory(),
		MemberID:    g.generateMemberID(),
		MemberName:  g.faker.Person().Name(),
		MemberEmail: g.faker.Internet().Email(),
		IssueDate:   time.Now(),
		DueDate:     time.Now().Add(14 * 24 * time.Hour), // 2 weeks
		LibraryID:   g.generateLibraryID(),
		BranchName:  g.faker.Address().City() + " Branch",
	}

	if err := event.SetData(events.ContentTypeJSON, bookData); err != nil {
		g.logger.Error("Failed to set event data", zap.Error(err))
	}

	return event
}

// GenerateBookReturnedEvent generates a CloudEvent for a book returned scenario
func (g *Generator) GenerateBookReturnedEvent() cloudevents.Event {
	event := cloudevents.NewEvent()
	event.SetSpecVersion(cloudevents.VersionV1)
	event.SetID(uuid.New().String())
	event.SetType(events.EventTypeBookReturned)
	event.SetSource(events.EventSource)
	event.SetTime(time.Now())
	event.SetDataContentType(events.ContentTypeJSON)

	// Generate return data
	issueDate := time.Now().Add(-time.Duration(g.faker.IntBetween(7, 30)) * 24 * time.Hour)
	dueDate := issueDate.Add(14 * 24 * time.Hour)
	returnDate := time.Now()

	isLate := returnDate.After(dueDate)
	lateDays := 0
	lateFee := 0.0

	if isLate {
		lateDays = int(returnDate.Sub(dueDate).Hours() / 24)
		lateFee = float64(lateDays) * 0.50 // $0.50 per day
	}

	bookData := events.BookReturnedData{
		BookID:        g.generateBookID(),
		Title:         g.faker.Lorem().Sentence(5),
		ISBN:          g.generateISBN(),
		MemberID:      g.generateMemberID(),
		MemberName:    g.faker.Person().Name(),
		IssueDate:     issueDate,
		ReturnDate:    returnDate,
		DueDate:       dueDate,
		IsLate:        isLate,
		LateDays:      lateDays,
		LateFeeAmount: lateFee,
		LibraryID:     g.generateLibraryID(),
		BranchName:    g.faker.Address().City() + " Branch",
		Condition:     g.randomCondition(),
	}

	if err := event.SetData(events.ContentTypeJSON, bookData); err != nil {
		g.logger.Error("Failed to set event data", zap.Error(err))
	}

	return event
}

// Helper functions for generating realistic data

func (g *Generator) generateBookID() string {
	return "B" + g.faker.UUID().V4()[0:8]
}

func (g *Generator) generateMemberID() string {
	return "M" + g.faker.UUID().V4()[0:8]
}

func (g *Generator) generateLibraryID() string {
	return "LIB" + g.faker.UUID().V4()[0:6]
}

func (g *Generator) generateISBN() string {
	// Generate a simple ISBN-13 format
	return "978-" + g.faker.RandomStringWithLength(10)
}

func (g *Generator) randomCategory() string {
	categories := []string{
		"Fiction",
		"Non-Fiction",
		"Science",
		"Technology",
		"History",
		"Biography",
		"Mystery",
		"Thriller",
		"Romance",
		"Fantasy",
		"Self-Help",
		"Business",
		"Programming",
		"Art",
		"Philosophy",
	}
	return categories[g.faker.IntBetween(0, len(categories)-1)]
}

func (g *Generator) randomCondition() string {
	conditions := []string{"good", "fair", "damaged"}
	weights := []int{70, 25, 5} // 70% good, 25% fair, 5% damaged

	rand := g.faker.IntBetween(1, 100)
	cumulative := 0

	for i, weight := range weights {
		cumulative += weight
		if rand <= cumulative {
			return conditions[i]
		}
	}

	return conditions[0]
}
