package job

import (
	"context"
	"errors"
	"fmt"
	"testing"

	"github.com/riverqueue/river"
	"github.com/riverqueue/river/rivertype"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/skryfon-org/employee-360/backend/internal/domain/service"
)

type fakeEmail struct {
	got service.EmailMessage
	err error
}

func (f *fakeEmail) Send(_ context.Context, m service.EmailMessage) error {
	f.got = m
	return f.err
}

func run(f *fakeEmail) error {
	w := NewEmailWorker(f)
	return w.Work(context.Background(), &river.Job[SendEmailArgs]{
		JobRow: &rivertype.JobRow{},
		Args: SendEmailArgs{
			TemplateName: service.EmailTemplatePasswordReset,
			To:           "a@x.io",
			Subject:      "s",
			TemplateData: map[string]interface{}{"k": "v"},
		},
	})
}

func TestEmailWorker_Success(t *testing.T) {
	f := &fakeEmail{}
	require.NoError(t, run(f))
	assert.Equal(t, "a@x.io", f.got.To)
	assert.Equal(t, service.EmailTemplatePasswordReset, f.got.TemplateName)
	assert.Equal(t, "v", f.got.TemplateData["k"])
}

func TestEmailWorker_PermanentErrorCancels(t *testing.T) {
	err := run(&fakeEmail{err: fmt.Errorf("%w: bad", service.ErrInvalidEmailMessage)})
	var cancel *river.JobCancelError
	require.ErrorAs(t, err, &cancel)
}

func TestEmailWorker_TransientErrorRetries(t *testing.T) {
	boom := errors.New("smtp down")
	err := run(&fakeEmail{err: boom})
	require.ErrorIs(t, err, boom)
	var cancel *river.JobCancelError
	assert.False(t, errors.As(err, &cancel))
}
