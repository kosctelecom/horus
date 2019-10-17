package agent

import (
	"context"
	"errors"
	"horus/log"
	"math/rand"
	"sync/atomic"
	"time"

	"github.com/vma/glog"
)

// mockResults is used for the duration and error of the mock polling
// A random entry is picked at each call.
var mockResults = [...]struct {
	pollErr error
	pollDur int64
}{
	{nil, 15000},
	{nil, 20000},
	{nil, 25000},
	{nil, 50000},
	{nil, 100000},
	{errors.New("Request timeout (after 2 retries)"), 30000},
}

// mockPoll simulates an snmp poll request.
func (sq *snmpQueue) mockPoll(ctx context.Context, req SnmpRequest) {
	req.Debug(1, "start mock polling")
	ongoingMu.Lock()
	ongoingReqs[req.UID] = true
	ongoingMu.Unlock()
	atomic.AddInt64(&waiting, -1)
	mockRes := mockResults[rand.Intn(len(mockResults))]
	res := MakePollResult(req)
	res.pollErr, res.Duration = mockRes.pollErr, mockRes.pollDur
	select {
	case <-ctx.Done():
		req.Debug(1, ">> cancelling mock poll...")
		return
	case <-time.After(time.Duration(res.Duration) * time.Millisecond):
		pollResults <- &res
		req.Debug(1, ">> done mock polling")
		<-sq.workers
	}
}

// mockPush sends a report of the mock poll to the controller.
func (res PollResult) mockPush() {
	log.Debug2f("%s: start sending mock results", res.RequestID)
	if res.pollErr != nil {
		glog.Warningf("mock poll `%s` to %s failed: %v", res.RequestID, res.IPAddr, res.pollErr)
		res.sendReport()
		return
	}
	time.Sleep(10 * time.Millisecond)
	log.Debug2f("%s: sending report", res.RequestID)
	res.sendReport()
}
