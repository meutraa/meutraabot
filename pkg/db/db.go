// Code generated by sqlc. DO NOT EDIT.

package db

import (
	"context"
	"database/sql"
	"fmt"
)

type DBTX interface {
	ExecContext(context.Context, string, ...interface{}) (sql.Result, error)
	PrepareContext(context.Context, string) (*sql.Stmt, error)
	QueryContext(context.Context, string, ...interface{}) (*sql.Rows, error)
	QueryRowContext(context.Context, string, ...interface{}) *sql.Row
}

func New(db DBTX) *Queries {
	return &Queries{db: db}
}

func Prepare(ctx context.Context, db DBTX) (*Queries, error) {
	q := Queries{db: db}
	var err error
	if q.approveStmt, err = db.PrepareContext(ctx, approve); err != nil {
		return nil, fmt.Errorf("error preparing query Approve: %w", err)
	}
	if q.createChannelStmt, err = db.PrepareContext(ctx, createChannel); err != nil {
		return nil, fmt.Errorf("error preparing query CreateChannel: %w", err)
	}
	if q.createMessageStmt, err = db.PrepareContext(ctx, createMessage); err != nil {
		return nil, fmt.Errorf("error preparing query CreateMessage: %w", err)
	}
	if q.createUserStmt, err = db.PrepareContext(ctx, createUser); err != nil {
		return nil, fmt.Errorf("error preparing query CreateUser: %w", err)
	}
	if q.deleteChannelStmt, err = db.PrepareContext(ctx, deleteChannel); err != nil {
		return nil, fmt.Errorf("error preparing query DeleteChannel: %w", err)
	}
	if q.deleteCommandStmt, err = db.PrepareContext(ctx, deleteCommand); err != nil {
		return nil, fmt.Errorf("error preparing query DeleteCommand: %w", err)
	}
	if q.getChannelsStmt, err = db.PrepareContext(ctx, getChannels); err != nil {
		return nil, fmt.Errorf("error preparing query GetChannels: %w", err)
	}
	if q.getCommandStmt, err = db.PrepareContext(ctx, getCommand); err != nil {
		return nil, fmt.Errorf("error preparing query GetCommand: %w", err)
	}
	if q.getCommandsStmt, err = db.PrepareContext(ctx, getCommands); err != nil {
		return nil, fmt.Errorf("error preparing query GetCommands: %w", err)
	}
	if q.getCounterStmt, err = db.PrepareContext(ctx, getCounter); err != nil {
		return nil, fmt.Errorf("error preparing query GetCounter: %w", err)
	}
	if q.getMatchingCommandsStmt, err = db.PrepareContext(ctx, getMatchingCommands); err != nil {
		return nil, fmt.Errorf("error preparing query GetMatchingCommands: %w", err)
	}
	if q.getMessageCountStmt, err = db.PrepareContext(ctx, getMessageCount); err != nil {
		return nil, fmt.Errorf("error preparing query GetMessageCount: %w", err)
	}
	if q.getMetricsStmt, err = db.PrepareContext(ctx, getMetrics); err != nil {
		return nil, fmt.Errorf("error preparing query GetMetrics: %w", err)
	}
	if q.getTopWatchersStmt, err = db.PrepareContext(ctx, getTopWatchers); err != nil {
		return nil, fmt.Errorf("error preparing query GetTopWatchers: %w", err)
	}
	if q.getTopWatchersAverageStmt, err = db.PrepareContext(ctx, getTopWatchersAverage); err != nil {
		return nil, fmt.Errorf("error preparing query GetTopWatchersAverage: %w", err)
	}
	if q.getWatchTimeRankStmt, err = db.PrepareContext(ctx, getWatchTimeRank); err != nil {
		return nil, fmt.Errorf("error preparing query GetWatchTimeRank: %w", err)
	}
	if q.getWatchTimeRankAverageStmt, err = db.PrepareContext(ctx, getWatchTimeRankAverage); err != nil {
		return nil, fmt.Errorf("error preparing query GetWatchTimeRankAverage: %w", err)
	}
	if q.isApprovedStmt, err = db.PrepareContext(ctx, isApproved); err != nil {
		return nil, fmt.Errorf("error preparing query IsApproved: %w", err)
	}
	if q.setCommandStmt, err = db.PrepareContext(ctx, setCommand); err != nil {
		return nil, fmt.Errorf("error preparing query SetCommand: %w", err)
	}
	if q.unapproveStmt, err = db.PrepareContext(ctx, unapprove); err != nil {
		return nil, fmt.Errorf("error preparing query Unapprove: %w", err)
	}
	if q.updateCounterStmt, err = db.PrepareContext(ctx, updateCounter); err != nil {
		return nil, fmt.Errorf("error preparing query UpdateCounter: %w", err)
	}
	if q.updateMetricsStmt, err = db.PrepareContext(ctx, updateMetrics); err != nil {
		return nil, fmt.Errorf("error preparing query UpdateMetrics: %w", err)
	}
	return &q, nil
}

