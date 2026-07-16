package executil

import "testing"

func TestCommandStringQuotesUnsafeArgument(t *testing.T) {
	got := commandString("git", []string{"diff", "--", "file with space.txt"})
	want := "git diff -- 'file with space.txt'"
	if got != want {
		t.Fatalf("got %q want %q", got, want)
	}
}
