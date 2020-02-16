// Copyright (c) 2016 - 2020 Sqreen. All Rights Reserved.
// Please refer to our terms for more information:
// https://www.sqreen.io/terms.html

package event_test

import (
	"testing"

	"github.com/sqreen/go-agent/internal/event"
	"github.com/stretchr/testify/require"
)

func TestUsage(t *testing.T) {
	t.Run("value", func(t *testing.T) {
		var record event.Record
		uid := event.UserIdentifierMap{"uid": "my uid"}
		record.AddCustomEvent("test").WithUserIdentifiers(uid)
		recorded := record.CloseRecord()
		event := recorded.CustomEvents[0]
		require.Equal(t, "test", event.Event)
		require.Equal(t, uid, event.UserID)
	})
}
