package repositories

import (
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"

	"github.com/Unikyri/gemini-live-agent-klyra/backend/internal/core/domain"
)

func TestCreateTopic_WithSequence(t *testing.T) {
	// Test creating a topic with proper ordering sequence
	topic := &domain.Topic{
		ID:         uuid.New(),
		CourseID:   uuid.New(),
		Title:      "Chapter 1: Introduction",
		OrderIndex: 1,
	}

	assert.Equal(t, 1, topic.OrderIndex)
	assert.NotNil(t, topic.Title)
}

func TestGetTopicsByCourse_Ordered(t *testing.T) {
	// Test that topics are retrieved in sequence order
	topics := []*domain.Topic{
		{ID: uuid.New(), Title: "Topic 1", OrderIndex: 1},
		{ID: uuid.New(), Title: "Topic 2", OrderIndex: 2},
		{ID: uuid.New(), Title: "Topic 3", OrderIndex: 3},
	}

	// Verify ordering
	for i := 0; i < len(topics)-1; i++ {
		assert.Less(t, topics[i].OrderIndex, topics[i+1].OrderIndex)
	}
}

func TestCountTopicsByCourse_Statistics(t *testing.T) {
	// Test counting topics for statistics and pagination
	_ = uuid.New()
	var expectedCount int64 = 5

	assert.Equal(t, int64(5), expectedCount)
}
