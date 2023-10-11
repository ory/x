package crdbx

import (
	"github.com/gobuffalo/pop/v6"
	"github.com/ory/x/sqlcon"
	"net/http"
)

// Control API consistency guarantees
//
// swagger:model consistencyRequestParameters
type ConsistencyRequestParameters struct {
	// Read Consistency Level
	//
	// The read consistency level determines the consistency guarantee for reads and queries:
	//
	// - strong (slow): The read is guaranteed to return the most recent data committed at the start of the read.
	// - eventual (very fast): The result will return data that is about 4.8 seconds old.
	//
	// Ory Network projects created before October 2023 default to strong consistency and projects created after
	// October 2023 default to eventual consistency, if this parameter is not set.
	//
	// The default consistency guarantee can be changed in the Ory Network Console or using the Ory CLI with
	// `ory patch project --replace '/database/default_consistency_level="strong"'`.
	//
	// This feature is fully functional only in Ory Network and currently experimental.
	//
	// required: false
	// in: query
	Consistency ConsistencyLevel `json:"consistency"`
}

// ConsistencyLevel is the consistency level.
// swagger:enum ConsistencyLevel
type ConsistencyLevel string

const (
	// ConsistencyLevelUnset is the unset / default consistency level.
	ConsistencyLevelUnset ConsistencyLevel = ""
	// ConsistencyLevelStrong is the strong consistency level.
	ConsistencyLevelStrong ConsistencyLevel = "strong"
	// ConsistencyLevelEventual is the eventual consistency level using follower read timestamps.
	ConsistencyLevelEventual ConsistencyLevel = "eventual"
)

// ConsistencyLevelFromRequest extracts the consistency level from a request.
func ConsistencyLevelFromRequest(r *http.Request) ConsistencyLevel {
	return ConsistencyLevelFromString(r.URL.Query().Get("consistency"))
}

// ConsistencyLevelFromString converts a string to a ConsistencyLevel.
// If the string is not recognized or unset, ConsistencyLevelStrong is returned.
func ConsistencyLevelFromString(in string) ConsistencyLevel {
	switch in {
	case string(ConsistencyLevelStrong):
		return ConsistencyLevelStrong
	case string(ConsistencyLevelEventual):
		return ConsistencyLevelEventual
	case string(ConsistencyLevelUnset):
		return ConsistencyLevelStrong
	}
	return ConsistencyLevelStrong
}

// SetTransactionConsistency sets the transaction consistency level for CockroachDB.
func SetTransactionConsistency(c *pop.Connection, level ConsistencyLevel, fallback ConsistencyLevel) error {
	if c.Dialect.Name() != "cockroach" {
		// Only CockroachDB supports this.
		return nil
	}

	switch level {
	case ConsistencyLevelStrong:
		// Nothing to do
		return nil
	case ConsistencyLevelEventual:
		// Jumps to end of function
	case ConsistencyLevelUnset:
		fallthrough
	default:
		if fallback != ConsistencyLevelEventual {
			// Nothing to do
			return nil
		}

		// Jumps to end of function
	}

	return sqlcon.HandleError(c.RawQuery("SET TRANSACTION AS OF SYSTEM TIME follower_read_timestamp()").Exec())
}
