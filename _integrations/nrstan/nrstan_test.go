package nrstan

import (
	"os"
	"sync"
	"testing"

	"github.com/nats-io/nats-streaming-server/server"
	"github.com/nats-io/stan.go"
	newrelic "github.com/newrelic/go-agent"
	"github.com/newrelic/go-agent/internal"
)

const (
	clusterName = "my_test_cluster"
	clientName  = "me"
)

func TestMain(m *testing.M) {
	s, err := server.RunServer(clusterName)
	if err != nil {
		panic(err)
	}
	defer s.Shutdown()
	os.Exit(m.Run())
}

func createTestApp(t *testing.T) newrelic.Application {
	cfg := newrelic.NewConfig("appname", "0123456789012345678901234567890123456789")
	cfg.Enabled = false
	cfg.DistributedTracer.Enabled = true
	cfg.TransactionTracer.SegmentThreshold = 0
	cfg.TransactionTracer.Threshold.IsApdexFailing = false
	cfg.TransactionTracer.Threshold.Duration = 0
	app, err := newrelic.NewApplication(cfg)
	if nil != err {
		t.Fatal(err)
	}
	replyfn := func(reply *internal.ConnectReply) {
		reply.AdaptiveSampler = internal.SampleEverything{}
		reply.AccountID = "123"
		reply.TrustedAccountKey = "123"
		reply.PrimaryAppID = "456"
	}
	internal.HarvestTesting(app, replyfn)
	return app
}

func TestSubWrapperWithNilApp(t *testing.T) {
	subject := "sample.subject1"
	sc, err := stan.Connect(clusterName, clientName)
	if err != nil {
		t.Fatal("Couldn't connect to server", err)
	}
	defer sc.Close()

	wg := sync.WaitGroup{}
	sc.Subscribe(subject, StreamingSubWrapper(nil, func(msg *stan.Msg) {
		defer wg.Done()
	}))
	wg.Add(1)
	sc.Publish(subject, []byte("data"))
	wg.Wait()
}

func TestSubWrapper(t *testing.T) {
	subject := "sample.subject2"
	sc, err := stan.Connect(clusterName, clientName)
	if err != nil {
		t.Fatal("Couldn't connect to server", err)
	}
	defer sc.Close()

	wg := sync.WaitGroup{}
	app := createTestApp(t)
	sc.Subscribe(subject, WgWrapper(&wg, StreamingSubWrapper(app, func(msg *stan.Msg) {})))

	wg.Add(1)
	sc.Publish(subject, []byte("data"))
	wg.Wait()

	app.(internal.Expect).ExpectMetrics(t, []internal.WantMetric{
		{Name: "OtherTransaction/all", Scope: "", Forced: true, Data: nil},
		{Name: "OtherTransactionTotalTime", Scope: "", Forced: true, Data: nil},
		{Name: "DurationByCaller/Unknown/Unknown/Unknown/Unknown/all", Scope: "", Forced: false, Data: nil},
		{Name: "DurationByCaller/Unknown/Unknown/Unknown/Unknown/allOther", Scope: "", Forced: false, Data: nil},
		{Name: "OtherTransaction/Go/Message/stan.go/Topic/sample.subject2:subscriber", Scope: "", Forced: true, Data: nil},
		{Name: "OtherTransactionTotalTime/Go/Message/stan.go/Topic/sample.subject2:subscriber", Scope: "", Forced: false, Data: nil},
	})
}

// Wrapper function to ensure that the NR wrapper is done recording transaction data before wg.Done() is called
func WgWrapper(wg *sync.WaitGroup, nrWrap func(msg *stan.Msg)) func(msg *stan.Msg) {
	return func(msg *stan.Msg) {
		nrWrap(msg)
		wg.Done()
	}
}
