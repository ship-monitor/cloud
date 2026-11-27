//go:build integration

package di

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net"
	"net/http"
	"sync"
	"testing"
	"time"

	"github.com/lerenn/asyncapi-codegen/pkg/extensions"
	broker "github.com/lerenn/asyncapi-codegen/pkg/extensions/brokers/nats"
	"github.com/nats-io/nats.go"
	"github.com/ship-monitor/cloud/internal/v2/adapters/subscriber"
	"github.com/ship-monitor/cloud/pkg/connector/asyncapi"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/wait"
	"go.uber.org/fx"
	"go.uber.org/zap"
)

var subscriberSubjects = []string{
	"device.*.health",
	"device.*.connect.responses",
	"device.*.metrics.sensor",
	"device.*.metrics",
	"device.*.sensors",
	"device.*.firmware",
	"device.*.notifications",
}

func subscriberContainerRequest() testcontainers.ContainerRequest {
	return testcontainers.ContainerRequest{
		Image:        "nats:2-alpine",
		ExposedPorts: []string{"4222/tcp", "8222/tcp"},
		Cmd:          []string{"-m", "8222"},
		WaitingFor:   wait.ForListeningPort("4222/tcp"),
	}
}

func subscriberTestServer(t *testing.T) (string, string) {
	t.Helper()
	ctx := t.Context()
	request := testcontainers.GenericContainerRequest{
		Started:          true,
		ContainerRequest: subscriberContainerRequest(),
	}

	container, err := testcontainers.GenericContainer(ctx, request)
	if container != nil {
		testcontainers.CleanupContainer(t, container)
	}

	require.NoError(t, err)
	host, err := container.Host(ctx)
	require.NoError(t, err)
	natsPort, err := container.MappedPort(ctx, "4222/tcp")
	require.NoError(t, err)
	monitorPort, err := container.MappedPort(ctx, "8222/tcp")
	require.NoError(t, err)

	return "nats://" + net.JoinHostPort(host, natsPort.Port()),
		"http://" + net.JoinHostPort(host, monitorPort.Port())
}

type subscriberTestFixture struct {
	app        *fx.App
	subscriber *subscriber.S
	started    bool
	stopOnce   sync.Once
	stopped    chan struct{}
	stopErr    error
}

func (f *subscriberTestFixture) stop() {
	f.stopOnce.Do(func() {
		go func() {
			ctx, cancel := context.WithTimeout(
				context.Background(),
				5*time.Second,
			)
			defer cancel()

			f.stopErr = f.app.Stop(ctx)
			close(f.stopped)
		}()
	})
}

func subscriberTestApp(
	t *testing.T, url string, opts ...asyncapi.ControllerOption,
) *subscriberTestFixture {
	t.Helper()

	connection, err := broker.NewController(url)
	require.NoError(t, err)
	t.Cleanup(func() { connection.Close() })

	controller, err := asyncapi.NewAppController(connection, opts...)
	require.NoError(t, err)

	fixture := &subscriberTestFixture{stopped: make(chan struct{})}
	fixture.app = fx.New(fx.NopLogger, fx.Invoke(func(lc fx.Lifecycle) {
		fixture.subscriber = newSubscriber(
			lc, controller, zap.NewNop(), nil, nil, nil, nil, nil, nil,
		)
	}))
	require.NoError(t, fixture.app.Err())
	t.Cleanup(func() {
		if !fixture.started {
			// Fx does not run the stop hook for a failed start hook.
			ctx, cancel := context.WithTimeout(
				context.Background(),
				5*time.Second,
			)
			defer cancel()

			err := fixture.subscriber.Stop(ctx)
			if err != nil {
				t.Errorf("clean up partial subscription: %v", err)
			}

			return
		}

		fixture.stop()

		select {
		case <-fixture.stopped:
			if fixture.stopErr != nil {
				t.Errorf("stop subscriber: %v", fixture.stopErr)
			}
		case <-time.After(6 * time.Second):
			t.Error("subscriber cleanup did not finish")
		}
	})

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	err = fixture.app.Start(ctx)
	require.NoError(t, err)

	fixture.started = true

	return fixture
}

func subscribedSubjects(url string) (map[string]bool, error) {
	client := &http.Client{Timeout: time.Second}

	response, err := client.Get(url + "/connz?subs=1")
	if err != nil {
		return nil, err
	}
	defer response.Body.Close()

	if response.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("monitor status: %s", response.Status)
	}

	var result struct {
		Connections []struct {
			Subscriptions []string `json:"subscriptions_list"`
		} `json:"connections"`
	}
	if err := json.NewDecoder(response.Body).Decode(&result); err != nil {
		return nil, err
	}

	subjects := make(map[string]bool)

	for _, conn := range result.Connections {
		for _, subject := range conn.Subscriptions {
			subjects[subject] = true
		}
	}

	return subjects, nil
}

func waitForHeartbeatSubscription(t *testing.T, monitor string) {
	t.Helper()
	require.Eventually(t, func() bool {
		subjects, err := subscribedSubjects(monitor)

		return err == nil && subjects["device.*.health"]
	}, 5*time.Second, 20*time.Millisecond, "heartbeat subscription not visible")
}

func TestSubscriberLifecycleUnsubscribes(t *testing.T) {
	url, monitor := subscriberTestServer(t)
	fixture := subscriberTestApp(t, url)
	require.Eventually(t, func() bool {
		subjects, err := subscribedSubjects(monitor)
		if err != nil {
			return false
		}

		for _, subject := range subscriberSubjects {
			if !subjects[subject] {
				return false
			}
		}

		return true
	}, 5*time.Second, 20*time.Millisecond, "all seven subscriptions should be active")
	fixture.stop()

	select {
	case <-fixture.stopped:
		require.NoError(t, fixture.stopErr)
	case <-time.After(6 * time.Second):
		t.Fatal("subscriber did not stop")
	}

	assert.Eventually(t, func() bool {
		subjects, err := subscribedSubjects(monitor)
		if err != nil {
			return false
		}

		for _, subject := range subscriberSubjects {
			if subjects[subject] {
				return false
			}
		}

		return true
	}, 5*time.Second, 20*time.Millisecond, "all subscriptions should be removed on stop")
}