func (q *Queries) Close() error {
	var err error
	if q.approveStmt != nil {
		if cerr := q.approveStmt.Close(); cerr != nil {
			err = fmt.Errorf("error closing approveStmt: %w", cerr)
		}
	}
	if q.createChannelStmt != nil {
		if cerr := q.createChannelStmt.Close(); cerr != nil {
			err = fmt.Errorf("error closing createChannelStmt: %w", cerr)
		}
	}
	if q.createMessageStmt != nil {
		if cerr := q.createMessageStmt.Close(); cerr != nil {
			err = fmt.Errorf("error closing createMessageStmt: %w", cerr)
		}
	}
	if q.createUserStmt != nil {
		if cerr := q.createUserStmt.Close(); cerr != nil {
			err = fmt.Errorf("error closing createUserStmt: %w", cerr)
		}
	}
	if q.deleteChannelStmt != nil {
		if cerr := q.deleteChannelStmt.Close(); cerr != nil {
			err = fmt.Errorf("error closing deleteChannelStmt: %w", cerr)
		}
	}
	if q.deleteCommandStmt != nil {
		if cerr := q.deleteCommandStmt.Close(); cerr != nil {
			err = fmt.Errorf("error closing deleteCommandStmt: %w", cerr)
		}
	}
	if q.getChannelsStmt != nil {
		if cerr := q.getChannelsStmt.Close(); cerr != nil {
			err = fmt.Errorf("error closing getChannelsStmt: %w", cerr)
		}
	}
	if q.getCommandStmt != nil {
		if cerr := q.getCommandStmt.Close(); cerr != nil {
			err = fmt.Errorf("error closing getCommandStmt: %w", cerr)
		}
	}
	if q.getCommandsStmt != nil {
		if cerr := q.getCommandsStmt.Close(); cerr != nil {
			err = fmt.Errorf("error closing getCommandsStmt: %w", cerr)
		}
	}
	if q.getCounterStmt != nil {
		if cerr := q.getCounterStmt.Close(); cerr != nil {
			err = fmt.Errorf("error closing getCounterStmt: %w", cerr)
		}
	}
	if q.getMatchingCommandsStmt != nil {
		if cerr := q.getMatchingCommandsStmt.Close(); cerr != nil {
			err = fmt.Errorf("error closing getMatchingCommandsStmt: %w", cerr)
		}
	}
	if q.getMessageCountStmt != nil {
		if cerr := q.getMessageCountStmt.Close(); cerr != nil {
			err = fmt.Errorf("error closing getMessageCountStmt: %w", cerr)
		}
	}
	if q.getMetricsStmt != nil {
		if cerr := q.getMetricsStmt.Close(); cerr != nil {
			err = fmt.Errorf("error closing getMetricsStmt: %w", cerr)
		}
	}
	if q.getTopWatchersStmt != nil {
		if cerr := q.getTopWatchersStmt.Close(); cerr != nil {
			err = fmt.Errorf("error closing getTopWatchersStmt: %w", cerr)
		}
	}
	if q.getTopWatchersAverageStmt != nil {
		if cerr := q.getTopWatchersAverageStmt.Close(); cerr != nil {
			err = fmt.Errorf("error closing getTopWatchersAverageStmt: %w", cerr)
		}
	}
	if q.getWatchTimeRankStmt != nil {
		if cerr := q.getWatchTimeRankStmt.Close(); cerr != nil {
			err = fmt.Errorf("error closing getWatchTimeRankStmt: %w", cerr)
		}
	}
	if q.getWatchTimeRankAverageStmt != nil {
		if cerr := q.getWatchTimeRankAverageStmt.Close(); cerr != nil {
			err = fmt.Errorf("error closing getWatchTimeRankAverageStmt: %w", cerr)
		}
	}
	if q.isApprovedStmt != nil {
		if cerr := q.isApprovedStmt.Close(); cerr != nil {
			err = fmt.Errorf("error closing isApprovedStmt: %w", cerr)
		}
	}
	if q.setCommandStmt != nil {
		if cerr := q.setCommandStmt.Close(); cerr != nil {
			err = fmt.Errorf("error closing setCommandStmt: %w", cerr)
		}
	}
	if q.unapproveStmt != nil {
		if cerr := q.unapproveStmt.Close(); cerr != nil {
			err = fmt.Errorf("error closing unapproveStmt: %w", cerr)
		}
	}
	if q.updateCounterStmt != nil {
		if cerr := q.updateCounterStmt.Close(); cerr != nil {
			err = fmt.Errorf("error closing updateCounterStmt: %w", cerr)
		}
	}
	if q.updateMetricsStmt != nil {
		if cerr := q.updateMetricsStmt.Close(); cerr != nil {
			err = fmt.Errorf("error closing updateMetricsStmt: %w", cerr)
		}
	}
	return err
}

