package upgdebug

import (
	"context"
	"database/sql"
	"os"
	"time"

	"uw/upg"

	"uw/ulog"
)

type Option func(*QueryHook)

type Logger interface {
	Printf(format string, v ...interface{})
}

// WithEnabled enables/disables the hook.
func WithEnabled(on bool) Option {
	return func(h *QueryHook) {
		h.enabled = on
	}
}

func WithLogger(logger Logger) Option {
	return func(h *QueryHook) {
		h.log = logger
	}
}

func WithColor(on bool) Option {
	return func(h *QueryHook) {
		h.color = on
	}
}

// WithVerbose configures the hook to log all queries
// (by default, only failed queries are logged).
func WithVerbose(on bool) Option {
	return func(h *QueryHook) {
		h.verbose = on
	}
}

// EmptyLine adds an empty line before each query.
func WithEmptyLine(on bool) Option {
	return func(h *QueryHook) {
		h.emptyLine = on
	}
}

// FromEnv configures the hook using the environment variable value.
// For example, WithEnv("UPGDEBUG"):
//   - UPGDEBUG=0 - disables the hook.
//   - UPGDEBUG=1 - enables the hook.
//   - UPGDEBUG=2 - enables the hook and verbose mode.
func FromEnv(keys ...string) Option {
	if len(keys) == 0 {
		keys = []string{"UPGDEBUG"}
	}
	return func(h *QueryHook) {
		for _, key := range keys {
			if env, ok := os.LookupEnv(key); ok {
				h.enabled = env != "" && env != "0"
				h.verbose = env == "2"
				break
			}
		}
	}
}

type QueryHook struct {
	enabled   bool
	verbose   bool
	emptyLine bool
	color     bool
	log       Logger
}

var _ upg.QueryHook = (*QueryHook)(nil)

func NewQueryHook(opts ...Option) *QueryHook {
	h := &QueryHook{
		enabled: true,
		color:   true,
		log:     ulog.GlobalLogger(),
	}
	for _, opt := range opts {
		opt(h)
	}
	return h
}

func (h *QueryHook) BeforeQuery(
	ctx context.Context, event *upg.QueryEvent,
) (context.Context, error) {
	return ctx, nil
}

func (h *QueryHook) emptyLineParse() string {
	if h.emptyLine {
		return "\r\n"
	}

	return ""
}

func (h *QueryHook) AfterQuery(ctx context.Context, evt *upg.QueryEvent) error {
	if !h.enabled {
		return nil
	}

	if !h.verbose {
		switch evt.Err {
		case nil, sql.ErrNoRows, sql.ErrTxDone:
			return nil
		}
	}

	if evt.Err != nil {
		h.log.Printf("[upg] [%s] [%s] %s\r\n"+h.emptyLineParse(),
			time.Since(evt.StartTime), h.ansiCode(ulog.ANSI.Red, evt.Err.Error()), evt.Query)
		return nil
	}

	if evt.Result != nil {
		h.log.Printf("[upg] [%s] (%d/%d) %s"+h.emptyLineParse(),
			time.Since(evt.StartTime), evt.Result.RowsAffected(),
			evt.Result.RowsReturned(), evt.Query)
		return nil
	}

	h.log.Printf("[upg] [%s] %s"+h.emptyLineParse(),
		time.Since(evt.StartTime), evt.Query)
	return nil
}

func (h *QueryHook) ansiCode(code string, s string) string {
	if !h.color {
		return s
	}

	return ulog.SetANSI(code, s)
}
