// Command work is a standalone CLI for the Work Graph (Layer 1).
// It exposes task management as signed, auditable events on the shared event graph.
package main

import (
	"context"
	"crypto/ed25519"
	"crypto/sha256"
	"flag"
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/lovyou-ai/eventgraph/go/pkg/actor"
	"github.com/lovyou-ai/eventgraph/go/pkg/actor/pgactor"
	"github.com/lovyou-ai/eventgraph/go/pkg/event"
	"github.com/lovyou-ai/eventgraph/go/pkg/store"
	"github.com/lovyou-ai/eventgraph/go/pkg/store/pgstore"
	"github.com/lovyou-ai/eventgraph/go/pkg/types"

	"github.com/lovyou-ai/work"
)

func main() {
	if err := run(); err != nil {
		fmt.Fprintf(os.Stderr, "%v\n", err)
		os.Exit(1)
	}
}

func run() error {
	// Global flags — must appear before the subcommand name.
	human := flag.String("human", "", "Human operator name (required)")
	storeDSN := flag.String("store", "", "Store connection string (postgres://... or empty for in-memory)")
	flag.Usage = func() {
		fmt.Fprintln(os.Stderr, "usage: work --human name [--store postgres://...] <subcommand> [flags]")
		fmt.Fprintln(os.Stderr, "")
		fmt.Fprintln(os.Stderr, "Subcommands:")
		fmt.Fprintln(os.Stderr, "  create   --title '...' [--description '...'] [--priority high]")
		fmt.Fprintln(os.Stderr, "  list     [--open] [--assignee actor_id] [--priority high]")
		fmt.Fprintln(os.Stderr, "  assign   --task task_id [--assignee actor_id]")
		fmt.Fprintln(os.Stderr, "  complete --task task_id [--summary '...']")
		fmt.Fprintln(os.Stderr, "  status   --task task_id")
	}
	flag.Parse()

	args := flag.Args()
	if len(args) == 0 {
		flag.Usage()
		return fmt.Errorf("subcommand required")
	}
	if *human == "" {
		return fmt.Errorf("--human is required (the name of the human operator)")
	}

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	// Resolve store DSN: flag > DATABASE_URL env > in-memory.
	dsn := *storeDSN
	if dsn == "" {
		dsn = os.Getenv("DATABASE_URL")
	}

	// Open shared pool for Postgres, or nil for in-memory.
	var pool *pgxpool.Pool
	if dsn != "" {
		fmt.Fprintf(os.Stderr, "Postgres: %s\n", dsn)
		var err error
		pool, err = pgxpool.New(ctx, dsn)
		if err != nil {
			return fmt.Errorf("postgres: %w", err)
		}
		defer pool.Close()
	}

	s, err := openStore(ctx, pool)
	if err != nil {
		return fmt.Errorf("store: %w", err)
	}
	defer func() {
		if err := s.Close(); err != nil {
			fmt.Fprintf(os.Stderr, "store close: %v\n", err)
		}
	}()

	actors, err := openActorStore(ctx, pool)
	if err != nil {
		return fmt.Errorf("actor store: %w", err)
	}

	// Bootstrap human actor — same key-derivation pattern as cmd/hive.
	if pool != nil {
		fmt.Fprintln(os.Stderr, "WARNING: CLI key derivation is insecure for persistent Postgres stores.")
		fmt.Fprintln(os.Stderr, "         Production should use Google auth. Proceeding for development.")
	}
	humanID, err := registerHuman(actors, *human)
	if err != nil {
		return fmt.Errorf("register human: %w", err)
	}

	// Register work event type unmarshalers before any store reads —
	// Head() deserializes the latest event which may be a work type.
	work.RegisterEventTypes()

	// Bootstrap the event graph if it has no genesis event.
	if err := bootstrapGraph(s, humanID); err != nil {
		return fmt.Errorf("bootstrap graph: %w", err)
	}

	// Build factory and signer for work events.
	registry := event.DefaultRegistry()
	work.RegisterWithRegistry(registry)
	factory := event.NewEventFactory(registry)
	signer := deriveSignerFromID(humanID)

	// Create the task store.
	ts := work.NewTaskStore(s, factory, signer)

	// Get current head for causality chain.
	head, err := s.Head()
	if err != nil {
		return fmt.Errorf("get head: %w", err)
	}
	var causes []types.EventID
	if head.IsSome() {
		causes = []types.EventID{head.Unwrap().ID()}
	}

	// Generate a new conversation ID for this CLI session.
	convID, err := newConversationID()
	if err != nil {
		return fmt.Errorf("conversation id: %w", err)
	}

	subcommand := args[0]
	subargs := args[1:]

	switch subcommand {
	case "create":
		return runCreate(ts, humanID, causes, convID, subargs)
	case "list":
		return runList(ts, subargs)
	case "assign":
		return runAssign(ts, humanID, causes, convID, subargs)
	case "complete":
		return runComplete(ts, humanID, causes, convID, subargs)
	case "status":
		return runStatus(ts, subargs)
	default:
		return fmt.Errorf("unknown subcommand %q — use: create, list, assign, complete, status", subcommand)
	}
}

