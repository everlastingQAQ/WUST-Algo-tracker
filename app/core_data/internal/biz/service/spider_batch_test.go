package service

import "testing"

func TestSpiderInsertBatchSizeStaysBelowPostgresParameterLimit(t *testing.T) {
	const postgresExtendedProtocolParameterLimit = 65535
	const submitLogWritableColumns = 8

	if spiderInsertBatchSize*submitLogWritableColumns >= postgresExtendedProtocolParameterLimit {
		t.Fatalf(
			"spiderInsertBatchSize=%d may exceed postgres parameter limit with %d columns",
			spiderInsertBatchSize,
			submitLogWritableColumns,
		)
	}
}
