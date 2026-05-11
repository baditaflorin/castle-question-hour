// Package utils contains small helpers shared across the backend.
//
// The most important helper here is HandleErrorOrLogWithMessages, which
// normalizes the "log on either branch, return the error on failure" pattern
// the team uses across repos.
package utils

import (
	"log/slog"
)

// HandleErrorOrLogWithMessages logs successMsg at info and returns nil if err
// is nil; otherwise logs errMsg at error with the wrapped err and returns it.
//
// Use it at call sites where you want a single line to express "do the thing
// and tell me about it either way".
func HandleErrorOrLogWithMessages(log *slog.Logger, err error, errMsg, successMsg string, attrs ...any) error {
	if err != nil {
		log.Error(errMsg, append(attrs, slog.Any("err", err))...)
		return err
	}
	if successMsg != "" {
		log.Info(successMsg, attrs...)
	}
	return nil
}
