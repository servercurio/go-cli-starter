package pool

import (
	"io"
	"testing"
	"time"

	"github.com/rs/zerolog"
	asrt "github.com/stretchr/testify/assert"
)

func TestDefaultConfig_SaneValues(t *testing.T) {
	assert := asrt.New(t)

	cfg := DefaultConfig()
	assert.Greater(cfg.Size, 0, "size should default to NumCPU()*2 > 0")
	assert.Equal(time.Minute, cfg.ExpiryDuration)
	assert.False(cfg.NonBlocking)
	assert.False(cfg.PreAlloc)
	assert.Equal(0, cfg.MaxBlockingTasks)
}

func TestConfig_FromEnv(t *testing.T) {
	assert := asrt.New(t)

	t.Setenv("APP_POOL_SIZE", "32")
	t.Setenv("APP_POOL_NON_BLOCKING", "true")
	t.Setenv("APP_POOL_EXPIRY_DURATION", "30s")
	t.Setenv("APP_POOL_PRE_ALLOC", "true")
	t.Setenv("APP_POOL_MAX_BLOCKING_TASKS", "100")

	cfg := DefaultConfig()
	cfg.FromEnv("APP_POOL")

	assert.Equal(32, cfg.Size)
	assert.True(cfg.NonBlocking)
	assert.Equal(30*time.Second, cfg.ExpiryDuration)
	assert.True(cfg.PreAlloc)
	assert.Equal(100, cfg.MaxBlockingTasks)
}

func TestConfig_FromEnv_LeavesUnsetValuesAlone(t *testing.T) {
	assert := asrt.New(t)

	cfg := &Config{
		Size:             7,
		ExpiryDuration:   2 * time.Minute,
		MaxBlockingTasks: 4,
	}
	// No env vars set.
	cfg.FromEnv("APP_POOL")

	assert.Equal(7, cfg.Size)
	assert.Equal(2*time.Minute, cfg.ExpiryDuration)
	assert.Equal(4, cfg.MaxBlockingTasks)
}

func TestConfig_Validate(t *testing.T) {
	assert := asrt.New(t)

	assert.NoError((*Config)(nil).Validate(), "nil config validates")

	assert.NoError(DefaultConfig().Validate(), "defaults validate")

	bad := &Config{ExpiryDuration: -1 * time.Second}
	err := bad.Validate()
	assert.Error(err)
	assert.Contains(err.Error(), "expiryDuration")

	bad2 := &Config{MaxBlockingTasks: -1}
	err = bad2.Validate()
	assert.Error(err)
	assert.Contains(err.Error(), "maxBlockingTasks")
}

func TestConfig_MarshalZerologObject_Smoke(t *testing.T) {
	// Render through an enabled logger writing to io.Discard so the
	// marshaller actually executes (zerolog.Nop disables the event and
	// would short-circuit EmbedObject).
	cfg := DefaultConfig()
	logger := zerolog.New(io.Discard)
	asrt.NotPanics(t, func() {
		logger.Info().EmbedObject(cfg).Send()
	})
}
