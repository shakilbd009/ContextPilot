package briefing

import (
	"context"
	"os"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/rs/zerolog"
)

// mockMeetingSourceRepo is a minimal implementation of MeetingSourceRepo for testing.
type mockMeetingSourceRepo struct {
meetings []MeetingWithMemory
err      error
}

func (m *mockMeetingSourceRepo) ListCompletedMeetingsWithMemory(ctx context.Context, userID uuid.UUID) ([]MeetingWithMemory, error) {
	if m.err != nil {
		return nil, m.err
	}
	return m.meetings, nil
}

func (m *mockMeetingSourceRepo) GetMeetingParticipants(ctx context.Context, meetingID uuid.UUID) ([]Participant_email_org, error) {
	return nil, nil
}

// mockPoolForBriefing implements Pool for briefing tests.
type mockPoolForBriefing struct {
	upcomingMeeting *UpcomingMeetingBriefing
	upcomingErr     error
}

func (p *mockPoolForBriefing) Query(ctx context.Context, sql string, args ...any) (pgx.Rows, error) {
	return nil, nil
}
func (p *mockPoolForBriefing) QueryRow(ctx context.Context, sql string, args ...any) pgx.Row {
	return &mockRowForBriefing{err: p.upcomingErr, meeting: p.upcomingMeeting}
}
func (p *mockPoolForBriefing) Exec(ctx context.Context, sql string, args ...any) (pgconn.CommandTag, error) {
	return pgconn.NewCommandTag("SELECT 1"), nil
}
func (p *mockPoolForBriefing) Begin(ctx context.Context) (pgx.Tx, error) {
	return &mockTxForBriefing{}, nil
}

type mockRowForBriefing struct {
	err     error
	meeting *UpcomingMeetingBriefing
	// scanned tracks whether Scan was called
	scanned bool
}

func (r *mockRowForBriefing) Scan(dest ...any) error {
	if r.err != nil {
		return r.err
	}
	if len(dest) < 7 || r.meeting == nil {
		// Nothing to scan
		r.scanned = true
		return nil
	}
	// dest[0]=ID (uuid.UUID)
	if id, ok := dest[0].(*uuid.UUID); ok {
		*id = r.meeting.ID
	}
	// dest[1]=Title (string)
	if t, ok := dest[1].(*string); ok {
		*t = r.meeting.Title
	}
	// dest[2]=ScheduledStart (time.Time)
	if st, ok := dest[2].(*time.Time); ok {
		*st = r.meeting.ScheduledStart
	}
	// dest[3]=Description (*string)
	if d, ok := dest[3].(**string); ok {
		*d = r.meeting.Description
	}
	// dest[4]=ClientOrOrganization (*string)
	if c, ok := dest[4].(**string); ok {
		*c = r.meeting.ClientOrOrganization
	}
	// dest[5]=CreatedBy (uuid.UUID)
	if cb, ok := dest[5].(*uuid.UUID); ok {
		*cb = r.meeting.CreatedBy
	}
	// dest[6]=CreatedAt (time.Time)
	if ca, ok := dest[6].(*time.Time); ok {
		*ca = r.meeting.CreatedAt
	}
	r.scanned = true
	return nil
}

type mockTxForBriefing struct{}

func (tx *mockTxForBriefing) Begin(ctx context.Context) (pgx.Tx, error)         { return tx, nil }
func (tx *mockTxForBriefing) Commit(ctx context.Context) error                  { return nil }
func (tx *mockTxForBriefing) Rollback(ctx context.Context) error                { return nil }
func (tx *mockTxForBriefing) Query(ctx context.Context, sql string, args ...any) (pgx.Rows, error) {
	return nil, nil
}
func (tx *mockTxForBriefing) QueryRow(ctx context.Context, sql string, args ...any) pgx.Row {
	return &mockRowForBriefing{}
}
func (tx *mockTxForBriefing) Exec(ctx context.Context, sql string, args ...any) (pgconn.CommandTag, error) {
	return pgconn.NewCommandTag("SELECT 1"), nil
}
func (tx *mockTxForBriefing) CopyFrom(ctx context.Context, tableName pgx.Identifier, columnNames []string, rowSrc pgx.CopyFromSource) (int64, error) {
	return 0, nil
}
func (tx *mockTxForBriefing) SendBatch(ctx context.Context, b *pgx.Batch) pgx.BatchResults { return nil }
func (tx *mockTxForBriefing) LargeObjects() pgx.LargeObjects                         { return pgx.LargeObjects{} }
func (tx *mockTxForBriefing) Prepare(ctx context.Context, name, sql string) (*pgconn.StatementDescription, error) {
	return nil, nil
}
func (tx *mockTxForBriefing) Conn() *pgx.Conn { return nil }

func TestNewProductionBriefingService(t *testing.T) {
	svc := NewProductionBriefingService(nil, nil, nil)
	if svc == nil {
		t.Fatal("NewProductionBriefingService() returned nil")
	}
}

func TestProductionBriefingService_Generate_ReturnsNoPriorMemory(t *testing.T) {
	userID := uuid.New()
	meetingID := uuid.New()
	scheduled := time.Now().Add(1 * time.Hour).UTC()

	pool := &mockPoolForBriefing{
		upcomingMeeting: &UpcomingMeetingBriefing{
			ID:             meetingID,
			Title:          "Test Meeting",
			CreatedBy:        userID,
			ScheduledStart: scheduled,
		},
	}
	repo := &Repository{pool: pool}
	meetRepo := &mockMeetingSourceRepo{meetings: nil}
	logger := zerolog.New(os.Stdout).Level(zerolog.WarnLevel)
	svc := NewProductionBriefingService(repo, meetRepo, &logger)
	input := BriefingGeneratorInput{
		UpcomingMeetingID: meetingID,
	}
	content, result, sourceIDs, err := svc.Generate(context.Background(), input)
	if err != nil {
		t.Fatalf("Generate() error = %v, want nil", err)
	}
	if result != BriefingResultNoPriorMemory {
		t.Errorf("Generate() result = %v, want %v", result, BriefingResultNoPriorMemory)
	}
	if content == nil {
		t.Fatal("Generate() content = nil, want non-nil BriefingContent")
	}
	if content.NoPriorMemoryShell == nil {
		t.Error("Generate() content.NoPriorMemoryShell = nil, want non-nil")
	}
	if !content.NoPriorMemoryShell.Generated {
		t.Error("Generate() content.NoPriorMemoryShell.Generated = false, want true")
	}
	if len(sourceIDs) != 0 {
		t.Errorf("Generate() sourceIDs = %v, want empty slice", sourceIDs)
	}
}
