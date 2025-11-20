package main

import (
	"context"
	"fmt"
	"log"
	"math/rand"
	"petri-net-mvp/core/petrinet"
	"time"
)

// This example demonstrates:
// 1) Context place for shared workflow state (serialized updates)
// 2) Resource tokens gating concurrency (api_tokens)
// 3) Barrier that waits for both branches before merging
func main() {
	net := petrinet.NewPetriNet("Context + Barrier Demo")

	// Places
	start := petrinet.NewPlace("start", "Start", 1)
	_ = start.AddTokens(&petrinet.Token{ID: "start"})

	docs := petrinet.NewPlace("docs", "Docs", 2)
	results := petrinet.NewPlace("results", "Results", -1)
	final := petrinet.NewPlace("final", "Final", 1)

	api := petrinet.NewPlace("api_tokens", "API Tokens", 2)
	for i := 0; i < 2; i++ {
		_ = api.AddTokens(&petrinet.Token{ID: fmt.Sprintf("api-%d", i), Data: "api_tokens"})
	}

	ctxPlace := petrinet.NewPlace("workflow_ctx", "Workflow Ctx", 1)
	_ = ctxPlace.AddTokens(&petrinet.Token{ID: "ctx-0", Data: map[string]interface{}{"processed": 0}})

	doneA := petrinet.NewPlace("process_a_done", "process_a Done", 1)
	doneB := petrinet.NewPlace("process_b_done", "process_b Done", 1)
	barrierComplete := petrinet.NewPlace("barrier_complete", "Barrier Complete", 1)

	for _, p := range []*petrinet.Place{start, docs, results, final, api, ctxPlace, doneA, doneB, barrierComplete} {
		net.AddPlace(p)
	}

	// Transition: Load two documents into the queue
	load := petrinet.NewTransition("load_docs", "Load Docs")
	load.AddInputArc(start, 1)
	load.AddOutputArc(docs, 2)
	load.Action = func(ctx context.Context, tokens []*petrinet.Token) ([]*petrinet.Token, error) {
		log.Println("➡️  load_docs: enqueueing doc-0, doc-1")
		return []*petrinet.Token{
			{ID: "doc-0", Data: "doc-0"},
			{ID: "doc-1", Data: "doc-1"},
		}, nil
	}
	net.AddTransition(load)

	processFactory := func(name string, done *petrinet.Place) *petrinet.Transition {
		t := petrinet.NewTransition(name, name)
		t.AddInputArc(docs, 1)      // data
		t.AddInputArc(api, 1)       // resource
		t.AddInputArc(ctxPlace, 1)  // context
		t.AddOutputArc(results, 1)  // data out
		t.AddOutputArc(api, 1)      // return resource
		t.AddOutputArc(ctxPlace, 1) // return context
		t.AddOutputArc(done, 1)     // signal barrier

		t.Action = func(c context.Context, toks []*petrinet.Token) ([]*petrinet.Token, error) {
			var docTok, ctxTok, resTok *petrinet.Token
			for _, tk := range toks {
				switch v := tk.Data.(type) {
				case string:
					if v == "api_tokens" {
						resTok = tk
					} else {
						docTok = tk
					}
				case map[string]interface{}:
					ctxTok = tk
				}
			}
			delay := time.Duration(100+rand.Intn(150)) * time.Millisecond
			log.Printf("⚙️  %s: start doc=%v delay=%s ctx=%v", name, docTok.Data, delay, ctxTok.Data)
			time.Sleep(delay)
			if ctxTok != nil {
				if m, ok := ctxTok.Data.(map[string]interface{}); ok {
					if cur, ok := m["processed"].(int); ok {
						m["processed"] = cur + 1
					} else {
						m["processed"] = 1
					}
					m[name] = docTok.Data
				}
			}
			log.Printf("✅ %s: done doc=%v ctx=%v", name, docTok.Data, ctxTok.Data)
			resultTok := &petrinet.Token{ID: name + "-result", Data: fmt.Sprintf("%s -> %s_done", docTok.Data, name)}
			doneTok := &petrinet.Token{ID: name + "-done", Data: name}
			return []*petrinet.Token{resultTok, resTok, ctxTok, doneTok}, nil
		}
		return t
	}

	net.AddTransition(processFactory("process_a", doneA))
	net.AddTransition(processFactory("process_b", doneB))

	// Barrier waits for both branches
	barrier := petrinet.NewTransition("sync_barrier", "Barrier")
	barrier.AddInputArc(doneA, 1)
	barrier.AddInputArc(doneB, 1)
	barrier.AddOutputArc(barrierComplete, 1)
	net.AddTransition(barrier)

	// Merge after barrier
	merge := petrinet.NewTransition("merge", "Merge and Finish")
	merge.AddInputArc(barrierComplete, 1)
	merge.AddInputArc(results, 2)
	merge.AddOutputArc(final, 1)
	merge.Action = func(ctx context.Context, toks []*petrinet.Token) ([]*petrinet.Token, error) {
		fmt.Printf("📦 merge: consumed %d tokens, data=%v\n", len(toks), toks)
		return []*petrinet.Token{{ID: "final-token", Data: "workflow-complete"}}, nil
	}
	net.AddTransition(merge)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := net.Run(ctx); err != nil {
		log.Fatalf("run failed: %v", err)
	}

	if len(ctxPlace.Tokens) > 0 {
		log.Printf("🧠 final context: %+v", ctxPlace.Tokens[0].Data)
	}
}
