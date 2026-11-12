package images

import "testing"

func TestImageRegistryFailClosed(t *testing.T) {
	for _, endpoint := range []string{"http://example.com:8188/prompt","http://8.8.8.8:8188/prompt","http://127.0.0.1:8188/v1/chat/completions","http://127.0.0.1:8188/prompt"} {
		_, err := NewRegistry([]ProviderConfig{{ID:"local",Kind:"comfyui",Endpoint:endpoint}},[]ModelConfig{{ID:"image/local",Provider:"local",UpstreamID:"workflow",Location:"selfhosted",Operations:[]string{"images.generate"}}})
		if err == nil { t.Fatalf("ComfyUI must fail closed without native adapter: %s",endpoint) }
	}
	if _, ok := (*Registry)(nil).Resolve("unknown"); ok { t.Fatal("nil registry resolved route") }
}
