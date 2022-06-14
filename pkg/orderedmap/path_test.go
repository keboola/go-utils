package orderedmap

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestPath_String(t *testing.T) {
	t.Parallel()
	assert.Equal(t, ``, (Path{}).String())
	assert.Equal(t, `foo`, (Path{MapStep(`foo`)}).String())
	assert.Equal(t, `[123]`, (Path{SliceStep(123)}).String())
	assert.Equal(t, `foo1.foo2[1][2].xyz`, (Path{MapStep(`foo1`), MapStep(`foo2`), SliceStep(1), SliceStep(2), MapStep(`xyz`)}).String())
}

func TestPathFromStr(t *testing.T) {
	t.Parallel()
	assert.Equal(t, Path{}, PathFromStr(``))
	assert.Equal(t, Path{MapStep(`foo`)}, PathFromStr(`foo`))
	assert.Equal(t, Path{SliceStep(123)}, PathFromStr(`[123]`))
	assert.Equal(t, Path{MapStep(`foo1`), MapStep(`foo2`), SliceStep(1), SliceStep(2), MapStep(`xyz`)}, PathFromStr(`foo1.foo2[1][2].xyz`))
}

func TestPath_Last(t *testing.T) {
	t.Parallel()
	assert.Equal(t, nil, (Path{}).Last())
	assert.Equal(t, MapStep(`foo`), (Path{MapStep(`foo`)}).Last())
	assert.Equal(t, SliceStep(1), (Path{SliceStep(1)}).Last())
	assert.Equal(t, MapStep(`foo2`), (Path{MapStep(`foo1`), SliceStep(1), MapStep(`foo2`)}).Last())
	assert.Equal(t, SliceStep(2), (Path{MapStep(`foo1`), SliceStep(1), MapStep(`foo2`), SliceStep(2)}).Last())
}

func TestPath_WithoutLast(t *testing.T) {
	t.Parallel()
	assert.Equal(t, Path(nil), (Path{}).WithoutLast())
	assert.Equal(t, Path{}, (Path{MapStep(`foo`)}).WithoutLast())
	assert.Equal(t, Path{}, (Path{SliceStep(1)}).WithoutLast())
	assert.Equal(t, Path{MapStep(`foo1`), SliceStep(1)}, (Path{MapStep(`foo1`), SliceStep(1), MapStep(`foo2`)}).WithoutLast())
	assert.Equal(t, Path{MapStep(`foo1`), SliceStep(1), MapStep(`foo2`)}, (Path{MapStep(`foo1`), SliceStep(1), MapStep(`foo2`), SliceStep(2)}).WithoutLast())
}
