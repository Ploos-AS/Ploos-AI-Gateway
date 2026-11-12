package images

import "testing"

func TestImageRegistryFailClosed(t *testing.T) {
	p := []ProviderConfig{{ID:"local",Kind:"comfyui",Endpoint:"http://127.0.0.1:8188/prompt"}}
	m := []ModelConfig{{ID:"image/local",Provider:"local",UpstreamID:"workflow",Location:"selfhosted",Operations:[]string{"images.generate"}}}
	r, err := NewRegistry(p,m)
	if err != nil { t.Fatal(err) }
	if _, ok := r.Resolve("image/local"); !ok { t.Fatal("missing route") }
	if _, ok := r.Resolve("image/unknown"); ok { t.Fatal("unexpected fallback") }
	for _, endpoint := range []string{"http://example.com:8188/prompt","http://8.8.8.8:8188/prompt","http://127.0.0.1:8188/v1/chat/completions"} {
		if _, err := NewRegistry([]ProviderConfig{{ID:"local",Kind:"comfyui",Endpoint:endpoint}},m); err == nil { t.Fatalf("accepted %s",endpoint) }
	}
	m[0].Operations = []string{"chat.completions"}
	if _, err := NewRegistry(p,m); err == nil { t.Fatal("accepted chat as image capability") }
}
