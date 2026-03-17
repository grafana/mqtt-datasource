package plugin

import (
	"fmt"
	"sync"
	"testing"
)

func TestRefId_Concurrent(t *testing.T) {
	var wg sync.WaitGroup
	m := &RefIDs{}

	// Simulate concurrent access
	for i := range 100 {
		wg.Go(func() {
			topic := fmt.Sprintf("topic%d", i)
			refID := fmt.Sprintf("ref%d", i)
			m.Set(topic, refID)

			if got, exists := m.Get(topic); !exists || got != refID {
				t.Errorf("Expected refID %s for topic %s but got %s", refID, topic, got)
			}
		})
	}
	wg.Wait()
}
