package kafkasource

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

var (
	tTimeout = time.Duration(10 * time.Second)
)

func TestNewController(t *testing.T) {
	testCases := map[string]struct {
		inItems []item

		expectedCount int
	}{
		"no expired": {
			inItems: []item{
				staleItem(time.Second),
				staleItem(time.Second),
				staleItem(time.Second),
			},
			expectedCount: 3,
		},
		"one expired": {
			inItems: []item{
				staleItem(20 * time.Second),
				staleItem(time.Second),
				staleItem(time.Second),
			},
			expectedCount: 2,
		},
		"two expired": {
			inItems: []item{
				staleItem(20 * time.Second),
				staleItem(20 * time.Second),
				staleItem(time.Second),
			},
			expectedCount: 1,
		},
		"all expired": {
			inItems: []item{
				staleItem(20 * time.Second),
				staleItem(20 * time.Second),
				staleItem(20 * time.Second),
			},
			expectedCount: 0,
		},
	}

	for name, tc := range testCases {
		//nolint:scopelint
		t.Run(name, func(t *testing.T) {
			sl := &StaleList{
				items:   tc.inItems,
				timeout: tTimeout,
			}

			c := sl.Count()
			assert.Equal(t, tc.expectedCount, c)
		})
	}

}

func staleItem(age time.Duration) item {
	return item{
		object: nil,
		added:  time.Now().Add(age * -1),
	}
}
