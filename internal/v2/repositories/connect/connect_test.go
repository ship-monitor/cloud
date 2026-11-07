package connect_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
	storage "github.com/ship-monitor/cloud/internal/v2/adapters/db"
	"github.com/ship-monitor/cloud/internal/v2/domain/device"
	"github.com/ship-monitor/cloud/internal/v2/repositories/connect"
	_ "modernc.org/sqlite"
)

func newTestDB(t *testing.T) *sqlx.DB {
	t.Helper()

	db, err := storage.New(
		&storage.Config{
			Driver: "sqlite",
			DSN:    ":memory:",
		},
	)
	if err != nil {
		t.Fatalf("open SQLite: %v", err)
	}

	db.SetMaxOpenConns(1)
	t.Cleanup(func() {
		if err := db.Close(); err != nil {
			t.Errorf("close SQLite: %v", err)
		}
	})

	return db
}

func request(deviceID uuid.UUID, approved bool) device.ConnectRequest {
	return device.ConnectRequest{
		ID:       uuid.New(),
		DeviceID: deviceID,
		CreatedAt: time.Date(
			2026,
			11,
			7,
			12,
			34,
			56,
			123456789,
			time.FixedZone("test offset", 5*60*60+30*60),
		),
		Approved: approved,
	}
}

func assertRequest(t *testing.T, got, want device.ConnectRequest) {
	t.Helper()

	if got.ID != want.ID || got.DeviceID != want.DeviceID ||
		got.CreatedAt.Format(
			connect.TimeFormat,
		) != want.CreatedAt.Format(
			connect.TimeFormat,
		) || got.Approved != want.Approved {
		t.Errorf("request = %+v, want %+v", got, want)
	}
}

func mustCreate(t *testing.T, repo *connect.Repo, req device.ConnectRequest) {
	t.Helper()

	if err := repo.Create(context.Background(), req); err != nil {
		t.Fatalf("create request %s: %v", req.ID, err)
	}
}

func mustGet(
	t *testing.T,
	repo *connect.Repo,
	id uuid.UUID,
) device.ConnectRequest {
	t.Helper()

	req, err := repo.Get(context.Background(), id)
	if err != nil {
		t.Fatalf("get request %s: %v", id, err)
	}

	return req
}

func TestCreateGetAndDuplicate(t *testing.T) {
	t.Parallel()
	repo := connect.New(newTestDB(t))
	ctx := context.Background()
	deviceID := uuid.New()

	for _, approved := range []bool{false, true} {
		want := request(deviceID, approved)
		mustCreate(t, repo, want)
		assertRequest(t, mustGet(t, repo, want.ID), want)

		duplicate := request(uuid.New(), !approved)

		duplicate.ID = want.ID
		if err := repo.Create(ctx, duplicate); err == nil {
			t.Errorf("creating duplicate ID %s succeeded", want.ID)
		}

		assertRequest(t, mustGet(t, repo, want.ID), want)
	}

	if _, err := repo.Get(
		ctx,
		uuid.New(),
	); !errors.Is(
		err,
		connect.ErrNotFound,
	) {
		t.Errorf("get missing request: got %v, want ErrNotFound", err)
	}
}

func TestListByDevice(t *testing.T) {
	t.Parallel()
	repo := connect.New(newTestDB(t))
	ctx := context.Background()
	deviceID, otherDeviceID := uuid.New(), uuid.New()
	first, second := request(deviceID, false), request(deviceID, true)

	other := request(otherDeviceID, true)
	for _, req := range []device.ConnectRequest{first, second, other} {
		mustCreate(t, repo, req)
	}

	got, err := repo.ListByDevice(ctx, deviceID)
	if err != nil {
		t.Fatalf("list requests: %v", err)
	}

	want := map[uuid.UUID]device.ConnectRequest{
		first.ID:  first,
		second.ID: second,
	}
	if len(got) != len(want) {
		t.Errorf("list returned %d requests, want %d", len(got), len(want))
	}

	for _, req := range got {
		expected, ok := want[req.ID]
		if !ok {
			t.Errorf("list returned unexpected or duplicate request: %+v", req)

			continue
		}

		assertRequest(t, req, expected)
		delete(want, req.ID)
	}

	if len(want) != 0 {
		t.Errorf("list omitted requests: %v", want)
	}

	got, err = repo.ListByDevice(ctx, otherDeviceID)
	if err != nil {
		t.Fatalf("list other device: %v", err)
	}

	if len(got) != 1 {
		t.Errorf("list other device returned %d requests, want 1", len(got))
	} else {
		assertRequest(t, got[0], other)
	}

	got, err = repo.ListByDevice(ctx, uuid.New())
	if err != nil || len(got) != 0 {
		t.Errorf(
			"list unknown device: got %v, error %v; want empty list and nil error",
			got,
			err,
		)
	}
}