// runCreate implements: work create --title '...' [--description '...'] [--priority high]
func runCreate(ts *work.TaskStore, humanID types.ActorID, causes []types.EventID, convID types.ConversationID, args []string) error {
	fs := flag.NewFlagSet("create", flag.ContinueOnError)
	title := fs.String("title", "", "Task title (required)")
	description := fs.String("description", "", "Task description")
	priority := fs.String("priority", "", "Task priority: low, medium, high, critical (default: medium)")
	if err := fs.Parse(args); err != nil {
		return err
	}
	if *title == "" {
		return fmt.Errorf("--title is required")
	}
	task, err := ts.Create(humanID, *title, *description, causes, convID, work.TaskPriority(*priority))
	if err != nil {
		return fmt.Errorf("create: %w", err)
	}
	fmt.Printf("Created task %s\n", task.ID)
	fmt.Printf("  title:    %s\n", task.Title)
	if task.Description != "" {
		fmt.Printf("  desc:     %s\n", task.Description)
	}
	fmt.Printf("  priority: %s\n", task.Priority)
	fmt.Printf("  by:       %s\n", task.CreatedBy)
	return nil
}

// runList implements: work list [--open] [--assignee actor_id] [--priority high]
func runList(ts *work.TaskStore, args []string) error {
	fs := flag.NewFlagSet("list", flag.ContinueOnError)
	open := fs.Bool("open", false, "Only show open (unblocked, incomplete) tasks")
	assignee := fs.String("assignee", "", "Filter by assignee actor ID")
	priority := fs.String("priority", "", "Filter by priority: low, medium, high, critical")
	if err := fs.Parse(args); err != nil {
		return err
	}

	var tasks []work.Task
	var err error

	switch {
	case *assignee != "":
		aid, aidErr := types.NewActorID(*assignee)
		if aidErr != nil {
			return fmt.Errorf("invalid assignee: %w", aidErr)
		}
		tasks, err = ts.GetByAssignee(aid)
	case *open:
		tasks, err = ts.ListOpen()
	default:
		tasks, err = ts.List(100)
	}
	if err != nil {
		return fmt.Errorf("list: %w", err)
	}

	// Apply priority filter if specified.
	if *priority != "" {
		p := work.TaskPriority(*priority)
		filtered := tasks[:0]
		for _, t := range tasks {
			if t.Priority == p {
				filtered = append(filtered, t)
			}
		}
		tasks = filtered
	}

	if len(tasks) == 0 {
		fmt.Println("No tasks found.")
		return nil
	}
	for _, t := range tasks {
		fmt.Printf("%s  [%s]  %s\n", t.ID, t.Priority, t.Title)
		if t.Description != "" {
			fmt.Printf("                                                      %s\n", t.Description)
		}
	}
	return nil
}

