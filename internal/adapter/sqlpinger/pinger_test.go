package sqlpinger_test

import (
	"testing"

	"github.com/eduardo-pacheco-dev/api-stations-and-radio-link/internal/adapter/sqlpinger"
)

func TestNew_Driver(t *testing.T) {
	t.Parallel()

	pinger := sqlpinger.New(nil, "mysql")

	if got := pinger.Driver(); got != "mysql" {
		t.Errorf("Driver() = %q, want %q", got, "mysql")
	}
}