func TestSetApprovedAndDelete(t *testing.T) {
	t.Parallel()
	repo := connect.New(newTestDB(t))
	ctx := context.Background()
	want := request(uuid.New(), false)
	mustCreate(t, repo, want)

	for _, approved := range []bool{true, true, false, false} {
		if err := repo.SetApproved(
			ctx,
			want.ID,
			want.DeviceID,
			approved,
		); err != nil {
			t.Fatalf("set approved=%t: %v", approved, err)
		}

		want.Approved = approved
		assertRequest(t, mustGet(t, repo, want.ID), want)
	}

	if err := repo.SetApproved(
		ctx,
		want.ID,
		uuid.New(),
		true,
	); !errors.Is(
		err,
		connect.ErrNotFound,
	) {
		t.Errorf("set approved for wrong device: got %v, want ErrNotFound", err)
	}

	assertRequest(t, mustGet(t, repo, want.ID), want)

	if err := repo.SetApproved(
		ctx,
		uuid.New(),
		want.DeviceID,
		true,
	); !errors.Is(
		err,
		connect.ErrNotFound,
	) {
		t.Errorf(
			"set approved for missing request: got %v, want ErrNotFound",
			err,
		)
	}

	if err := repo.Delete(ctx, want.ID); err != nil {
		t.Fatalf("delete request: %v", err)
	}

	if _, err := repo.Get(ctx, want.ID); !errors.Is(err, connect.ErrNotFound) {
		t.Errorf("get deleted request: got %v, want ErrNotFound", err)
	}

	if err := repo.Delete(ctx, want.ID); !errors.Is(err, connect.ErrNotFound) {
		t.Errorf("delete missing request: got %v, want ErrNotFound", err)
	}
}

func TestWithTxRollback(t *testing.T) {
	t.Parallel()
	db := newTestDB(t)
	repo := connect.New(db)
	ctx := context.Background()
	deviceID := uuid.New()
	updated, deleted := request(deviceID, false), request(deviceID, true)
	mustCreate(t, repo, updated)
	mustCreate(t, repo, deleted)

	tx, err := db.BeginTxx(ctx, nil) //nolint:varnamelen
	if err != nil {
		t.Fatalf("begin transaction: %v", err)
	}

	t.Cleanup(func() { _ = tx.Rollback() })

	txRepo := repo.WithTx(tx)
	created := request(deviceID, true)
	mustCreate(t, txRepo, created)
	assertRequest(t, mustGet(t, txRepo, created.ID), created)

	if err := txRepo.SetApproved(ctx, updated.ID, deviceID, true); err != nil {
		t.Fatalf("set approved in transaction: %v", err)
	}

	inside := updated
	inside.Approved = true
	assertRequest(t, mustGet(t, txRepo, updated.ID), inside)

	if err := txRepo.Delete(ctx, deleted.ID); err != nil {
		t.Fatalf("delete in transaction: %v", err)
	}

	if _, err := txRepo.Get(
		ctx,
		deleted.ID,
	); !errors.Is(
		err,
		connect.ErrNotFound,
	) {
		t.Errorf(
			"get deleted request in transaction: got %v, want ErrNotFound",
			err,
		)
	}

	listed, err := txRepo.ListByDevice(ctx, deviceID)
	if err != nil {
		t.Fatalf("list in transaction: %v", err)
	}

	if len(listed) != 2 {
		t.Errorf(
			"list in transaction returned %d requests, want 2",
			len(listed),
		)
	}

	seen := make(map[uuid.UUID]device.ConnectRequest, len(listed))
	for _, req := range listed {
		seen[req.ID] = req
	}

	for _, want := range []device.ConnectRequest{inside, created} {
		if got, ok := seen[want.ID]; ok {
			assertRequest(t, got, want)
		} else {
			t.Errorf("list in transaction omitted %s", want.ID)
		}
	}

	if err := tx.Rollback(); err != nil {
		t.Fatalf("rollback: %v", err)
	}

	if _, err := repo.Get(
		ctx,
		created.ID,
	); !errors.Is(
		err,
		connect.ErrNotFound,
	) {
		t.Errorf("get rolled-back creation: got %v, want ErrNotFound", err)
	}

	assertRequest(t, mustGet(t, repo, updated.ID), updated)
	assertRequest(t, mustGet(t, repo, deleted.ID), deleted)
	mustCreate(t, repo, request(deviceID, false))
}

//nolint:wrapcheck
func TestCanceledContext(t *testing.T) {
	t.Parallel()

	repo := connect.New(newTestDB(t))
	want := request(uuid.New(), false)
	mustCreate(t, repo, want)

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	newRequest := request(want.DeviceID, true)

	checks := []struct {
		name string
		run  func() error
	}{
		{"Create", func() error { return repo.Create(ctx, newRequest) }},
		{"Get", func() error {
			_, err := repo.Get(ctx, want.ID)

			return err
		}},
		{"ListByDevice", func() error {
			_, err := repo.ListByDevice(ctx, want.DeviceID)

			return err
		}},
		{
			"SetApproved",
			func() error { return repo.SetApproved(ctx, want.ID, want.DeviceID, true) },
		},
		{"Delete", func() error { return repo.Delete(ctx, want.ID) }},
	}
	for _, check := range checks {
		t.Run(check.name, func(t *testing.T) {
			t.Parallel()

			if err := check.run(); !errors.Is(err, context.Canceled) {
				t.Errorf("got %v, want context.Canceled", err)
			}
		})
	}

	assertRequest(t, mustGet(t, repo, want.ID), want)

	if _, err := repo.Get(
		context.Background(),
		newRequest.ID,
	); !errors.Is(
		err,
		connect.ErrNotFound,
	) {
		t.Errorf("get canceled creation: got %v, want ErrNotFound", err)
	}
}