// runAssign implements: work assign --task task_id [--assignee actor_id]
func runAssign(ts *work.TaskStore, humanID types.ActorID, causes []types.EventID, convID types.ConversationID, args []string) error {
	fs := flag.NewFlagSet("assign", flag.ContinueOnError)
	taskIDStr := fs.String("task", "", "Task ID (required)")
	assigneeStr := fs.String("assignee", "", "Assignee actor ID (defaults to the human operator)")
	if err := fs.Parse(args); err != nil {
		return err
	}
	if *taskIDStr == "" {
		return fmt.Errorf("--task is required")
	}
	taskID, err := types.NewEventID(*taskIDStr)
	if err != nil {
		return fmt.Errorf("invalid task id: %w", err)
	}
	assignee := humanID
	if *assigneeStr != "" {
		assignee, err = types.NewActorID(*assigneeStr)
		if err != nil {
			return fmt.Errorf("invalid assignee: %w", err)
		}
	}
	if err := ts.Assign(humanID, taskID, assignee, causes, convID); err != nil {
		return fmt.Errorf("assign: %w", err)
	}
	fmt.Printf("Assigned task %s to %s\n", taskID, assignee)
	return nil
}

// runComplete implements: work complete --task task_id [--summary '...']
func runComplete(ts *work.TaskStore, humanID types.ActorID, causes []types.EventID, convID types.ConversationID, args []string) error {
	fs := flag.NewFlagSet("complete", flag.ContinueOnError)
	taskIDStr := fs.String("task", "", "Task ID (required)")
	summary := fs.String("summary", "", "Completion summary")
	if err := fs.Parse(args); err != nil {
		return err
	}
	if *taskIDStr == "" {
		return fmt.Errorf("--task is required")
	}
	taskID, err := types.NewEventID(*taskIDStr)
	if err != nil {
		return fmt.Errorf("invalid task id: %w", err)
	}
	if err := ts.Complete(humanID, taskID, *summary, causes, convID); err != nil {
		return fmt.Errorf("complete: %w", err)
	}
	fmt.Printf("Completed task %s\n", taskID)
	if *summary != "" {
		fmt.Printf("  summary: %s\n", *summary)
	}
	return nil
}

// runStatus implements: work status --task task_id
func runStatus(ts *work.TaskStore, args []string) error {
	fs := flag.NewFlagSet("status", flag.ContinueOnError)
	taskIDStr := fs.String("task", "", "Task ID (required)")
	if err := fs.Parse(args); err != nil {
		return err
	}
	if *taskIDStr == "" {
		return fmt.Errorf("--task is required")
	}
	taskID, err := types.NewEventID(*taskIDStr)
	if err != nil {
		return fmt.Errorf("invalid task id: %w", err)
	}
	status, err := ts.GetStatus(taskID)
	if err != nil {
		return fmt.Errorf("status: %w", err)
	}
	priority, err := ts.GetPriority(taskID)
	if err != nil {
		return fmt.Errorf("priority: %w", err)
	}
	blocked, err := ts.IsBlocked(taskID)
	if err != nil {
		return fmt.Errorf("blocked: %w", err)
	}
	fmt.Printf("Task %s\n", taskID)
	fmt.Printf("  status:   %s\n", status)
	fmt.Printf("  priority: %s\n", priority)
	fmt.Printf("  blocked:  %v\n", blocked)
	return nil
}

// --- Infrastructure helpers (mirror of cmd/hive patterns) ---

// openStore creates a Store from a shared pool.
// nil pool → in-memory. Non-nil → PostgresStore (shared pool).
func openStore(ctx context.Context, pool *pgxpool.Pool) (store.Store, error) {
	if pool == nil {
		fmt.Fprintln(os.Stderr, "Store: in-memory")
		return store.NewInMemoryStore(), nil
	}
	fmt.Fprintln(os.Stderr, "Store: postgres")
	return pgstore.NewPostgresStoreFromPool(ctx, pool)
}

