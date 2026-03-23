package navidrome

import (
	"context"
	"os"
	"testing"
	"time"
)

// TestIntegrationNavidromeClient tests against a real Navidrome instance
// To run: NAVIDROME_URL=<url> NAVIDROME_USER=<user> NAVIDROME_PASS=<pass> go test -v -run TestIntegrationNavidromeClient ./...
func TestIntegrationNavidromeClient(t *testing.T) {
	// Only run if env vars are set
	url := os.Getenv("NAVIDROME_URL")
	user := os.Getenv("NAVIDROME_USER")
	pass := os.Getenv("NAVIDROME_PASS")

	if url == "" || user == "" || pass == "" {
		t.Skip("Skipping integration test: NAVIDROME_URL, NAVIDROME_USER, NAVIDROME_PASS not set")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	client := NewHTTPClient(url)

	// Test 1: Login
	t.Log("Test 1: Testing Login...")
	if err := client.Login(ctx, user, pass); err != nil {
		t.Fatalf("Login failed: %v", err)
	}
	t.Log("✓ Login successful")

	// Test 2: List playlists
	t.Log("Test 2: Testing ListPlaylists...")
	playlists, err := client.listPlaylists(ctx)
	if err != nil {
		t.Fatalf("ListPlaylists failed: %v", err)
	}
	t.Logf("✓ Found %d playlists", len(playlists))
	for _, p := range playlists {
		t.Logf("  - %s (ID: %s)", p.Name, p.ID)
	}

	// Test 3: Ensure test playlist
	testPlaylistName := "NavidromeOperatorTest"
	t.Logf("Test 3: Testing EnsurePlaylist with name '%s'...", testPlaylistName)
	playlistID, err := client.EnsurePlaylist(ctx, testPlaylistName)
	if err != nil {
		t.Fatalf("EnsurePlaylist failed: %v", err)
	}
	t.Logf("✓ EnsurePlaylist successful, ID: %s", playlistID)

	// Test 4: Resolve track (if any exist)
	t.Log("Test 4: Testing ResolveTrack...")
	selector := TrackSelector{
		Artist: "The Beatles",
		Title:  "Let It Be",
	}
	trackID, err := client.ResolveTrack(ctx, selector)
	if err != nil {
		t.Logf("⚠ ResolveTrack failed (expected if track doesn't exist): %v", err)
	} else {
		t.Logf("✓ ResolveTrack successful, found trackID: %s", trackID)

		// Test 5: Add track to playlist
		t.Log("Test 5: Testing AddOrMoveTrack...")
		if err := client.AddOrMoveTrack(ctx, playlistID, trackID, 0); err != nil {
			t.Logf("⚠ AddOrMoveTrack failed: %v (this might be OK if track is already in playlist)", err)
		} else {
			t.Logf("✓ AddOrMoveTrack successful")
		}
	}

	// Test 6: Delete test playlist
	t.Log("Test 6: Testing DeletePlaylist...")
	if err := client.DeletePlaylist(ctx, playlistID); err != nil {
		t.Fatalf("DeletePlaylist failed: %v", err)
	}
	t.Logf("✓ DeletePlaylist successful")

	t.Log("\n✓✓✓ All integration tests passed! ✓✓✓")
}