func (q *Queries) exec(ctx context.Context, stmt *sql.Stmt, query string, args ...interface{}) (sql.Result, error) {
	switch {
	case stmt != nil && q.tx != nil:
		return q.tx.StmtContext(ctx, stmt).ExecContext(ctx, args...)
	case stmt != nil:
		return stmt.ExecContext(ctx, args...)
	default:
		return q.db.ExecContext(ctx, query, args...)
	}
}

func (q *Queries) query(ctx context.Context, stmt *sql.Stmt, query string, args ...interface{}) (*sql.Rows, error) {
	switch {
	case stmt != nil && q.tx != nil:
		return q.tx.StmtContext(ctx, stmt).QueryContext(ctx, args...)
	case stmt != nil:
		return stmt.QueryContext(ctx, args...)
	default:
		return q.db.QueryContext(ctx, query, args...)
	}
}

func (q *Queries) queryRow(ctx context.Context, stmt *sql.Stmt, query string, args ...interface{}) *sql.Row {
	switch {
	case stmt != nil && q.tx != nil:
		return q.tx.StmtContext(ctx, stmt).QueryRowContext(ctx, args...)
	case stmt != nil:
		return stmt.QueryRowContext(ctx, args...)
	default:
		return q.db.QueryRowContext(ctx, query, args...)
	}
}

type Queries struct {
	db                          DBTX
	tx                          *sql.Tx
	approveStmt                 *sql.Stmt
	createChannelStmt           *sql.Stmt
	createMessageStmt           *sql.Stmt
	createUserStmt              *sql.Stmt
	deleteChannelStmt           *sql.Stmt
	deleteCommandStmt           *sql.Stmt
	getChannelsStmt             *sql.Stmt
	getCommandStmt              *sql.Stmt
	getCommandsStmt             *sql.Stmt
	getCounterStmt              *sql.Stmt
	getMatchingCommandsStmt     *sql.Stmt
	getMessageCountStmt         *sql.Stmt
	getMetricsStmt              *sql.Stmt
	getTopWatchersStmt          *sql.Stmt
	getTopWatchersAverageStmt   *sql.Stmt
	getWatchTimeRankStmt        *sql.Stmt
	getWatchTimeRankAverageStmt *sql.Stmt
	isApprovedStmt              *sql.Stmt
	setCommandStmt              *sql.Stmt
	unapproveStmt               *sql.Stmt
	updateCounterStmt           *sql.Stmt
	updateMetricsStmt           *sql.Stmt
}

func (q *Queries) WithTx(tx *sql.Tx) *Queries {
	return &Queries{
		db:                          tx,
		tx:                          tx,
		approveStmt:                 q.approveStmt,
		createChannelStmt:           q.createChannelStmt,
		createMessageStmt:           q.createMessageStmt,
		createUserStmt:              q.createUserStmt,
		deleteChannelStmt:           q.deleteChannelStmt,
		deleteCommandStmt:           q.deleteCommandStmt,
		getChannelsStmt:             q.getChannelsStmt,
		getCommandStmt:              q.getCommandStmt,
		getCommandsStmt:             q.getCommandsStmt,
		getCounterStmt:              q.getCounterStmt,
		getMatchingCommandsStmt:     q.getMatchingCommandsStmt,
		getMessageCountStmt:         q.getMessageCountStmt,
		getMetricsStmt:              q.getMetricsStmt,
		getTopWatchersStmt:          q.getTopWatchersStmt,
		getTopWatchersAverageStmt:   q.getTopWatchersAverageStmt,
		getWatchTimeRankStmt:        q.getWatchTimeRankStmt,
		getWatchTimeRankAverageStmt: q.getWatchTimeRankAverageStmt,
		isApprovedStmt:              q.isApprovedStmt,
		setCommandStmt:              q.setCommandStmt,
		unapproveStmt:               q.unapproveStmt,
		updateCounterStmt:           q.updateCounterStmt,
		updateMetricsStmt:           q.updateMetricsStmt,
	}
}
