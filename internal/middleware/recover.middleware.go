package middleware

import (
	"fmt"
	"net/http"
	"runtime/debug"

	"github.com/loopinnovates/wallet-transfer-assignment/pkg/utils"
)

func Recover(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		defer func() {
			if rec := recover(); rec != nil {
				fmt.Printf("panic recovered: %v\n%s\n", rec, debug.Stack())
				utils.WriteError(w, nil)
			}
		}()
		next.ServeHTTP(w, r)
	})
}
