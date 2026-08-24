// Command svcregistry runs a deterministic demo of the distributed service
// registry: it registers instances, heartbeats, discovers, routes, applies a
// gray rollout, runs eviction and prints a report.
package main

import (
	"fmt"
	"time"

	"svcregistry/internal/control"
	"svcregistry/internal/evict"
	"svcregistry/internal/health"
	"svcregistry/internal/heartbeat"
	"svcregistry/internal/instance"
	"svcregistry/internal/lease"
	"svcregistry/internal/lookup"
	"svcregistry/internal/meta"
	"svcregistry/internal/metric"
	"svcregistry/internal/model"
	"svcregistry/internal/policy"
	"svcregistry/internal/quota"
	"svcregistry/internal/registry"
	"svcregistry/internal/router"
	"svcregistry/internal/sync"
	"svcregistry/internal/version"
)

func main() {
	reg := registry.New()
	leases := lease.New()
	healthStatus := health.New()
	versions := version.New()
	lookuper := lookup.New(reg)
	heartbeater := heartbeat.New(leases, reg)
	lifecycle := instance.New(reg)
	lc := instance.NewLifecycle()
	metaStore := meta.New()
	quotaMgr := quota.NewManager()
	metrics := metric.New()
	syncer := sync.New(reg)
	prober := health.NewProber(healthStatus, func(inst *model.Instance) bool { return inst.Weight > 0 })
	evictor := evict.New(reg, leases, healthStatus, versions)
	controller := control.New(reg, versions, metaStore, evictor)
	cfg := policy.DefaultConfig()
	_ = cfg.Valid()
	_ = cfg.Heartbeat.Valid()
	_ = cfg.Eviction.AllowEvict(0)
	_ = policy.DefaultHeartbeatPolicy().LeaseTTL
	_ = policy.DefaultEvictionPolicy().QuarantineLen
	_ = policy.DefaultQuotaPolicy().DefaultLimit

	quotaMgr.SetLimit("cart", 10)
	quotaMgr.SetLimit("checkout", 5)

	// register instances
	i1 := model.NewInstance("a1", "cart", "v1", "10.0.0.1:8080", 5)
	i2 := model.NewInstance("a2", "cart", "v1", "10.0.0.2:8080", 5)
	i3 := model.NewInstance("b1", "cart", "v2", "10.0.0.3:8080", 1)
	_ = reg.Register("cart", i1)
	_ = reg.Register("cart", i2)
	_ = reg.Register("cart", i3)
	reg.ConfirmDurable("a1")
	reg.ConfirmDurable("a2")
	reg.ConfirmDurable("b1")
	reg.AddTag("cart", "canary")
	_, _ = reg.Service("cart")
	_, _ = reg.Service("missing")

	// leases
	leases.Start("a1", 30*time.Second)
	leases.Start("a2", 30*time.Second)
	leases.Start("b1", 30*time.Second)
	_, _ = leases.Get("a1")
	_, _ = leases.Get("nope")
	guard := lease.NewRenewGuard()
	guard.Accept("a1", model.Lease{InstanceID: "a1", Holder: "a1", Generation: 1})
	guard.Accept("a1", model.Lease{InstanceID: "a1", Holder: "a1", Generation: 0})

	// lifecycle + health
	lifecycle.MarkHealthy("a1")
	lifecycle.MarkHealthy("a2")
	lifecycle.MarkHealthy("b1")
	_ = lifecycle.State("a1")
	_ = lifecycle.State("gone")
	_ = lc.CanTransition(model.InstanceHealthy, model.InstanceSuspect)
	_ = lc.CanTransition(model.InstanceEvicted, model.InstanceHealthy)
	_ = lc.Writable(model.InstanceRegistered)
	_ = instance.View(i1)
	healthStatus.RecordProbe("a1", true)
	healthStatus.RecordProbe("a2", true)
	healthStatus.RecordPassiveFailureDefault("b1")
	healthStatus.RecordPassiveFailure("b1")
	prober.Probe(i1)
	prober.ProbeAll([]*model.Instance{i1, i2, i3})
	_ = healthStatus.Snapshot("a1")
	_ = healthStatus.Snapshot("zzz")
	_ = healthStatus.Unhealthy([]string{"a1", "b1"})
	_ = healthStatus.Suspect([]string{"b1", "a1"})

	// heartbeat
	heartbeater.Beat("a1", 30*time.Second)
	heartbeater.Beat("a2", 30*time.Second)
	_ = heartbeater.Stale(time.Now().Add(time.Hour))
	detector := heartbeat.NewTimeoutDetector(reg)
	_ = detector.StaleInstances(30 * time.Second)
	_ = heartbeat.SortStable([]string{"b1", "a1"})
	reg.Ack("a1")
	reg.TouchHeartbeat("a1", time.Now())

	// versions + rollout
	versions.Ensure("cart", "v1")
	versions.Ensure("cart", "v2")
	versions.SetWeight("cart", "v1", 80)
	versions.SetWeight("cart", "v2", 20)
	_ = versions.Versions("cart")
	_ = versions.Snapshot("cart")
	snap := versions.Snapshot("cart")
	_ = version.ActiveVersions(snap)
	_ = version.Weighted(snap)
	rollout := version.NewRollout(versions)
	rollout.Start("cart", "v2", 20)
	rollout.Finish("cart", "v1")
	rollout.Rollback("cart", "v1")

	// lookup + router
	found := lookuper.Find("cart")
	_ = found
	_ = lookuper.FindVersion("cart", "v1")
	_ = lookuper.FindTag("cart", "canary")
	_ = lookuper.FindTag("cart", "nope")
	f := lookup.Filter{Version: "v1"}
	_ = f.Apply(found)
	rr := router.NewRoundRobin()
	hash := router.NewHashRing()
	lcStrat := router.NewLeastConn(reg.Inflight)
	r1 := router.New(lookuper, versions, rr)
	r2 := router.New(lookuper, versions, hash)
	r3 := router.New(lookuper, versions, lcStrat)
	_, _ = r1.Route("cart", "user-1")
	_, _ = r2.Route("cart", "user-2")
	_, _ = r3.Route("cart", "user-3")
	_, _ = r1.Route("missing", "x")
	_ = r1.Routes("cart", []string{"k1", "k2"})
	_ = rr.Pick("k", []*model.Instance{i1, i2})
	_ = hash.Pick("k", []*model.Instance{i1, i2})
	_ = lcStrat.Pick("k", []*model.Instance{i1, i2})
	_ = rr.Pick("k", nil)

	// in-flight tracking
	reg.BeginRequest("a1")
	reg.BeginRequest("a1")
	_ = reg.Inflight("a1")
	reg.EndRequest("a1")
	reg.EndRequest("a1")
	reg.EndRequest("a1")

	// eviction
	_ = evictor.EvictExpired(time.Now().Add(-time.Hour))
	_ = evictor.EvictUnhealthy()
	_ = evictor.EvictInactiveVersion()
	_ = evictor.Scan(time.Now())
	_ = controller.ListServices()
	_ = controller.ListInstances("cart")
	_ = controller.RunEviction(time.Now())
	controller.ResetService("cart", []string{"v1", "v2"})
	_ = controller.DeleteService("checkout")
	q := evict.NewQuarantine(10)
	q.Add(i1)
	q.Add(i2)
	_ = q.Contains("a1")
	_ = q.Contains("zzz")
	_ = q.Size()

	// metadata
	metaStore.Set("a1", "zone", "cn-north-1")
	metaStore.Set("a2", "zone", "cn-east-1")
	im := metaStore.Get("a1")
	_ = im.Get("zone")
	_ = im.Clone()
	_ = metaStore.Get("gone").Get("x")
	metaStore.Remove("b1")
	_ = metaStore.Size()
	ts := meta.NewTagSet()
	ts.Add("stable")
	ts.Add("canary")
	_ = ts.Has("canary")
	_ = ts.Has("gone")
	_ = ts.List()

	// quota
	bucket := quota.NewBucket(1, 1)
	_ = bucket.Allow()
	_ = bucket.Allow()
	bucket.Refill()
	_ = quotaMgr.Reserve("cart")
	_ = quotaMgr.Reserve("checkout")
	_ = quotaMgr.Used("cart")
	_ = quotaMgr.Limit("cart")
	quotaMgr.Release("cart")
	_ = quotaMgr.Used("cart")

	// metrics
	metrics.RecordRegistration()
	metrics.RecordRegistration()
	metrics.RecordDiscovery()
	metrics.RecordHeartbeat()
	metrics.RecordEviction()
	_ = metrics.Snapshot()
	hist := metric.NewHistogram()
	hist.Record(12)
	hist.Record(8)
	_ = hist.Average()

	// sync
	rows := syncer.Snapshot()
	_ = rows
	cursor := sync.NewCursor()
	cursor.Advance(2)
	_ = cursor.Position()
	cursor.Reset()

	// lease loop (short-lived)
	stop := make(chan struct{})
	go func() { time.Sleep(40 * time.Millisecond); close(stop) }()
	go leases.LeaseLoop("a1", 20*time.Millisecond, 10*time.Millisecond, stop)
	time.Sleep(50 * time.Millisecond)

	// report
	_ = model.QuotaState{Service: "cart", Limit: 10, Used: 2}
	_ = model.QuotaState{}.Available()
	_ = model.Stats{Service: "cart", Instances: 3, Healthy: 3}
	_ = reg.Snapshot()
	_ = i1.TakesTraffic()
	_ = i1.Status.String()
	_ = model.HealthSnapshot{InstanceID: "a1", OK: true}.Healthy()
	_ = model.VersionSnapshot{Service: "cart", Active: map[string]bool{"v1": true}}.ActiveVersions()
	_ = model.Lease{InstanceID: "a1", ExpiresAt: time.Now().Add(-time.Minute)}.Expired(time.Now())
	_ = model.NewService("cart").HasTag("canary")
	_ = model.NewService("cart").HasTag("nope")

	// evict an instance for the report
	lifecycle.MarkSuspect("b1")
	_ = lc.CanTransition(model.InstanceSuspect, model.InstanceHealthy)
	_ = lc.Writable(model.InstanceHealthy)

	fmt.Printf("services=%d instances=%d snapshot=%d avg_ms=%.0f metrics=%s\n",
		len(reg.Services()), len(reg.Instances("cart")), len(rows), hist.Average(), metrics.Report())
}