// openActorStore creates an IActorStore from a shared pool.
// nil pool → in-memory. Non-nil → PostgresActorStore (shared pool).
func openActorStore(ctx context.Context, pool *pgxpool.Pool) (actor.IActorStore, error) {
	if pool == nil {
		fmt.Fprintln(os.Stderr, "Actor store: in-memory")
		return actor.NewInMemoryActorStore(), nil
	}
	fmt.Fprintln(os.Stderr, "Actor store: postgres")
	return pgactor.NewPostgresActorStoreFromPool(ctx, pool)
}

// bootstrapGraph emits the genesis event if the store is empty.
// Idempotent — does nothing if the graph already has events.
func bootstrapGraph(s store.Store, humanID types.ActorID) error {
	head, err := s.Head()
	if err != nil {
		return fmt.Errorf("check head: %w", err)
	}
	if head.IsSome() {
		return nil // already bootstrapped
	}
	fmt.Fprintln(os.Stderr, "Bootstrapping event graph...")
	registry := event.DefaultRegistry()
	bsFactory := event.NewBootstrapFactory(registry)
	signer := &bootstrapSigner{humanID: humanID}
	bootstrap, err := bsFactory.Init(humanID, signer)
	if err != nil {
		return fmt.Errorf("create genesis event: %w", err)
	}
	if _, err := s.Append(bootstrap); err != nil {
		return fmt.Errorf("append genesis event: %w", err)
	}
	fmt.Fprintln(os.Stderr, "Event graph bootstrapped.")
	return nil
}

// bootstrapSigner provides a minimal Signer for the genesis event.
type bootstrapSigner struct {
	humanID types.ActorID
}

func (b *bootstrapSigner) Sign(data []byte) (types.Signature, error) {
	h := sha256.Sum256([]byte("signer:" + b.humanID.Value()))
	priv := ed25519.NewKeyFromSeed(h[:])
	sig := ed25519.Sign(priv, data)
	return types.NewSignature(sig)
}

// registerHuman bootstraps a human operator in the actor store.
// WARNING: derives key from display name — insecure for production persistent stores.
// Mirrors cmd/hive registerHuman exactly so the same --human name produces the same ActorID.
func registerHuman(actors actor.IActorStore, displayName string) (types.ActorID, error) {
	h := sha256.Sum256([]byte("human:" + displayName))
	priv := ed25519.NewKeyFromSeed(h[:])
	pub := priv.Public().(ed25519.PublicKey)
	pk, err := types.NewPublicKey([]byte(pub))
	if err != nil {
		return types.ActorID{}, fmt.Errorf("public key: %w", err)
	}
	a, err := actors.Register(pk, displayName, event.ActorTypeHuman)
	if err != nil {
		return types.ActorID{}, err
	}
	return a.ID(), nil
}

// ed25519Signer implements event.Signer for work-emitted events.
type ed25519Signer struct {
	key ed25519.PrivateKey
}

func (s *ed25519Signer) Sign(data []byte) (types.Signature, error) {
	sig := ed25519.Sign(s.key, data)
	return types.NewSignature(sig)
}

// deriveSignerFromID creates a deterministic Ed25519 signer from an ActorID.
// Stable across restarts — the same humanID always produces the same key.
func deriveSignerFromID(id types.ActorID) *ed25519Signer {
	h := sha256.Sum256([]byte("signer:" + id.Value()))
	priv := ed25519.NewKeyFromSeed(h[:])
	return &ed25519Signer{key: priv}
}

// newConversationID generates a unique ConversationID for this CLI session.
func newConversationID() (types.ConversationID, error) {
	id, err := types.NewEventIDFromNew()
	if err != nil {
		return types.ConversationID{}, err
	}
	return types.NewConversationID("work-cli-" + id.Value())
}
