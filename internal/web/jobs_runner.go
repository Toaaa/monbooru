package web

import (
	"net/http"
)

// startJob attempts to register a foreground job of the given type. On
// conflict it writes 409 + the standard inline flash and returns false;
// the caller should `return`. On success it returns true and the caller
// owns the goroutine + the eventual 202 response.
func (s *Server) startJob(w http.ResponseWriter, jobType string) bool {
	if err := s.jobs.Start(jobType); err != nil {
		flashStatus(w, http.StatusConflict, "A job is already running.")
		return false
	}
	return true
}

// finishJob writes a chunked job's terminal state: the failure, the
// cancelled summary, or the success summary.
func (s *Server) finishJob(err error, cancelled bool, cancelMsg, doneMsg string) {
	if err != nil {
		s.jobs.Fail(err.Error())
		return
	}
	if cancelled {
		s.jobs.Complete(cancelMsg)
		return
	}
	s.jobs.Complete(doneMsg)
}
