package handoff

import (
	"os"
	"testing"

	"github.com/russellpope/spine/internal/hermetic"
)

func TestMain(m *testing.M) { os.Exit(hermetic.Main(m)) }
