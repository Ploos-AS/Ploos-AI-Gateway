package capabilities

import "testing"

func TestCatalog(t *testing.T) {
	tests := []struct {
		op     Operation
		output Modality
		async  bool
	}{
		{Chat, Text, false},
		{ImageGeneration, Image, false},
		{Speech, Audio, false},
		{Transcription, Text, false},
		{VideoGeneration, Video, true},
	}
	for _, tt := range tests {
		d, ok := Lookup(tt.op)
		if !ok || d.Operation != tt.op || d.Output[0] != tt.output || d.Async != tt.async {
			t.Fatalf("unexpected descriptor for %q: %+v", tt.op, d)
		}
	}
	if _, ok := Lookup("unknown"); ok {
		t.Fatal("unknown operation accepted")
	}
}

func TestLookupReturnsCopy(t *testing.T) {
	d, _ := Lookup(Chat)
	d.Output[0] = Video
	again, _ := Lookup(Chat)
	if again.Output[0] != Text {
		t.Fatal("catalog mutated through returned descriptor")
	}
}

func TestModelCapabilitiesFailClosed(t *testing.T) {
	m := Model{ID: "model-a", Operations: []Operation{Chat}}
	if err := m.Validate(); err != nil {
		t.Fatal(err)
	}
	if !m.Supports(Chat) || m.Supports(ImageGeneration) {
		t.Fatal("unexpected supported operations")
	}
	for _, bad := range []Model{
		{ID: "", Operations: []Operation{Chat}},
		{ID: "x"},
		{ID: "x", Operations: []Operation{"made.up"}},
		{ID: "x", Operations: []Operation{Chat, Chat}},
	} {
		if bad.Validate() == nil {
			t.Fatalf("accepted invalid model: %+v", bad)
		}
		if bad.Supports(Chat) {
			t.Fatal("invalid model must not route")
		}
	}
}
