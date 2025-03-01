package utils

import (
	"context"
	"database/sql/driver"
	"fmt"
	"io"
	"net"
	"testing"

	"github.com/go-sql-driver/mysql"
	"github.com/pingcap/errors"
	tmysql "github.com/pingcap/tidb/errno"
	"github.com/stretchr/testify/require"
	"go.uber.org/multierr"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func TestIsRetryableError(t *testing.T) {
	require.False(t, IsRetryableError(context.Canceled))
	require.False(t, IsRetryableError(context.DeadlineExceeded))
	require.False(t, IsRetryableError(io.EOF))
	require.False(t, IsRetryableError(&net.AddrError{}))
	require.False(t, IsRetryableError(&net.DNSError{}))
	require.True(t, IsRetryableError(&net.DNSError{IsTimeout: true}))

	// net: connection refused
	_, err := net.Dial("tcp", "localhost:65533")
	require.Error(t, err)
	require.True(t, IsRetryableError(err))

	// MySQL Errors
	require.False(t, IsRetryableError(&mysql.MySQLError{}))
	require.True(t, IsRetryableError(&mysql.MySQLError{Number: tmysql.ErrUnknown}))
	require.True(t, IsRetryableError(&mysql.MySQLError{Number: tmysql.ErrLockDeadlock}))
	require.True(t, IsRetryableError(&mysql.MySQLError{Number: tmysql.ErrPDServerTimeout}))
	require.True(t, IsRetryableError(&mysql.MySQLError{Number: tmysql.ErrTiKVServerTimeout}))
	require.True(t, IsRetryableError(&mysql.MySQLError{Number: tmysql.ErrTiKVServerBusy}))
	require.True(t, IsRetryableError(&mysql.MySQLError{Number: tmysql.ErrResolveLockTimeout}))
	require.True(t, IsRetryableError(&mysql.MySQLError{Number: tmysql.ErrRegionUnavailable}))
	require.True(t, IsRetryableError(&mysql.MySQLError{Number: tmysql.ErrWriteConflictInTiDB}))
	require.True(t, IsRetryableError(&mysql.MySQLError{Number: tmysql.ErrWriteConflict}))
	require.True(t, IsRetryableError(&mysql.MySQLError{Number: tmysql.ErrInfoSchemaExpired}))
	require.True(t, IsRetryableError(&mysql.MySQLError{Number: tmysql.ErrInfoSchemaChanged}))
	require.True(t, IsRetryableError(&mysql.MySQLError{Number: tmysql.ErrTxnRetryable}))

	// gRPC Errors
	require.False(t, IsRetryableError(status.Error(codes.Canceled, "")))
	require.False(t, IsRetryableError(status.Error(codes.Unknown, "")))
	require.True(t, IsRetryableError(status.Error(codes.DeadlineExceeded, "")))
	require.True(t, IsRetryableError(status.Error(codes.NotFound, "")))
	require.True(t, IsRetryableError(status.Error(codes.AlreadyExists, "")))
	require.True(t, IsRetryableError(status.Error(codes.PermissionDenied, "")))
	require.True(t, IsRetryableError(status.Error(codes.ResourceExhausted, "")))
	require.True(t, IsRetryableError(status.Error(codes.Aborted, "")))
	require.True(t, IsRetryableError(status.Error(codes.OutOfRange, "")))
	require.True(t, IsRetryableError(status.Error(codes.Unavailable, "")))
	require.True(t, IsRetryableError(status.Error(codes.DataLoss, "")))

	// sqlmock errors
	require.False(t, IsRetryableError(fmt.Errorf("call to database Close was not expected")))
	require.False(t, IsRetryableError(errors.New("call to database Close was not expected")))

	// stderr
	require.True(t, IsRetryableError(mysql.ErrInvalidConn))
	require.True(t, IsRetryableError(driver.ErrBadConn))
	require.False(t, IsRetryableError(fmt.Errorf("error")))

	// multierr
	require.False(t, IsRetryableError(multierr.Combine(context.Canceled, context.Canceled)))
	require.True(t, IsRetryableError(multierr.Combine(&net.DNSError{IsTimeout: true}, &net.DNSError{IsTimeout: true})))
	require.False(t, IsRetryableError(multierr.Combine(context.Canceled, &net.DNSError{IsTimeout: true})))
}
