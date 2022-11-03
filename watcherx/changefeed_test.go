// Copyright © 2022 Ory Corp
// SPDX-License-Identifier: Apache-2.0

package watcherx

import (
	"context"
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/cockroachdb/cockroach-go/v2/testserver"
	"github.com/google/uuid"
	_ "github.com/jackc/pgx/v4/stdlib"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/tidwall/gjson"

	"github.com/ory/x/logrusx"
)

// Run this test with
//
// docker run --name cloud_kratos_test_database_cockroach -p 3446:26257 -d cockroachdb/cockroach:v21.1.21 start-single-node --insecure
// export TEST_DATABASE_COCKROACHDB="cockroach://root@127.0.0.1:3446/defaultdb?sslmode=disable"
func TestChangeFeed(t *testing.T) {
	tableName := "t_" + strings.ReplaceAll(uuid.New().String(), "-", "")
	tableName = "asdfuhasdfuih"

	var watcherCount = 1
	var itemCount int = 5

	ctx := context.Background()
	l := logrusx.New("", "")
	db, err := testserver.NewTestServer()
	require.NoError(t, err)

	dsnp := db.PGURL()
	dsnp.Scheme = "cockroach"
	dsn := dsnp.String()

	cx, err := NewChangeFeedConnection(ctx, l, dsn)
	require.NoError(t, err)

	_, err = cx.Exec("CREATE TABLE IF NOT EXISTS " + tableName + " (id UUID PRIMARY KEY, value VARCHAR(64))")
	require.NoError(t, err)

	time.Sleep(time.Second)
	start := time.Now()

	ctx, cancel := context.WithCancel(context.Background())
	t.Cleanup(cancel)

	ctx, cancel = context.WithTimeout(ctx, time.Second*60)
	t.Cleanup(cancel)

	events := make(EventChannel)

	worker := func() {
		c, err := NewChangeFeedConnection(ctx, l, dsn)
		require.NoError(t, err)

		_, err = WatchChangeFeed(ctx, c, tableName, events, time.Now().Add(time.Minute))
		require.Error(t, err, "not able to watch changes from the future")

		_, err = WatchChangeFeed(ctx, c, tableName, events, start)
		require.NoError(t, err)
	}

	for i := 0; i < watcherCount; i++ {
		go worker()
	}

	rowsToCreate := make([]struct {
		id    string
		value string
	}, itemCount)

	go func() {
		for k := range rowsToCreate {
			c := rowsToCreate[k]
			c.id = uuid.New().String()
			c.value = c.id[:8]

			rowsToCreate[k] = c
			time.Sleep(time.Millisecond * 200)

			_, err := cx.Exec("INSERT INTO "+tableName+" (id, value) VALUES ($1, $2)", c.id, c.id)
			require.NoError(t, err)
			time.Sleep(time.Millisecond * 200)

			_, err = cx.Exec("UPDATE "+tableName+" SET value = $1 WHERE id = $2", c.value, c.id)
			require.NoError(t, err)
			time.Sleep(time.Millisecond * 200)

			_, err = cx.Exec("DELETE FROM "+tableName+" WHERE id = $1", c.id)
			require.NoError(t, err)
		}
	}()

	expectedEventCount := watcherCount * itemCount * 3 // 3 operations: insert, update, delete
	var received []Event
	done := false
	for !done {
		select {
		case <-time.After(time.Second*time.Duration(expectedEventCount) + time.Second*5):
			done = true
		case row, ok := <-events:
			if !ok {
				done = true
			} else {
				t.Logf("%+v", row)
				received = append(received, row)
			}
		}
	}

	require.Len(t, received, expectedEventCount)

	// We expect
	// - numOfItems of INSERT (value is id)
	// - numOfItems of UPDATE (value is first 8 chars)
	// - numOfItems of DELETE

	for i := 0; i < len(received); i += 3 {
		inserted := received[i+0]
		updated := received[i+1]
		deleted := received[i+2]

		expectedPk := rowsToCreate[i/3].id
		expectedMessage := fmt.Sprintf("%d: %+v", i/3, rowsToCreate[i/3])

		require.NotEmpty(t, expectedPk, expectedMessage)
		assert.IsType(t, &ChangeEvent{}, inserted, expectedMessage)
		assert.Equal(t, expectedPk, inserted.Source(), expectedMessage)
		assert.Equal(t, expectedPk, gjson.Get(inserted.String(), "value").String(), expectedMessage)

		assert.IsType(t, &ChangeEvent{}, updated, expectedMessage, expectedMessage)
		assert.Equal(t, expectedPk, updated.Source(), expectedMessage)
		assert.Equal(t, expectedPk[:8], gjson.Get(updated.String(), "value").String(), expectedMessage)

		assert.IsType(t, &RemoveEvent{}, deleted, expectedMessage, expectedMessage)
		assert.Equal(t, expectedPk, deleted.Source(), expectedMessage)
	}
}

func send(ctx context.Context, ev chan<- Event, events []Event) {
	for _, e := range events {
		select {
		case <-ctx.Done():
			break
		case ev <- e:
		}
	}
	close(ev)
}

func recv(ctx context.Context, ev <-chan Event) (events []Event) {
	for {
		select {
		case <-ctx.Done():
			return
		case e, ok := <-ev:
			if !ok {
				return
			}
			events = append(events, e)
		}
	}
}

func Test_deduplicate(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	events := []Event{}
	for i := 0; i < 100; i++ {
		events = append(events, &ErrorEvent{
			source: source(fmt.Sprintf("Event %d", i)),
		})
	}

	t.Run("case=proxies", func(t *testing.T) {
		childCtx, cancel := context.WithCancel(ctx)
		defer cancel()
		eventCh := make(EventChannel)
		deduplicatedEvents := make(EventChannel)

		deduplicate(eventCh, deduplicatedEvents, 100)
		go send(childCtx, eventCh, events)
		received := recv(ctx, deduplicatedEvents)

		assert.Equal(t, events, received)
	})

	t.Run("case=deduplicates", func(t *testing.T) {
		childCtx, cancel := context.WithCancel(ctx)
		defer cancel()
		eventCh := make(EventChannel)
		deduplicatedEvents := make(EventChannel)

		duplicateEvents := append(events, events...)

		deduplicate(eventCh, deduplicatedEvents, 100)
		go send(childCtx, eventCh, duplicateEvents)
		received := recv(ctx, deduplicatedEvents)

		assert.Equal(t, events, received)
	})

	t.Run("case=does not deduplicate past capacity", func(t *testing.T) {
		childCtx, cancel := context.WithCancel(ctx)
		defer cancel()
		eventCh := make(EventChannel)
		deduplicatedEvents := make(EventChannel)

		duplicateEvents := append([]Event{events[0]}, events...)
		duplicateEvents = append(duplicateEvents, events[0])
		expectedEvents := append(events, events[0])

		deduplicate(eventCh, deduplicatedEvents, 99)
		go send(childCtx, eventCh, duplicateEvents)
		received := recv(ctx, deduplicatedEvents)

		assert.Equal(t, expectedEvents, received)
	})
}
