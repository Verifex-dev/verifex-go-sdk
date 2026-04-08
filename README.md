# Verifex Go SDK

Official Go client for the [Verifex](https://verifex.dev) sanctions screening API.

## Installation

```bash
go get github.com/Verifex-dev/verifex-go-sdk
```

## Quick Start

```go
package main

import (
	"context"
	"fmt"
	"log"

	verifex "github.com/Verifex-dev/verifex-go-sdk"
)

func main() {
	client := verifex.New("vfx_your_api_key")

	result, err := client.Screen(context.Background(), verifex.ScreenRequest{
		Name: "Vladimir Putin",
		Type: "person",
	})
	if err != nil {
		log.Fatal(err)
	}

	if result.IsClear() {
		fmt.Println("No sanctions match found")
		return
	}

	fmt.Printf("Risk: %s (%d matches)\n", result.RiskLevel, result.TotalMatches)
	for _, m := range result.Matches {
		fmt.Printf("  %s (%s) — %d%% confidence\n", m.Name, m.Source, m.Confidence)
	}
}
```

## Batch Screening

```go
batch, err := client.BatchScreen(ctx, []verifex.ScreenRequest{
	{Name: "Vladimir Putin", Type: "person"},
	{Name: "Sberbank", Type: "entity"},
})
for _, r := range batch.Results {
	fmt.Printf("%s: %s\n", r.Query.Name, r.RiskLevel)
}
```

## Error Handling

```go
result, err := client.Screen(ctx, verifex.ScreenRequest{Name: "test"})
if verifex.IsAuthError(err) {
	log.Fatal("Invalid API key")
}
if verifex.IsRateLimitError(err) {
	log.Fatal("Rate limited — slow down")
}
if verifex.IsQuotaExceededError(err) {
	log.Fatal("Monthly quota reached — upgrade plan")
}
```

## Configuration

```go
client := verifex.New("vfx_key",
	verifex.WithTimeout(10 * time.Second),
	verifex.WithBaseURL("https://custom-api.example.com"),
)
```

## API Reference

| Method | Description |
|--------|-------------|
| `Screen(ctx, req)` | Screen a single entity |
| `BatchScreen(ctx, entities)` | Screen multiple entities (Pro+) |
| `Usage(ctx)` | Get monthly usage stats |
| `Health(ctx)` | Check API health (no auth) |
| `ListKeys(ctx)` | List API keys |
| `CreateKey(ctx, name)` | Create new API key |
| `RevokeKey(ctx, id)` | Revoke an API key |

## License

MIT
