package regression

import (
	"context"
	"database/sql/driver"
	"errors"
	"io"
)

// SQLConnector is the fake database reports that the device exists, but the
// owner-scoped update affects no rows, as it would for a device owned by
// another user.
type SQLConnector struct {
	updateArgs []driver.NamedValue
}

func (c *SQLConnector) UpdateArgs() []driver.NamedValue {
	return c.updateArgs
}

func (c *SQLConnector) Connect(context.Context) (driver.Conn, error) {
	return SQLConn{connector: c}, nil
}

func (c *SQLConnector) Driver() driver.Driver {
	return SQLDriver{connector: c}
}

type SQLDriver struct {
	connector *SQLConnector
}

func (d SQLDriver) Open(string) (driver.Conn, error) {
	return SQLConn(d), nil
}

type SQLConn struct {
	connector *SQLConnector
}

var (
	errPrepare     = errors.New("unexpected prepare")
	errTransaction = errors.New("unexpected transaction")
)

func (SQLConn) Prepare(string) (driver.Stmt, error) {
	return nil, errPrepare
}

func (SQLConn) Close() error { return nil }

func (SQLConn) Begin() (driver.Tx, error) {
	return nil, errTransaction
}

func (SQLConn) QueryContext(
	_ context.Context,
	_ string,
	_ []driver.NamedValue,
) (driver.Rows, error) {
	return &ExistsRows{done: false}, nil
}

func (c SQLConn) ExecContext(
	_ context.Context,
	_ string,
	args []driver.NamedValue,
) (driver.Result, error) {
	c.connector.updateArgs = append([]driver.NamedValue(nil), args...)

	return driver.RowsAffected(0), nil
}

type ExistsRows struct {
	done bool
}

func (*ExistsRows) Columns() []string { return []string{"exists"} }
func (*ExistsRows) Close() error      { return nil }

func (r *ExistsRows) Next(dest []driver.Value) error {
	if r.done {
		return io.EOF
	}

	r.done = true
	dest[0] = true

	return nil
}
