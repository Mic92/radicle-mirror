package main

import (
	"reflect"
	"testing"
)

func TestMissingDelegates(t *testing.T) {
	current := `did:key:z6Mkm3QV7chqdCrksAYoJDWpwLg1yoXbU52NZFgWq13bv3gp
did:key:z6MkjE3BSJn4Y129rhqi5rViSUru8KSBcCQdQcDZq1cnjumw
`
	wanted := []string{
		"did:key:z6MkjE3BSJn4Y129rhqi5rViSUru8KSBcCQdQcDZq1cnjumw",
		"did:key:z6MkhmR8XjssbtLcNv5RK5eAJGKc4S3S8R7yUGZKmNyeXPuw",
	}
	got := missingDelegates(current, wanted)
	want := []string{"did:key:z6MkhmR8XjssbtLcNv5RK5eAJGKc4S3S8R7yUGZKmNyeXPuw"}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("missingDelegates = %v, want %v", got, want)
	}
	if got := missingDelegates(current, wanted[:1]); len(got) != 0 {
		t.Errorf("expected no missing delegates, got %v", got)
	}
}
