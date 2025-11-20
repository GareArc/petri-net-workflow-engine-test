package main

import (
	"bufio"
	"context"
	"fmt"
	"log"
	"math/rand"
	"os"
	"petri-net-mvp/core/petrinet"
	"strings"
	"time"
)

// Ticket models a simple helpdesk ticket.
type Ticket struct {
	ID       string
	Subject  string
	Body     string
	Intent   string // "billing" or "tech"
	Approved bool
	Notes    []string
}

func main() {
	rand.Seed(time.Now().UnixNano())

	net := petrinet.NewPetriNet("Customer Service Demo")

	// Places
	inbox := petrinet.NewPlace("inbox", "Inbox", -1)
	intentPending := petrinet.NewPlace("intent_pending", "Intent Pending", 1)
	classified := petrinet.NewPlace("classified", "Classified", -1)
	billingQ := petrinet.NewPlace("billing_queue", "Billing Queue", -1)
	techQ := petrinet.NewPlace("tech_queue", "Tech Queue", -1)
	reviewQ := petrinet.NewPlace("review_queue", "Review Queue", -1)
	reviewed := petrinet.NewPlace("reviewed", "Reviewed", -1)
	approved := petrinet.NewPlace("approved", "Approved", -1)
	rejected := petrinet.NewPlace("rejected", "Rejected", -1)

	billingAgents := petrinet.NewPlace("billing_agents", "Billing Agents", 2)
	techAgents := petrinet.NewPlace("tech_agents", "Tech Agents", 2)
	reviewers := petrinet.NewPlace("reviewers", "Reviewers", 1)

	// Seed resource tokens
	for i := 0; i < 2; i++ {
		_ = billingAgents.AddTokens(&petrinet.Token{ID: fmt.Sprintf("bill-agent-%d", i), Data: "billing_agent"})
		_ = techAgents.AddTokens(&petrinet.Token{ID: fmt.Sprintf("tech-agent-%d", i), Data: "tech_agent"})
	}
	_ = reviewers.AddTokens(&petrinet.Token{ID: "reviewer-0", Data: "reviewer"})

	// Seed inbox tickets
	tickets := []*Ticket{
		{ID: "T-1001", Subject: "Billing discrepancy", Body: "Charged twice last month"},
		{ID: "T-1002", Subject: "Cannot sign in", Body: "Password reset loop"},
		{ID: "T-1003", Subject: "Invoice request", Body: "Need invoice for March"},
		{ID: "T-1004", Subject: "API timeout", Body: "Requests failing intermittently"},
	}
	for _, t := range tickets {
		_ = inbox.AddTokens(&petrinet.Token{ID: t.ID, Data: t})
	}

	for _, p := range []*petrinet.Place{
		inbox, intentPending, classified, billingQ, techQ, reviewQ, reviewed, approved, rejected,
		billingAgents, techAgents, reviewers,
	} {
		net.AddPlace(p)
	}

	// Transition: LLM classify
	classify := petrinet.NewTransition("llm_classify", "LLM Classify")
	classify.AddInputArc(inbox, 1)
	classify.AddOutputArc(intentPending, 1)
	classify.Action = func(ctx context.Context, toks []*petrinet.Token) ([]*petrinet.Token, error) {
		ticket := toks[0].Data.(*Ticket)
		time.Sleep(time.Duration(80+rand.Intn(120)) * time.Millisecond)
		ticket.Intent = inferIntent(ticket) // suggested intent
		log.Printf("🤖 classify %s -> %s (suggested)\n", ticket.ID, ticket.Intent)
		return []*petrinet.Token{{ID: ticket.ID + "-classified", Data: ticket}}, nil
	}
	net.AddTransition(classify)

	// Transition: user intent selection (mock prompt)
	userIntent := petrinet.NewTransition("user_intent", "User Intent")
	userIntent.AddInputArc(intentPending, 1)
	userIntent.AddOutputArc(classified, 1)
	userIntent.Action = func(ctx context.Context, toks []*petrinet.Token) ([]*petrinet.Token, error) {
		ticket := toks[0].Data.(*Ticket)
		reader := bufio.NewReader(os.Stdin)
		fmt.Printf("Please route ticket %s [%s]: (billing/tech, default=%s) ", ticket.ID, ticket.Subject, ticket.Intent)
		input, _ := reader.ReadString('\n')
		choice := strings.ToLower(strings.TrimSpace(input))
		fallback := false
		if choice == "" {
			choice = ticket.Intent
			fallback = true
		}
		if choice != "billing" && choice != "tech" {
			log.Printf("⚠️  invalid input %q, falling back to %s", choice, ticket.Intent)
			choice = ticket.Intent
			fallback = true
		}
		ticket.Intent = choice
		if fallback {
			log.Printf("👤 user routed %s -> %s (defaulted)", ticket.ID, ticket.Intent)
		} else {
			log.Printf("👤 user routed %s -> %s", ticket.ID, ticket.Intent)
		}
		return []*petrinet.Token{{ID: ticket.ID + "-user-intent", Data: ticket}}, nil
	}
	net.AddTransition(userIntent)

	// Transition: route to billing
	routeBilling := petrinet.NewTransition("route_billing", "Route Billing")
	routeBilling.AddInputArc(classified, 1)
	routeBilling.AddOutputArc(billingQ, 1)
	routeBilling.Guard = func(toks []*petrinet.Token) bool {
		ticket := toks[0].Data.(*Ticket)
		return ticket.Intent == "billing"
	}
	routeBilling.Action = func(ctx context.Context, toks []*petrinet.Token) ([]*petrinet.Token, error) {
		ticket := toks[0].Data.(*Ticket)
		log.Printf("➡️  %s -> billing_queue\n", ticket.ID)
		return []*petrinet.Token{{ID: ticket.ID + "-bill", Data: ticket}}, nil
	}
	net.AddTransition(routeBilling)

	// Transition: route to tech
	routeTech := petrinet.NewTransition("route_tech", "Route Tech")
	routeTech.AddInputArc(classified, 1)
	routeTech.AddOutputArc(techQ, 1)
	routeTech.Guard = func(toks []*petrinet.Token) bool {
		ticket := toks[0].Data.(*Ticket)
		return ticket.Intent == "tech"
	}
	routeTech.Action = func(ctx context.Context, toks []*petrinet.Token) ([]*petrinet.Token, error) {
		ticket := toks[0].Data.(*Ticket)
		log.Printf("➡️  %s -> tech_queue\n", ticket.ID)
		return []*petrinet.Token{{ID: ticket.ID + "-tech", Data: ticket}}, nil
	}
	net.AddTransition(routeTech)

	// Transition: billing agent
	billAgent := petrinet.NewTransition("billing_agent", "Billing Agent")
	billAgent.AddInputArc(billingQ, 1)
	billAgent.AddInputArc(billingAgents, 1)
	billAgent.AddOutputArc(reviewQ, 1)
	billAgent.AddOutputArc(billingAgents, 1)
	billAgent.Action = func(ctx context.Context, toks []*petrinet.Token) ([]*petrinet.Token, error) {
		ticket := toks[0].Data.(*Ticket)
		time.Sleep(time.Duration(120+rand.Intn(180)) * time.Millisecond)
		ticket.Notes = append(ticket.Notes, "Billing agent prepared adjustment")
		log.Printf("💵 billing_agent processed %s\n", ticket.ID)
		return []*petrinet.Token{{ID: ticket.ID + "-bill-done", Data: ticket}, toks[1]}, nil
	}
	net.AddTransition(billAgent)

	// Transition: tech agent
	techAgent := petrinet.NewTransition("tech_agent", "Tech Agent")
	techAgent.AddInputArc(techQ, 1)
	techAgent.AddInputArc(techAgents, 1)
	techAgent.AddOutputArc(reviewQ, 1)
	techAgent.AddOutputArc(techAgents, 1)
	techAgent.Action = func(ctx context.Context, toks []*petrinet.Token) ([]*petrinet.Token, error) {
		ticket := toks[0].Data.(*Ticket)
		time.Sleep(time.Duration(150+rand.Intn(200)) * time.Millisecond)
		ticket.Notes = append(ticket.Notes, "Tech agent drafted fix")
		log.Printf("🛠️  tech_agent processed %s\n", ticket.ID)
		return []*petrinet.Token{{ID: ticket.ID + "-tech-done", Data: ticket}, toks[1]}, nil
	}
	net.AddTransition(techAgent)

	// Transition: QA review
	review := petrinet.NewTransition("review", "QA Review")
	review.AddInputArc(reviewQ, 1)
	review.AddInputArc(reviewers, 1)
	review.AddOutputArc(reviewed, 1)
	review.AddOutputArc(reviewers, 1)
	review.Action = func(ctx context.Context, toks []*petrinet.Token) ([]*petrinet.Token, error) {
		ticket := toks[0].Data.(*Ticket)
		approvedFlag := rand.Float64() > 0.2
		if approvedFlag {
			ticket.Notes = append(ticket.Notes, "QA approved")
			ticket.Approved = true
			log.Printf("✅ QA approved %s (%s)\n", ticket.ID, ticket.Intent)
			return []*petrinet.Token{
				{ID: ticket.ID + "-reviewed", Data: ticket},
				toks[1], // return reviewer
			}, nil
		}
		ticket.Notes = append(ticket.Notes, "QA rejected")
		ticket.Approved = false
		log.Printf("❌ QA rejected %s (%s)\n", ticket.ID, ticket.Intent)
		return []*petrinet.Token{
			{ID: ticket.ID + "-reviewed", Data: ticket},
			toks[1], // return reviewer
		}, nil
	}
	net.AddTransition(review)

	routeApproved := petrinet.NewTransition("route_approved", "Route Approved")
	routeApproved.AddInputArc(reviewed, 1)
	routeApproved.AddOutputArc(approved, 1)
	routeApproved.Guard = func(toks []*petrinet.Token) bool {
		return toks[0].Data.(*Ticket).Approved
	}
	routeApproved.Action = func(ctx context.Context, toks []*petrinet.Token) ([]*petrinet.Token, error) {
		return []*petrinet.Token{{ID: toks[0].ID + "-ok", Data: toks[0].Data}}, nil
	}
	net.AddTransition(routeApproved)

	routeRejected := petrinet.NewTransition("route_rejected", "Route Rejected")
	routeRejected.AddInputArc(reviewed, 1)
	routeRejected.AddOutputArc(rejected, 1)
	routeRejected.Guard = func(toks []*petrinet.Token) bool {
		return !toks[0].Data.(*Ticket).Approved
	}
	routeRejected.Action = func(ctx context.Context, toks []*petrinet.Token) ([]*petrinet.Token, error) {
		return []*petrinet.Token{{ID: toks[0].ID + "-reject", Data: toks[0].Data}}, nil
	}
	net.AddTransition(routeRejected)

	ctx := context.Background() // interactive; avoid timeouts while waiting for user input
	if err := net.Run(ctx); err != nil {
		log.Fatalf("run failed: %v", err)
	}

	log.Printf("🏁 finished: approved=%d rejected=%d\n", approved.TokenCount(), rejected.TokenCount())
	printTickets("approved", approved.Tokens)
	printTickets("rejected", rejected.Tokens)
}

func inferIntent(t *Ticket) string {
	text := strings.ToLower(t.Subject + " " + t.Body)
	if strings.Contains(text, "bill") || strings.Contains(text, "invoice") || strings.Contains(text, "charge") {
		return "billing"
	}
	return "tech"
}

func printTickets(label string, toks []*petrinet.Token) {
	log.Printf("---- %s ----", label)
	for _, tk := range toks {
		t := tk.Data.(*Ticket)
		log.Printf("%s intent=%s notes=%v", t.ID, t.Intent, t.Notes)
	}
}
