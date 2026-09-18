package engine_test

import (
	"context"
	"testing"

	"github.com/jjw/docksnap/internal/engine"
	"github.com/stretchr/testify/require"
)

func Test_HelperImage_is_alpine_3_21(t *testing.T) {
	require.Equal(t, "docker.io/library/alpine:3.21", engine.HelperImage)
}

func Test_Fake_ImageInspect_missing_helper_image_contains_ref(t *testing.T) {
	ctx := context.Background()
	f := engine.NewFake()

	err := f.ImageInspect(ctx, engine.HelperImage)
	require.Error(t, err)
	require.Contains(t, err.Error(), engine.HelperImage)
}

func Test_Fake_ImagePull_then_ImageInspect_helper_image(t *testing.T) {
	ctx := context.Background()
	f := engine.NewFake()

	require.NoError(t, f.ImagePull(ctx, engine.HelperImage))
	require.NoError(t, f.ImageInspect(ctx, engine.HelperImage))
}

func Test_New_signature_is_Engine(t *testing.T) {
	var fn func() (engine.Engine, error) = engine.New
	require.NotNil(t, fn)
}
