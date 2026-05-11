package tts

import (
	"bytes"
	"encoding/binary"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestWrapWAV_HasRIFFHeader(t *testing.T) {
	pcm := make([]byte, 100)
	wav := wrapWAV(pcm)

	assert.Equal(t, "RIFF", string(wav[0:4]))
	assert.Equal(t, "WAVE", string(wav[8:12]))
	assert.Equal(t, "fmt ", string(wav[12:16]))
	assert.Equal(t, "data", string(wav[36:40]))

	var dataSize uint32
	require := assert.New(t)
	require.NoError(binary.Read(bytes.NewReader(wav[40:44]), binary.LittleEndian, &dataSize))
	assert.Equal(t, uint32(100), dataSize)
	assert.Len(t, wav, 144)
}
