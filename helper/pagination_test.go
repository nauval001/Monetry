package helper_test

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"monetry-api/helper"
)

// TestEncodeDecodeCursor memastikan cursor paginasi bisa di-encode dan
// dikembalikan ke nilai aslinya (round trip).
func TestEncodeDecodeCursor(t *testing.T) {
	date := time.Date(2026, time.November, 18, 0, 0, 0, 0, time.UTC)
	const id int64 = 42

	cursor := helper.EncodeCursor(date, id)
	require.NotEmpty(t, cursor)

	decodedDate, decodedID, err := helper.DecodeCursor(cursor)
	require.NoError(t, err)

	assert.True(t, date.Equal(decodedDate), "tanggal hasil decode harus sama")
	assert.Equal(t, id, decodedID)
}

func TestDecodeCursorInvalid(t *testing.T) {
	_, _, err := helper.DecodeCursor("cursor-tidak-valid-@@@")
	assert.Error(t, err)
}
