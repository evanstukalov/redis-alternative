package store

import "testing"

func TestBinarySearch(t *testing.T) {
	messages := StreamMessages{
		Messages: []StreamMessage{
			{
				ID: "1526985054069-0",
			},
			{
				ID: "1526985054075-0",
			},
			{
				ID: "1526985054079-0",
			},
		},
	}

	t.Log(binarySearch(messages, "1526985054075-0"))
}