func releaseSubscriberMessage(
	t *testing.T, unblock func(), entered, exited <-chan struct{},
) {
	t.Helper()
	unblock()

	select {
	case <-entered:
		select {
		case <-exited:
		case <-time.After(5 * time.Second):
			t.Error("subscriber callback did not exit during cleanup")
		}
	default:
	}
}

func TestSubscriberShutdownWaitsForInFlightMessage(t *testing.T) {
	url, monitor := subscriberTestServer(t)
	entered := make(chan struct{})
	exited := make(chan struct{})
	release := make(chan struct{})

	var releaseOnce sync.Once

	unblock := func() { releaseOnce.Do(func() { close(release) }) }
	middleware := func(_ context.Context, _ *extensions.BrokerMessage, _ extensions.NextMiddleware) error {
		close(entered)
		defer close(exited)

		select {
		case <-release:
		case <-time.After(10 * time.Second):
		}

		return errors.New("test message completed")
	}
	fixture := subscriberTestApp(t, url, asyncapi.WithMiddlewares(middleware))
	t.Cleanup(func() { releaseSubscriberMessage(t, unblock, entered, exited) })
	waitForHeartbeatSubscription(t, monitor)

	publisher, err := nats.Connect(url)
	require.NoError(t, err)

	defer publisher.Close()

	require.NoError(t, publisher.Publish(
		"device.11111111-1111-1111-1111-111111111111.health",
		[]byte(`{"timestamp":"2026-01-01T00:00:00Z"}`),
	))
	require.NoError(t, publisher.FlushTimeout(time.Second))

	select {
	case <-entered:
	case <-time.After(5 * time.Second):
		t.Fatal("message did not reach the subscriber")
	}

	fixture.stop()

	select {
	case <-fixture.stopped:
		assert.Fail(
			t,
			"shutdown returned while a message was in flight",
			"%v",
			fixture.stopErr,
		)
	case <-exited:
		t.Fatal("in-flight middleware exited before release")
	case <-time.After(200 * time.Millisecond):
	}

	unblock()

	select {
	case <-exited:
	case <-time.After(5 * time.Second):
		t.Fatal("in-flight middleware did not exit")
	}

	select {
	case <-fixture.stopped:
		assert.NoError(t, fixture.stopErr)
	case <-time.After(6 * time.Second):
		t.Fatal("shutdown did not finish after the message completed")
	}
}

func TestSubscriberSlowDeviceDoesNotBlockAnother(t *testing.T) {
	url, monitor := subscriberTestServer(t)

	const (
		slowPayload = `{"timestamp":"2026-01-01T00:00:00Z"}`
		fastPayload = `{"timestamp":"2026-01-01T00:00:01Z"}`
		slowSubject = "device.11111111-1111-1111-1111-111111111111.health"
		fastSubject = "device.22222222-2222-2222-2222-222222222222.health"
	)

	slowEntered, slowExited := make(chan struct{}), make(chan struct{})
	fastEntered := make(chan struct{})
	release := make(chan struct{})

	var releaseOnce sync.Once

	unblock := func() { releaseOnce.Do(func() { close(release) }) }
	middleware := func(_ context.Context, msg *extensions.BrokerMessage, _ extensions.NextMiddleware) error {
		switch string(msg.Payload) {
		case slowPayload:
			close(slowEntered)
			defer close(slowExited)

			select {
			case <-release:
			case <-time.After(30 * time.Second):
			}
		case fastPayload:
			close(fastEntered)
		}

		return errors.New("test message completed")
	}
	subscriberTestApp(t, url, asyncapi.WithMiddlewares(middleware))
	t.Cleanup(func() {
		releaseSubscriberMessage(t, unblock, slowEntered, slowExited)
	})
	waitForHeartbeatSubscription(t, monitor)

	publisher, err := nats.Connect(url)
	require.NoError(t, err)

	defer publisher.Close()

	observer, err := publisher.SubscribeSync(fastSubject)
	require.NoError(t, err)
	require.NoError(t, publisher.FlushTimeout(time.Second))
	require.NoError(t, publisher.Publish(slowSubject, []byte(slowPayload)))
	require.NoError(t, publisher.FlushTimeout(time.Second))

	select {
	case <-slowEntered:
	case <-time.After(5 * time.Second):
		t.Fatal("slow device message did not enter the subscriber")
	}

	require.NoError(t, publisher.Publish(fastSubject, []byte(fastPayload)))
	require.NoError(t, publisher.FlushTimeout(time.Second))
	_, err = observer.NextMsg(5 * time.Second)
	require.NoError(t, err, "NATS did not deliver the fast device message")

	select {
	case <-fastEntered:
		select {
		case <-slowExited:
			t.Fatal("slow message exited before release")
		default:
		}
	case <-slowExited:
		t.Fatal("slow message exited before release")
	case <-time.After(2 * time.Second):
		t.Error("fast device was blocked by another device's message")
	}

	unblock()

	select {
	case <-slowExited:
	case <-time.After(5 * time.Second):
		t.Fatal("slow message did not exit after release")
	}

	select {
	case <-fastEntered:
	case <-time.After(5 * time.Second):
		t.Fatal(
			"fast message was not handled even after releasing the slow one",
		)
	}
}
